package stashsync

import (
	"encoding/base64"
	"errors"
	"slices"
	"strings"
	"time"

	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

// These defaults and precedence follow Stash v0.31.1 internal/identify.
// In particular createMissing belongs to the selected field rule, source
// rules replace whole default rules, and an explicit empty gender list matters.
type codeFieldRule struct {
	Strategy      gql.IdentifyFieldStrategy
	CreateMissing bool
}
type codeIdentifyOptions struct {
	Fields          map[string]codeFieldRule
	Cover           bool
	Organized       bool
	IncludeMale     bool
	Genders         []gql.GenderEnum
	SkipSingleName  bool
	SkipSingleTag   string
	SkipMultiple    bool
	SkipMultipleTag string
}

func effectiveCodeOptions(defaults, source *gql.IdentifyMetadataOptionsInput) codeIdentifyOptions {
	out := codeIdentifyOptions{Fields: map[string]codeFieldRule{}, Cover: true, IncludeMale: true}
	for _, options := range []*gql.IdentifyMetadataOptionsInput{defaults, source} {
		if options == nil {
			continue
		}
		if options.SetCoverImage != nil {
			out.Cover = *options.SetCoverImage
		}
		if options.SetOrganized != nil {
			out.Organized = *options.SetOrganized
		}
		if options.IncludeMalePerformers != nil {
			out.IncludeMale = *options.IncludeMalePerformers
		}
		if options.PerformerGenders != nil {
			out.Genders = append([]gql.GenderEnum{}, options.PerformerGenders...)
		}
		if options.SkipSingleNamePerformers != nil {
			out.SkipSingleName = *options.SkipSingleNamePerformers
		}
		if options.SkipSingleNamePerformerTag != nil && *options.SkipSingleNamePerformerTag != "" {
			out.SkipSingleTag = *options.SkipSingleNamePerformerTag
		}
		if options.SkipMultipleMatches != nil {
			out.SkipMultiple = *options.SkipMultipleMatches
		}
		if options.SkipMultipleMatchTag != nil && *options.SkipMultipleMatchTag != "" {
			out.SkipMultipleTag = *options.SkipMultipleMatchTag
		}
	}
	// First matching rule wins, exactly as native getFieldOptions.
	for _, options := range []*gql.IdentifyMetadataOptionsInput{source, defaults} {
		if options == nil {
			continue
		}
		for _, field := range options.FieldOptions {
			if field == nil {
				continue
			}
			if _, found := out.Fields[field.Field]; found {
				continue
			}
			strategy := field.Strategy
			if !strategy.IsValid() {
				strategy = gql.IdentifyFieldStrategyMerge
			}
			out.Fields[field.Field] = codeFieldRule{strategy, field.CreateMissing != nil && *field.CreateMissing}
		}
	}
	return out
}
func (o codeIdentifyOptions) rule(field string) codeFieldRule {
	if f, found := o.Fields[field]; found {
		return f
	}
	return codeFieldRule{Strategy: gql.IdentifyFieldStrategyMerge}
}
func (o codeIdentifyOptions) acceptsGender(p *gql.ScrapedPerformer) bool {
	if p.Gender == nil {
		return true
	}
	gender := gql.GenderEnum(strings.ToUpper(*p.Gender))
	if o.Genders != nil {
		return slices.Contains(o.Genders, gender)
	}
	return o.IncludeMale || gender != gql.GenderEnumMale
}
func identifySettingsDigest(defaults *gql.IdentifyMetadataInput) string {
	effective := []codeIdentifyOptions{}
	if defaults != nil {
		effective = append(effective, effectiveCodeOptions(defaults.Options, nil))
		for _, source := range defaults.Sources {
			if source != nil {
				effective = append(effective, effectiveCodeOptions(defaults.Options, source.Options))
			}
		}
	}
	return metadataDigest(struct {
		Input     *gql.IdentifyMetadataInput
		Effective []codeIdentifyOptions
	}{defaults, effective})
}
func validScrapedDate(value *string) *string {
	if value == nil {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02 15:04:05", "2006-01", "2006"} {
		if parsed, err := time.Parse(layout, *value); err == nil {
			formatted := parsed.Format("2006-01-02")
			if layout == "2006-01" || layout == "2006" {
				formatted = parsed.Format(layout)
			}
			return &formatted
		}
	}
	return nil
}
func coverDigest(image *string) (string, error) {
	if image == nil || *image == "" {
		return "", nil
	}
	raw := *image
	comma := strings.Index(raw, ";base64,")
	if !strings.HasPrefix(raw, "data:") || comma < 0 {
		return "", errors.New("Stash did not return an encoded scraped cover")
	}
	raw = raw[comma+8:]
	if len(raw) > 12*1024*1024 {
		return "", errors.New("scraped cover exceeds image size limit")
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", errors.New("Stash returned an invalid scraped cover")
	}
	if len(decoded) > 8*1024*1024 {
		return "", errors.New("scraped cover exceeds image size limit")
	}
	return bytesDigest(decoded), nil
}
func codeSceneInput(scene IngestScene, match *gql.ScrapedScene, options codeIdentifyOptions, endpoint string, updated time.Time, skipSingle bool) gql.SceneUpdateInput {
	input := gql.SceneUpdateInput{ID: scene.ID}
	set := func(field, old string, value *string) *string {
		rule := options.rule(field)
		if value == nil || *value == old || rule.Strategy == gql.IdentifyFieldStrategyIgnore {
			return nil
		}
		if old == "" || rule.Strategy == gql.IdentifyFieldStrategyOverwrite {
			return value
		}
		return nil
	}
	input.Title, input.Code, input.Date = set("title", scene.Title, match.Title), set("code", scene.Code, match.Code), set("date", scene.Date, validScrapedDate(match.Date))
	input.Details, input.Director = set("details", scene.Details, match.Details), set("director", scene.Director, match.Director)
	if rule := options.rule("studio"); rule.Strategy != gql.IdentifyFieldStrategyIgnore && (scene.StudioID == "" || rule.Strategy == gql.IdentifyFieldStrategyOverwrite) && match.Studio != nil && match.Studio.StoredID != nil && *match.Studio.StoredID != scene.StudioID {
		input.StudioID = match.Studio.StoredID
	}
	if rule := options.rule("performers"); rule.Strategy != gql.IdentifyFieldStrategyIgnore && len(match.Performers) > 0 {
		var ids []string
		if rule.Strategy == gql.IdentifyFieldStrategyMerge {
			ids = append(ids, scene.PerformerIDs...)
		}
		for _, p := range match.Performers {
			if p != nil && options.acceptsGender(p) && p.StoredID != nil && !slices.Contains(ids, *p.StoredID) {
				ids = append(ids, *p.StoredID)
			}
		}
		if ids != nil && !sameStringSet(ids, scene.PerformerIDs) {
			input.PerformerIds = ids
		}
	}
	if rule := options.rule("tags"); rule.Strategy != gql.IdentifyFieldStrategyIgnore && len(match.Tags) > 0 {
		var ids []string
		if rule.Strategy == gql.IdentifyFieldStrategyMerge {
			ids = append(ids, scene.TagIDs...)
		}
		for _, tag := range match.Tags {
			if tag != nil && tag.StoredID != nil && !slices.Contains(ids, *tag.StoredID) {
				ids = append(ids, *tag.StoredID)
			}
		}
		if ids != nil && !sameStringSet(ids, scene.TagIDs) {
			input.TagIds = ids
		}
	}
	if skipSingle && options.SkipSingleTag != "" {
		if input.TagIds == nil {
			input.TagIds = []string{}
		}
		if !slices.Contains(input.TagIds, options.SkipSingleTag) {
			input.TagIds = append(input.TagIds, options.SkipSingleTag)
		}
	}
	if rule := options.rule("url"); rule.Strategy != gql.IdentifyFieldStrategyIgnore && len(match.Urls) > 0 {
		var ids []string
		if rule.Strategy == gql.IdentifyFieldStrategyMerge {
			ids = append(ids, scene.URLs...)
		}
		for _, u := range match.Urls {
			if !slices.Contains(ids, u) {
				ids = append(ids, u)
			}
		}
		// Native URL overwrite only updates when the incoming set adds a URL.
		if rule.Strategy == gql.IdentifyFieldStrategyMerge && len(ids) != len(scene.URLs) || rule.Strategy == gql.IdentifyFieldStrategyOverwrite && hasNewString(match.Urls, scene.URLs) {
			input.Urls = ids
		}
	}
	if options.Cover && match.Image != nil && *match.Image != "" {
		input.CoverImage = match.Image
	}
	if options.Organized && !scene.Organized {
		yes := true
		input.Organized = &yes
	}
	if rule := options.rule("stash_ids"); rule.Strategy != gql.IdentifyFieldStrategyIgnore && match.RemoteSiteID != nil && endpoint != "" {
		if rule.Strategy == gql.IdentifyFieldStrategyMerge {
			input.StashIds = append([]*gql.StashIDInput{}, scene.StashIDs...)
		} else {
			input.StashIds = []*gql.StashIDInput{}
		}
		found := false
		for i, ref := range input.StashIds {
			if ref != nil && ref.Endpoint == endpoint {
				input.StashIds[i] = &gql.StashIDInput{Endpoint: endpoint, StashID: *match.RemoteSiteID, UpdatedAt: &updated}
				found = true
				break
			}
		}
		if !found {
			input.StashIds = append(input.StashIds, &gql.StashIDInput{Endpoint: endpoint, StashID: *match.RemoteSiteID, UpdatedAt: &updated})
		}
	}
	return input
}
func sameStringSet(a, b []string) bool {
	aa, bb := append([]string{}, a...), append([]string{}, b...)
	slices.Sort(aa)
	slices.Sort(bb)
	return slices.Equal(aa, bb)
}
func hasNewString(a, b []string) bool {
	for _, s := range a {
		if !slices.Contains(b, s) {
			return true
		}
	}
	return false
}
