package stashsync

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type codeEntityClient interface {
	FindIngestEntity(context.Context, string, string, string) (string, error)
	CreateIngestStudio(context.Context, string, gql.StudioCreateInput) (string, error)
	UpdateIngestStudio(context.Context, string, gql.StudioUpdateInput) error
	IngestStudioStashIDs(context.Context, string) ([]*gql.StashIDInput, error)
	CreateIngestPerformer(context.Context, string, gql.PerformerCreateInput) (string, error)
	CreateIngestTag(context.Context, string, gql.TagCreateInput) (string, error)
}

func scrapedRefs(endpoint string, remote *string, now time.Time) []*gql.StashIDInput {
	if remote == nil || *remote == "" || endpoint == "" {
		return nil
	}
	return []*gql.StashIDInput{{Endpoint: endpoint, StashID: *remote, UpdatedAt: &now}}
}
func splitAliases(value *string) []string {
	out := []string{}
	if value != nil {
		for _, v := range strings.Split(*value, ",") {
			if v = strings.TrimSpace(v); v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}
func studioCreateInput(s *gql.ScrapedStudio, endpoint string, now time.Time) gql.StudioCreateInput {
	in := gql.StudioCreateInput{Name: strings.TrimSpace(s.Name), Details: s.Details, Aliases: splitAliases(s.Aliases), StashIds: scrapedRefs(endpoint, s.RemoteSiteID, now), Image: s.Image}
	in.Urls = append([]string{}, s.Urls...)
	if len(in.Urls) == 0 && s.URL != nil {
		in.Urls = append(in.Urls, *s.URL)
	}
	if s.Parent != nil {
		in.ParentID = s.Parent.StoredID
	}
	return in
}
func performerCreateInput(p *gql.ScrapedPerformer, endpoint string, now time.Time) gql.PerformerCreateInput {
	in := gql.PerformerCreateInput{Name: strings.TrimSpace(valueOrEmpty(p.Name)), Disambiguation: p.Disambiguation, Country: p.Country, Ethnicity: p.Ethnicity, EyeColor: p.EyeColor, HairColor: p.HairColor, FakeTits: p.FakeTits, Measurements: p.Measurements, Details: p.Details, Piercings: p.Piercings, Tattoos: p.Tattoos, AliasList: splitAliases(p.Aliases), StashIds: scrapedRefs(endpoint, p.RemoteSiteID, now)}
	in.Birthdate, in.DeathDate = validScrapedDate(p.Birthdate), validScrapedDate(p.DeathDate)
	in.CareerStart, in.CareerEnd = validScrapedDate(p.CareerStart), validScrapedDate(p.CareerEnd)
	if in.CareerStart == nil && in.CareerEnd == nil {
		in.CareerLength = p.CareerLength
	}
	if p.Gender != nil {
		g := gql.GenderEnum(*p.Gender)
		if g.IsValid() {
			in.Gender = &g
		}
	}
	if p.Circumcised != nil {
		v := gql.CircumcisedEnum(*p.Circumcised)
		if v.IsValid() {
			in.Circumcised = &v
		}
	}
	if p.Height != nil {
		if v, e := strconv.Atoi(*p.Height); e == nil {
			in.HeightCm = &v
		}
	}
	if p.Weight != nil {
		if v, e := strconv.Atoi(*p.Weight); e == nil {
			in.Weight = &v
		}
	}
	if p.PenisLength != nil {
		if v, e := strconv.ParseFloat(*p.PenisLength, 64); e == nil {
			in.PenisLength = &v
		}
	}
	in.Urls = append([]string{}, p.Urls...)
	if len(in.Urls) == 0 {
		for _, u := range []*string{p.URL, p.Twitter, p.Instagram} {
			if u != nil {
				in.Urls = append(in.Urls, *u)
			}
		}
	}
	if len(p.Images) > 0 {
		in.Image = &p.Images[0]
	}
	return in
}

func resolveCodeEntities(ctx context.Context, client codeIdentifyClient, plan *CodeIdentifyPlan, scene IngestScene, match *gql.ScrapedScene, allowCreate bool) (bool, error) {
	var api codeEntityClient
	getAPI := func() (codeEntityClient, error) {
		if api != nil {
			return api, nil
		}
		var ok bool
		api, ok = client.(codeEntityClient)
		if !ok {
			return nil, ErrCodeRelations
		}
		return api, nil
	}
	options := plan.Options
	if rule := options.rule("studio"); rule.Strategy != gql.IdentifyFieldStrategyIgnore && (scene.StudioID == "" || rule.Strategy == gql.IdentifyFieldStrategyOverwrite) && match.Studio != nil && match.Studio.StoredID == nil && rule.CreateMissing {
		c, e := getAPI()
		if e != nil {
			return false, e
		}
		id, e := resolveCodeStudio(ctx, c, plan, match.Studio, allowCreate)
		if e != nil {
			return false, e
		}
		match.Studio.StoredID = &id
	}
	skipped := false
	if rule := options.rule("performers"); rule.Strategy != gql.IdentifyFieldStrategyIgnore && rule.CreateMissing {
		for _, p := range match.Performers {
			if p == nil || p.StoredID != nil || p.Name == nil || !options.acceptsGender(p) {
				continue
			}
			if options.SkipSingleName && !strings.Contains(*p.Name, " ") && valueOrEmpty(p.Disambiguation) == "" {
				skipped = true
				continue
			}
			c, e := getAPI()
			if e != nil {
				return false, e
			}
			if valueOrEmpty(p.RemoteSiteID) == "" {
				return false, ErrCodeRelations
			}
			id, e := c.FindIngestEntity(ctx, "performer", plan.Endpoint, *p.RemoteSiteID)
			if e != nil {
				return false, e
			}
			if id == "" && !allowCreate {
				return false, ErrCodePlanChanged
			}
			if id == "" {
				id, e = c.CreateIngestPerformer(ctx, plan.Identity, performerCreateInput(p, plan.Endpoint, plan.CreatedAt))
				if e != nil {
					return false, fmt.Errorf("%w: performer creation failed: %v", ErrCodeRelations, e)
				}
			}
			if id == "" {
				return false, ErrCodeRelations
			}
			p.StoredID = &id
		}
	}
	if rule := options.rule("tags"); rule.Strategy != gql.IdentifyFieldStrategyIgnore && rule.CreateMissing {
		for _, tag := range match.Tags {
			if tag == nil || tag.StoredID != nil {
				continue
			}
			c, e := getAPI()
			if e != nil {
				return false, e
			}
			if valueOrEmpty(tag.RemoteSiteID) == "" {
				return false, ErrCodeRelations
			}
			id, e := c.FindIngestEntity(ctx, "tag", plan.Endpoint, *tag.RemoteSiteID)
			if e != nil {
				return false, e
			}
			if id == "" && !allowCreate {
				return false, ErrCodePlanChanged
			}
			if id == "" {
				input := gql.TagCreateInput{Name: tag.Name, Description: tag.Description, Aliases: tag.AliasList, StashIds: scrapedRefs(plan.Endpoint, tag.RemoteSiteID, plan.CreatedAt)}
				if tag.Parent != nil && tag.Parent.StoredID != nil {
					input.ParentIds = []string{*tag.Parent.StoredID}
				}
				id, e = c.CreateIngestTag(ctx, plan.Identity, input)
				if e != nil {
					return false, fmt.Errorf("%w: tag creation failed: %v", ErrCodeRelations, e)
				}
			}
			if id == "" {
				return false, ErrCodeRelations
			}
			tag.StoredID = &id
		}
	}
	return skipped, nil
}
func resolveCodeStudio(ctx context.Context, c codeEntityClient, plan *CodeIdentifyPlan, s *gql.ScrapedStudio, allowCreate bool) (string, error) {
	if valueOrEmpty(s.RemoteSiteID) == "" {
		return "", ErrCodeRelations
	}
	id, e := c.FindIngestEntity(ctx, "studio", plan.Endpoint, *s.RemoteSiteID)
	if e != nil || id != "" {
		return id, e
	}
	if !allowCreate {
		return "", ErrCodePlanChanged
	}
	if s.Parent != nil {
		parent := s.Parent
		if parent.StoredID == nil {
			if valueOrEmpty(parent.RemoteSiteID) == "" {
				return "", ErrCodeRelations
			}
			parentID, e := c.FindIngestEntity(ctx, "studio", plan.Endpoint, *parent.RemoteSiteID)
			if e != nil {
				return "", e
			}
			if parentID == "" {
				parentID, e = c.CreateIngestStudio(ctx, plan.Identity, studioCreateInput(parent, plan.Endpoint, plan.CreatedAt))
				if e != nil {
					return "", studioSaveError(e)
				}
			}
			if parentID == "" {
				return "", ErrCodeRelations
			}
			parent.StoredID = &parentID
		} else {
			refs, e := c.IngestStudioStashIDs(ctx, *parent.StoredID)
			if e != nil {
				return "", e
			}
			for _, ref := range scrapedRefs(plan.Endpoint, parent.RemoteSiteID, plan.CreatedAt) {
				replaced := false
				for i, existing := range refs {
					if existing.Endpoint == ref.Endpoint {
						refs[i] = ref
						replaced = true
						break
					}
				}
				if !replaced {
					refs = append(refs, ref)
				}
			}
			input := gql.StudioUpdateInput{ID: *parent.StoredID, Name: &parent.Name, Details: parent.Details, Image: parent.Image, StashIds: refs}
			if len(parent.Urls) > 0 {
				input.Urls = parent.Urls
			} else if parent.URL != nil {
				input.Urls = []string{*parent.URL}
			}
			if valueOrEmpty(parent.Aliases) != "" {
				input.Aliases = splitAliases(parent.Aliases)
			}
			if parent.Parent != nil {
				input.ParentID = parent.Parent.StoredID
			}
			if e = c.UpdateIngestStudio(ctx, plan.Identity, input); e != nil {
				return "", studioSaveError(e)
			}
		}
	}
	id, e = c.CreateIngestStudio(ctx, plan.Identity, studioCreateInput(s, plan.Endpoint, plan.CreatedAt))
	if e != nil {
		return "", studioSaveError(e)
	}
	if id == "" {
		return "", ErrCodeRelations
	}
	return id, nil
}
func studioSaveError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "studio with name") && strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("%w: %v", ErrStudioConflict, err)
	}
	return fmt.Errorf("%w: studio creation or update failed: %v", ErrCodeRelations, err)
}
