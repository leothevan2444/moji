package indexer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/pkg/jackett"
)

type externalClient struct {
	api      *jackett.Client
	session  *HTTPSession
	refs     int
	lastUsed time.Time
}
type JackettAdapter struct {
	network *Network
	mu      sync.Mutex
	clients map[string]*externalClient
	closed  bool
}

func NewJackettAdapter(network *Network) (*JackettAdapter, error) {
	if network == nil {
		return nil, ErrNotConfigured
	}
	return &JackettAdapter{network: network, clients: map[string]*externalClient{}}, nil
}

// Close releases transports after this adapter's owner has joined its calls.
func (a *JackettAdapter) Close() {
	a.mu.Lock()
	a.closed = true
	clients := a.clients
	a.clients = map[string]*externalClient{}
	a.mu.Unlock()
	for _, client := range clients {
		client.session.Close()
	}
}

func (a *JackettAdapter) client(instance InstanceSnapshot) (*externalClient, func(), error) {
	if instance.BaseURL == "" || instance.JackettAPIKey == "" {
		return nil, nil, ErrNotConfigured
	}
	data, _ := json.Marshal(struct {
		URL, Key string
		Network  NetworkProfile
		Timeout  time.Duration
	}{instance.BaseURL, instance.JackettAPIKey, instance.Network, instance.RequestTimeout})
	sum := sha256.Sum256(data)
	key := hex.EncodeToString(sum[:])
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil, nil, context.Canceled
	}
	defer a.mu.Unlock()
	client := a.clients[key]
	if client == nil {
		if len(a.clients) >= 32 {
			oldest := ""
			var at time.Time
			for id, entry := range a.clients {
				if entry.refs == 0 && (oldest == "" || entry.lastUsed.Before(at)) {
					oldest = id
					at = entry.lastUsed
				}
			}
			if oldest == "" {
				return nil, nil, ErrBusy
			}
			a.clients[oldest].session.Close()
			delete(a.clients, oldest)
		}
		session, err := a.network.newExternalSession(instance)
		if err != nil {
			return nil, nil, err
		}
		httpClient := &http.Client{Transport: session.transport, Timeout: session.timeout, CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if len(via) > 5 || originOf(next.URL) != originOf(session.base) {
				return ErrUnsafeURL
			}
			if _, err := parseHTTPURL(next.URL.String()); err != nil {
				return err
			}
			return nil
		}}
		api, err := jackett.NewAPIClient(instance.BaseURL, instance.JackettAPIKey, httpClient)
		if err != nil {
			session.Close()
			return nil, nil, ErrInvalidSettings
		}
		client = &externalClient{api: api, session: session}
		a.clients[key] = client
	}
	client.refs++
	return client, func() { a.mu.Lock(); client.refs--; client.lastUsed = time.Now(); a.mu.Unlock() }, nil
}

func (a *JackettAdapter) Search(ctx context.Context, instance InstanceSnapshot, query Query) (Batch, error) {
	if instance.Kind != Jackett || instance.RemoteID == "" {
		return Batch{}, ErrInvalidSettings
	}
	if query.Mode != "" && query.Mode != "search" || len(query.Parameters) != 0 || query.Offset != 0 {
		return Batch{}, ErrUnsupported
	}
	client, release, err := a.client(instance)
	if err != nil {
		return Batch{}, err
	}
	defer release()
	receipt := &responseReceipt{}
	rows, statuses, err := client.api.SearchWithIndexerStatusContext(context.WithValue(ctx, receiptKey{}, receipt), jackett.SearchRequest{Query: query.Keywords, Trackers: []string{instance.RemoteID}, Categories: query.Categories})
	if err != nil {
		return Batch{}, jackettError(ctx, err, receipt)
	}
	batch := Batch{InstanceID: instance.ID}
	for _, status := range statuses {
		if status.ID == instance.RemoteID && (status.Status >= 400 || status.Error != "") {
			batch.Partial = true
		}
	}
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	if len(rows) > limit {
		rows = rows[:limit]
		batch.Truncated = true
	}
	for _, source := range rows {
		if source.TrackerID != "" && source.TrackerID != instance.RemoteID {
			batch.Partial = true
			continue
		}
		ref := ResourceRef{Version: 1, ID: uuid.NewString(), InstanceID: instance.ID, Kind: Jackett, ConfigRevision: instance.ConfigRevision, Locator: Locator{DownloadURL: source.Link, MagnetURI: source.MagnetURI, InfoHash: source.InfoHash, Title: source.Title}}
		release := Release{Title: source.Title, GUID: source.GUID, InstanceID: instance.ID, InstanceName: instance.Name, Kind: Jackett, DetailsURL: source.Details, Categories: append([]int(nil), source.Category...), InfoHash: source.InfoHash, Resource: ref}
		if source.SizeKnown {
			size := source.Size
			release.Size = &size
		}
		if source.Seeders >= 0 {
			seeders := source.Seeders
			release.Seeders = &seeders
		}
		if source.PeersKnown {
			peers := source.Peers
			release.Peers = &peers
		}
		if published, err := time.Parse(time.RFC3339, source.PublishDate); err == nil {
			release.PublishedAt = &published
		}
		batch.Releases = append(batch.Releases, release)
	}
	if batch.Partial {
		return batch, ErrTransport
	}
	return batch, nil
}

