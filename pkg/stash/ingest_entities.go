package stash

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

// FindIngestEntity resolves an exact remote identity, never a similar name.
// It is used to reconcile an entity creation whose response may have been lost.
func (c *Client) FindIngestEntity(ctx context.Context, kind, endpoint, remoteID string) (string, error) {
	if remoteID == "" {
		return "", nil
	}
	criterion := gql.StashIDCriterionInput{Endpoint: &endpoint, StashID: &remoteID, Modifier: gql.CriterionModifierEquals}
	var ids []string
	switch kind {
	case "studio":
		r, e := c.resolve().FindIngestStudios(ctx, criterion)
		if e != nil {
			return "", e
		}
		for _, v := range r.FindStudios.Studios {
			if v != nil {
				ids = append(ids, v.ID)
			}
		}
	case "performer":
		r, e := c.resolve().FindIngestPerformers(ctx, criterion)
		if e != nil {
			return "", e
		}
		for _, v := range r.FindPerformers.Performers {
			if v != nil {
				ids = append(ids, v.ID)
			}
		}
	case "tag":
		r, e := c.resolve().FindIngestTags(ctx, criterion)
		if e != nil {
			return "", e
		}
		for _, v := range r.FindTags.Tags {
			if v != nil {
				ids = append(ids, v.ID)
			}
		}
	default:
		return "", errors.New("unknown Stash relation kind")
	}
	if len(ids) > 1 {
		return "", errors.New("multiple Stash entities share the scraped identity")
	}
	if len(ids) == 1 {
		return ids[0], nil
	}
	return "", nil
}
func (c *Client) CreateIngestStudio(ctx context.Context, identity string, input gql.StudioCreateInput) (string, error) {
	client, e := c.ingestWriteClient(identity)
	if e != nil {
		return "", e
	}
	r, e := client.CreateIngestStudio(ctx, input)
	if e != nil {
		return "", e
	}
	if r.StudioCreate == nil {
		return "", errors.New("Stash did not confirm studio creation")
	}
	return r.StudioCreate.ID, nil
}
func (c *Client) UpdateIngestStudio(ctx context.Context, identity string, input gql.StudioUpdateInput) error {
	client, e := c.ingestWriteClient(identity)
	if e != nil {
		return e
	}
	r, e := client.UpdateIngestStudio(ctx, input)
	if e != nil {
		return e
	}
	if r.StudioUpdate == nil || r.StudioUpdate.ID != input.ID {
		return errors.New("Stash did not confirm studio update")
	}
	return nil
}
func (c *Client) CreateIngestPerformer(ctx context.Context, identity string, input gql.PerformerCreateInput) (string, error) {
	client, e := c.ingestWriteClient(identity)
	if e != nil {
		return "", e
	}
	r, e := client.CreateIngestPerformer(ctx, input)
	if e != nil {
		return "", e
	}
	if r.PerformerCreate == nil {
		return "", errors.New("Stash did not confirm performer creation")
	}
	return r.PerformerCreate.ID, nil
}
func (c *Client) CreateIngestTag(ctx context.Context, identity string, input gql.TagCreateInput) (string, error) {
	client, e := c.ingestWriteClient(identity)
	if e != nil {
		return "", e
	}
	r, e := client.CreateIngestTag(ctx, input)
	if e != nil {
		return "", e
	}
	if r.TagCreate == nil {
		return "", errors.New("Stash did not confirm tag creation")
	}
	return r.TagCreate.ID, nil
}
func (c *Client) IngestStudioStashIDs(ctx context.Context, id string) ([]*gql.StashIDInput, error) {
	r, e := c.resolve().IngestStudio(ctx, id)
	if e != nil {
		return nil, e
	}
	if r.FindStudio == nil {
		return nil, errors.New("Stash parent studio is missing")
	}
	ids := []*gql.StashIDInput{}
	for _, v := range r.FindStudio.StashIds {
		if v != nil {
			ids = append(ids, &gql.StashIDInput{Endpoint: v.Endpoint, StashID: v.StashID, UpdatedAt: &v.UpdatedAt})
		}
	}
	return ids, nil
}

// Read only the configured Stash screenshot endpoint. No remote image URL or
// redirect receives Stash credentials. The bytes are hashed, never logged.
func (c *Client) IngestCoverDigest(ctx context.Context, identity, id string) (string, error) {
	client, e := c.ingestWriteClient(identity)
	if e != nil {
		return "", e
	}
	c.mu.RLock()
	cfg := c.lastConfig
	transport := c.httpClient.Transport
	valid := c.graphql == client
	c.mu.RUnlock()
	if !valid {
		return "", errors.New("Stash connection changed")
	}
	u, e := url.Parse(cfg.URL)
	if e != nil {
		return "", e
	}
	credentials := u.User
	u.RawQuery = ""
	u.Fragment = ""
	u.User = nil
	u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), "/graphql") + "/scene/" + url.PathEscape(id) + "/screenshot"
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if e != nil {
		return "", e
	}
	req.Header.Set("ApiKey", cfg.APIKey)
	if credentials != nil {
		password, _ := credentials.Password()
		req.SetBasicAuth(credentials.Username(), password)
	}
	httpClient := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, e := httpClient.Do(req)
	if e != nil {
		return "", e
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if response.StatusCode != http.StatusOK {
		return "", errors.New("Stash cover could not be read")
	}
	data, e := io.ReadAll(io.LimitReader(response.Body, 8*1024*1024+1))
	if e != nil {
		return "", e
	}
	if len(data) > 8*1024*1024 {
		return "", errors.New("Stash cover exceeds image size limit")
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
