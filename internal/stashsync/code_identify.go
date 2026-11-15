package stashsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/leothevan2444/moji/internal/film"
	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

var (
	ErrCodeNoMatch     = errors.New("no exact code match was found in the configured StashBox sources")
	ErrCodeAmbiguous   = errors.New("more than one exact code match was found; review the candidates in Stash")
	ErrCodeRelations   = errors.New("Stash could not resolve or create the scraped studio, performers, or tags")
	ErrCodePlanChanged = errors.New("Stash settings or scene metadata changed; review the identification in Stash")
)

const MaxCodeIdentifyPlanBytes = 32 * 1024 * 1024

type codeIdentifyClient interface {
	IngestClient
	IngestIdentity() string
	ScrapeIngestScene(context.Context, gql.ScraperSourceInput, gql.ScrapeSingleSceneInput, bool) ([]*gql.ScrapedScene, error)
	SaveIngestScene(context.Context, string, gql.SceneUpdateInput) error
}
type codeCoverClient interface {
	IngestCoverDigest(context.Context, string, string) (string, error)
}

// Version 2 stores the selected remote payload once, including its cover and
// effective native settings. All entity/scene writes happen after journaling.
// Version 1 inputs remain readable only to upgrade the same selected identity.
type CodeIdentifyPlan struct {
	Version     int                 `json:"version"`
	Code        string              `json:"code"`
	Identity    string              `json:"identity"`
	Settings    string              `json:"settings"`
	Endpoint    string              `json:"endpoint"`
	RemoteID    string              `json:"remote_id"`
	CreatedAt   time.Time           `json:"created_at,omitempty"`
	Options     codeIdentifyOptions `json:"options"`
	Selected    *gql.ScrapedScene   `json:"selected,omitempty"`
	BlockReason string              `json:"block_reason,omitempty"`
	Scenes      []CodeSceneSave     `json:"scenes"`
}
type CodeSceneSave struct {
	Before      string               `json:"before"`
	Input       gql.SceneUpdateInput `json:"input"`
	Snapshot    *IngestScene         `json:"snapshot,omitempty"`
	BeforeCover string               `json:"before_cover,omitempty"`
}