func (a *JackettAdapter) Fetch(ctx context.Context, instance InstanceSnapshot, ref ResourceRef) (Artifact, error) {
	if ref.Version != 1 || ref.Kind != Jackett || ref.InstanceID != instance.ID || ref.ConfigRevision != instance.ConfigRevision {
		return Artifact{}, ErrReference
	}
	target := ref.Locator.DownloadURL
	if target == "" {
		target = ref.Locator.MagnetURI
	}
	if target == "" {
		return Artifact{}, ErrReference
	}
	client, release, err := a.client(instance)
	if err != nil {
		return Artifact{}, err
	}
	defer release()
	base := client.session.base
	for redirects := 0; redirects <= 5; redirects++ {
		parsed, err := url.Parse(target)
		if err != nil || parsed.User != nil {
			return Artifact{}, ErrReference
		}
		parsed = base.ResolveReference(parsed)
		if parsed.Scheme == "magnet" {
			if parsed.Query().Get("xt") == "" {
				return Artifact{}, ErrReference
			}
			return Artifact{Magnet: parsed.String(), Resource: ref, AcquiredAt: time.Now().UTC()}, nil
		}
		response, err := client.session.Do(ctx, cardigann.Request{URL: parsed.String(), Torrent: true})
		if err != nil {
			return Artifact{}, err
		}
		if response.Status >= 300 && response.Status < 400 {
			target = response.Headers.Get("Location")
			if target == "" {
				return Artifact{}, ErrTransport
			}
			base, _ = url.Parse(response.URL)
			continue
		}
		if response.Status < 200 || response.Status >= 300 {
			return Artifact{}, ErrTransport
		}
		return Artifact{TorrentBytes: response.Body, Resource: ref, AcquiredAt: time.Now().UTC()}, nil
	}
	return Artifact{}, ErrTransport
}

func (a *JackettAdapter) Catalog(ctx context.Context, connection InstanceSnapshot) ([]InstanceSnapshot, error) {
	client, release, err := a.client(connection)
	if err != nil {
		return nil, err
	}
	defer release()
	receipt := &responseReceipt{}
	sites, err := client.api.GetAPIIndexersContext(context.WithValue(ctx, receiptKey{}, receipt))
	if err != nil {
		return nil, jackettError(ctx, err, receipt)
	}
	if len(sites) > 1024 {
		return nil, ErrResponseBudget
	}
	out := []InstanceSnapshot{}
	for _, site := range sites {
		if !site.Configured {
			continue
		}
		copy := cloneInstance(connection)
		copy.ID = "jackett/" + site.ID
		copy.Kind = Jackett
		copy.RemoteID = site.ID
		copy.Name = site.Name
		copy.SiteURL, copy.SiteType, copy.Language = site.SiteLink, site.Type, site.Language
		out = append(out, copy)
	}
	return out, nil
}

func jackettError(ctx context.Context, err error, receipt *responseReceipt) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if receipt != nil && receipt.status == http.StatusTooManyRequests && !receipt.retryAt.IsZero() {
		return &CoolingDownError{Until: receipt.retryAt}
	}
	var status *jackett.HTTPError
	if errors.As(err, &status) {
		if status.Status == 429 {
			return &CoolingDownError{Until: time.Now().Add(retryDelay(status.RetryAfter, 1))}
		}
	}
	var cooling *CoolingDownError
	if errors.As(err, &cooling) {
		return cooling
	}
	for _, known := range []error{ErrUnsafeURL, ErrTransport, ErrStateUnavailable, ErrBusy, context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, known) {
			return known
		}
	}
	if errors.Is(err, jackett.ErrResponseTooLarge) {
		return ErrResponseBudget
	}
	return ErrTransport
}