func bytesDigest(raw []byte) string   { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func metadataDigest(value any) string { raw, _ := json.Marshal(value); return bytesDigest(raw) }
func sceneMetadataDigest(scene IngestScene) string {
	scene.ImageURL, scene.StudioName, scene.PerformerNames = "", "", nil
	return metadataDigest(scene)
}
func exactScrapedCode(value, code string) bool {
	return film.Code(value) != "" && film.Code(value) == film.Code(code)
}

func (s *Service) PrepareFilmIdentify(ctx context.Context, target film.Snapshot, path string, ids []string) (*CodeIdentifyPlan, error) {
	if target.Validate() != nil {
		return nil, film.ErrRequired
	}
	return s.prepareCodeIdentify(ctx, target.Code, path, ids, &target)
}
func (s *Service) PrepareCodeIdentify(ctx context.Context, code, path string, ids []string) (*CodeIdentifyPlan, error) {
	return s.prepareCodeIdentify(ctx, code, path, ids, nil)
}
func (s *Service) prepareCodeIdentify(ctx context.Context, code, path string, ids []string, target *film.Snapshot) (*CodeIdentifyPlan, error) {
	c, ok := s.client.(codeIdentifyClient)
	if !ok {
		return nil, nil
	}
	code = film.Code(code)
	if code == "" || len(ids) == 0 || len(ids) > 16 {
		return nil, ErrCodeNoMatch
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	identity := c.IngestIdentity()
	if identity == "" {
		return nil, ErrCodePlanChanged
	}
	defaults, e := c.IngestIdentifyDefaults(ctx)
	if e != nil {
		return nil, e
	}
	if defaults == nil || len(defaults.Sources) == 0 || len(defaults.Sources) > 8 {
		return nil, ErrIdentifySourcesMissing
	}
	scenes, e := s.FindIngestScenes(ctx, path, nil)
	if e != nil {
		return nil, e
	}
	byID := map[string]IngestScene{}
	for _, scene := range scenes {
		byID[scene.ID] = scene
	}
	for _, id := range ids {
		scene, found := byID[id]
		if !found {
			return nil, ErrCodePlanChanged
		}
		if len(scene.StashIDs) > 0 || (scene.Code != "" && film.Code(scene.Code) != film.Code(code)) {
			return nil, nil
		}
	}
	// Check every configured source before interpreting identification as a zero
	// match. A failed lookup or an existing fingerprint/fragment match never falls
	// through to a different identity.
	for _, source := range defaults.Sources {
		if source == nil || source.Source == nil {
			continue
		}
		for _, id := range ids {
			matches, e := c.ScrapeIngestScene(ctx, *source.Source, gql.ScrapeSingleSceneInput{SceneID: &id}, false)
			if e != nil {
				return nil, e
			}
			if len(matches) > 0 {
				return nil, nil
			}
		}
	}
	for _, source := range defaults.Sources {
		if source == nil || source.Source == nil || source.Source.StashBoxEndpoint == nil || *source.Source.StashBoxEndpoint == "" {
			continue
		}
		if target != nil && film.Endpoint(*source.Source.StashBoxEndpoint) != film.Endpoint(target.Endpoint) {
			continue
		}
		options := effectiveCodeOptions(defaults.Options, source.Options)
		matches, e := c.ScrapeIngestScene(ctx, *source.Source, gql.ScrapeSingleSceneInput{Query: &code}, options.Cover)
		if e != nil {
			return nil, e
		}
		exact := []*gql.ScrapedScene{}
		seen := map[string]bool{}
		for _, m := range matches {
			if target != nil && (m == nil || valueOrEmpty(m.RemoteSiteID) != target.SceneID) {
				continue
			}
			if m == nil || m.Code == nil || !exactScrapedCode(*m.Code, code) || valueOrEmpty(m.RemoteSiteID) == "" {
				continue
			}
			if !seen[*m.RemoteSiteID] {
				seen[*m.RemoteSiteID] = true
				exact = append(exact, m)
			}
		}
		if len(exact) == 0 {
			continue
		}
		if len(exact) > 1 {
			// The unique-code admission gate remains stricter than Identify's optional
			// first-result choice. Honor its configured skip tag without guessing.
			if options.SkipMultiple && options.SkipMultipleTag != "" {
				plan, e := s.newCodePlan(ctx, c, identity, defaults, source, code, ids, byID, nil)
				if e != nil {
					return nil, e
				}
				plan.BlockReason = "multiple"
				return plan, nil
			}
			return nil, ErrCodeAmbiguous
		}
		return s.newCodePlan(ctx, c, identity, defaults, source, code, ids, byID, exact[0])
	}
	return nil, ErrCodeNoMatch
}
func (s *Service) newCodePlan(ctx context.Context, c codeIdentifyClient, identity string, defaults *gql.IdentifyMetadataInput, source *gql.IdentifySourceInput, code string, ids []string, byID map[string]IngestScene, selected *gql.ScrapedScene) (*CodeIdentifyPlan, error) {
	if c.IngestIdentity() != identity {
		return nil, ErrCodePlanChanged
	}
	plan := &CodeIdentifyPlan{Version: 2, Code: code, Identity: identity, Settings: identifySettingsDigest(defaults), Endpoint: *source.Source.StashBoxEndpoint, CreatedAt: time.Now().UTC(), Options: effectiveCodeOptions(defaults.Options, source.Options), Selected: selected}
	if selected != nil {
		plan.RemoteID = valueOrEmpty(selected.RemoteSiteID)
		if len(selected.Performers) > 128 || len(selected.Tags) > 512 {
			return nil, errors.New("scraped entity count exceeds the identification budget")
		}
	}
	wanted := ""
	var e error
	if selected != nil && plan.Options.Cover {
		wanted, e = coverDigest(selected.Image)
		if e != nil {
			return nil, e
		}
	}
	for _, id := range ids {
		scene, found := byID[id]
		if !found {
			return nil, ErrCodePlanChanged
		}
		save := CodeSceneSave{Before: sceneMetadataDigest(scene), Input: gql.SceneUpdateInput{ID: id}, Snapshot: &scene}
		if wanted != "" {
			reader, ok := c.(codeCoverClient)
			if !ok {
				return nil, errors.New("Stash cover verification is unavailable")
			}
			save.BeforeCover, e = reader.IngestCoverDigest(ctx, identity, id)
			if e != nil {
				return nil, e
			}
		}
		plan.Scenes = append(plan.Scenes, save)
	}
	raw, e := json.Marshal(plan)
	if e != nil {
		return nil, e
	}
	if len(raw) > MaxCodeIdentifyPlanBytes {
		return nil, errors.New("Stash code identification plan exceeds the size limit")
	}
	return plan, nil
}

// Upgrade an unfinished old plan by fetching its exact selected remote ID, not
// by running a new keyword search. The new plan is journaled before any writes.
func (s *Service) UpgradeCodeIdentify(ctx context.Context, path string, ids []string, old *CodeIdentifyPlan) (*CodeIdentifyPlan, error) {
	c, ok := s.client.(codeIdentifyClient)
	if !ok || old == nil || old.Version != 1 || c.IngestIdentity() != old.Identity || old.RemoteID == "" {
		return nil, ErrCodePlanChanged
	}
	defaults, e := c.IngestIdentifyDefaults(ctx)
	if e != nil {
		return nil, e
	}
	if defaults == nil {
		return nil, ErrCodePlanChanged
	}
	var source *gql.IdentifySourceInput
	for _, v := range defaults.Sources {
		if v != nil && v.Source != nil && v.Source.StashBoxEndpoint != nil && *v.Source.StashBoxEndpoint == old.Endpoint {
			source = v
			break
		}
	}
	if source == nil {
		return nil, ErrCodePlanChanged
	}
	matches, e := c.ScrapeIngestScene(ctx, *source.Source, gql.ScrapeSingleSceneInput{Query: &old.RemoteID}, effectiveCodeOptions(defaults.Options, source.Options).Cover)
	if e != nil {
		return nil, e
	}
	var selected *gql.ScrapedScene
	for _, m := range matches {
		if m != nil && valueOrEmpty(m.RemoteSiteID) == old.RemoteID && exactScrapedCode(valueOrEmpty(m.Code), old.Code) {
			if selected != nil {
				return nil, ErrCodeAmbiguous
			}
			selected = m
		}
	}
	if selected == nil {
		return nil, ErrCodePlanChanged
	}
	scenes, e := s.FindIngestScenes(ctx, path, nil)
	if e != nil {
		return nil, e
	}
	byID := map[string]IngestScene{}
	for _, scene := range scenes {
		byID[scene.ID] = scene
	}
	for _, save := range old.Scenes {
		if !slices.Contains(ids, save.Input.ID) {
			return nil, ErrCodePlanChanged
		}
	}
	return s.newCodePlan(ctx, c, old.Identity, defaults, source, old.Code, ids, byID, selected)
}
func (s *Service) ApplyCodeIdentify(ctx context.Context, path string, ids []string, plan *CodeIdentifyPlan) error {
	c, ok := s.client.(codeIdentifyClient)
	if !ok || plan == nil || plan.Version != 2 || plan.Identity == "" || plan.Identity != c.IngestIdentity() || len(plan.Scenes) != len(ids) {
		return ErrCodePlanChanged
	}
	defaults, e := c.IngestIdentifyDefaults(ctx)
	if e != nil {
		return e
	}
	if identifySettingsDigest(defaults) != plan.Settings {
		return ErrCodePlanChanged
	}
	seen := map[string]bool{}
	for _, save := range plan.Scenes {
		if !slices.Contains(ids, save.Input.ID) || seen[save.Input.ID] || save.Snapshot == nil || save.Snapshot.ID != save.Input.ID {
			return ErrCodePlanChanged
		}
		seen[save.Input.ID] = true
		current, e := s.codeCurrentScene(ctx, path, save.Input.ID)
		if e != nil {
			return e
		}
		input := gql.SceneUpdateInput{ID: current.ID}
		if plan.BlockReason == "multiple" {
			if slices.Contains(current.TagIDs, plan.Options.SkipMultipleTag) {
				continue
			}
			if sceneMetadataDigest(*current) != save.Before {
				return ErrCodePlanChanged
			}
			input.TagIds = append(append([]string{}, current.TagIDs...), plan.Options.SkipMultipleTag)
		} else {
			if plan.Selected == nil || !exactScrapedCode(valueOrEmpty(plan.Selected.Code), plan.Code) || valueOrEmpty(plan.Selected.RemoteSiteID) != plan.RemoteID {
				return ErrCodePlanChanged
			}
			raw, _ := json.Marshal(plan.Selected)
			var selected gql.ScrapedScene
			if e = json.Unmarshal(raw, &selected); e != nil {
				return e
			}
			allowCreate := sceneMetadataDigest(*current) == save.Before
			skipped, e := resolveCodeEntities(ctx, c, plan, *save.Snapshot, &selected, allowCreate)
			if e != nil {
				return e
			}
			input = codeSceneInput(*save.Snapshot, &selected, plan.Options, plan.Endpoint, plan.CreatedAt, skipped)
		}
		wanted := ""
		coverCurrent := ""
		if input.CoverImage != nil {
			wanted, e = coverDigest(input.CoverImage)
			if e != nil {
				return e
			}
			reader, ok := c.(codeCoverClient)
			if !ok {
				return errors.New("Stash cover verification is unavailable")
			}
			coverCurrent, e = reader.IngestCoverDigest(ctx, plan.Identity, current.ID)
			if e != nil {
				return e
			}
		}
		if sceneMatchesInput(*current, input) && (wanted == "" || wanted == coverCurrent) {
			continue
		}
		if sceneMetadataDigest(*current) != save.Before {
			return ErrCodePlanChanged
		}
		if wanted != "" && coverCurrent != wanted && coverCurrent != save.BeforeCover {
			return ErrCodePlanChanged
		}
		if wanted != "" && coverCurrent == wanted {
			input.CoverImage = nil
		}
		if plan.Identity != c.IngestIdentity() {
			return ErrCodePlanChanged
		}
		if e = c.SaveIngestScene(ctx, plan.Identity, input); e != nil {
			return fmt.Errorf("Stash code identification save was not confirmed: %w", e)
		}
		observed, e := s.codeCurrentScene(ctx, path, current.ID)
		if e != nil {
			return e
		}
		if !sceneMatchesInput(*observed, input) {
			return errors.New("Stash did not retain all requested identification fields")
		}
		if wanted != "" {
			digest, e := c.(codeCoverClient).IngestCoverDigest(ctx, plan.Identity, current.ID)
			if e != nil {
				return e
			}
			if digest != wanted {
				return errors.New("Stash did not retain the scraped cover")
			}
		}
	}
	return nil
}
func (s *Service) codeCurrentScene(ctx context.Context, path, id string) (*IngestScene, error) {
	scenes, e := s.FindIngestScenes(ctx, path, nil)
	if e != nil {
		return nil, e
	}
	for i := range scenes {
		if scenes[i].ID == id {
			return &scenes[i], nil
		}
	}
	return nil, ErrCodePlanChanged
}
func sceneMatchesInput(scene IngestScene, in gql.SceneUpdateInput) bool {
	for _, field := range []struct {
		want  *string
		value string
	}{{in.Title, scene.Title}, {in.Code, scene.Code}, {in.Date, scene.Date}, {in.Details, scene.Details}, {in.Director, scene.Director}, {in.StudioID, scene.StudioID}} {
		if field.want != nil && *field.want != field.value {
			return false
		}
	}
	if in.Organized != nil && *in.Organized != scene.Organized {
		return false
	}
	if in.PerformerIds != nil && !sameStringSet(in.PerformerIds, scene.PerformerIDs) {
		return false
	}
	if in.TagIds != nil && !sameStringSet(in.TagIds, scene.TagIDs) {
		return false
	}
	if in.Urls != nil && !sameStringSet(in.Urls, scene.URLs) {
		return false
	}
	if in.StashIds != nil {
		pairs := func(refs []*gql.StashIDInput) []string {
			out := []string{}
			for _, v := range refs {
				if v != nil {
					out = append(out, v.Endpoint+"\x00"+v.StashID)
				}
			}
			return out
		}
		if !sameStringSet(pairs(in.StashIds), pairs(scene.StashIDs)) {
			return false
		}
	}
	return true
}
