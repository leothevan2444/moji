package indexeradmin

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexerdefs"
)

func TestSummaryKeepsUnknownSeparateFromConfirmedEmptyAndCountsOnlyEnabledSites(t *testing.T) {
	page := InstancePage{JackettEnabled: true, JackettConfigured: true, Items: []Instance{
		{ID: "builtin", Kind: indexer.Builtin, Enabled: true},
		{ID: "disabled", Kind: indexer.Builtin, Enabled: false, ErrorCode: "broken"},
		{ID: "jackett/one", Kind: indexer.Jackett, Enabled: true},
		{ID: "jackett/two", Kind: indexer.Jackett, Enabled: false},
	}}
	unknown := summarize(page)
	if unknown.Total != nil || unknown.Jackett == nil || unknown.Jackett.Sites != nil || unknown.BuiltinEnabled != 1 || unknown.BuiltinUnverified != 1 || unknown.Attention != 0 {
		t.Fatal("unloaded directory presented as known or disabled sources counted")
	}
	page.JackettCatalog.Known = true
	known := summarize(page)
	if known.Total == nil || *known.Total != 2 || known.Jackett.Sites == nil || *known.Jackett.Sites != 1 {
		t.Fatal("gateway counted as a site or enabled remote site lost")
	}
	page.Items = nil
	empty := summarize(page)
	if empty.Total == nil || *empty.Total != 0 || empty.Jackett.Sites == nil || *empty.Jackett.Sites != 0 {
		t.Fatal("confirmed empty directory became unknown")
	}
	for _, configured := range []bool{false, true} {
		page.JackettConfigured = configured
		page.JackettEnabled = !configured
		if summarize(page).Jackett != nil {
			t.Fatal("unconfigured or disabled gateway shown")
		}
	}
}

func TestSummaryBoundsIssuesAndRetainsStaleConfirmedCounts(t *testing.T) {
	confirmed := time.Now().Add(-time.Hour)
	page := InstancePage{JackettEnabled: true, JackettConfigured: true, CatalogErrorCode: indexer.ErrTransport.Error(), JackettCatalog: indexer.CatalogStatus{Known: true, Stale: true, UpdatedAt: &confirmed}}
	for i := 0; i < 20; i++ {
		page.Items = append(page.Items, Instance{ID: fmt.Sprint(i), Kind: indexer.Builtin, Enabled: true, ErrorCode: indexer.ErrInvalidSettings.Error()})
	}
	page.Items = append(page.Items, Instance{ID: "jackett/one", Kind: indexer.Jackett, Enabled: true})
	result := summarize(page)
	if len(result.Issues) != 5 || result.Attention != 21 || result.BuiltinUnverified != 0 || result.Total == nil || *result.Total != 21 || result.Jackett.Sites == nil || *result.Jackett.Sites != 1 || !result.Jackett.Stale || !result.Jackett.UpdatedAt.Equal(confirmed) {
		t.Fatal("bounded issue projection or stale last-confirmed counts lost")
	}
}

func TestSummaryKeepsSuccessfulSearchAndFailedAcquisitionSeparate(t *testing.T) {
	page := InstancePage{Items: []Instance{{ID: "site", Kind: indexer.Builtin, Enabled: true, Health: indexer.HealthState{Search: &indexer.Observation{StartedAt: time.Now().Add(-time.Minute)}, Fetch: &indexer.Observation{StartedAt: time.Now(), ErrorCode: indexer.ErrAuthentication.Error()}}}}}
	summary := summarize(page)
	if summary.BuiltinSearchSucceeded != 1 || summary.BuiltinUnverified != 0 || summary.Attention != 1 || len(summary.Issues) != 1 || summary.Issues[0].Operation != "FETCH" {
		t.Fatal("search success hid acquisition failure or implied unverified access")
	}
	page.HealthErrorCode = "INDEXER_STATUS_UNAVAILABLE"
	page.Items[0].Health = indexer.HealthState{}
	summary = summarize(page)
	if summary.BuiltinUnverified != 0 || summary.HealthErrorCode == "" {
		t.Fatal("unreadable history presented as no previous observation")
	}
}

func TestSummaryUsesLocalSnapshotWithoutRefreshingJackett(t *testing.T) {
	service, _, _, registry := managementFixture(t)
	result, err := service.Summary(context.Background())
	if err != nil || registry.refreshes != 0 || result.Jackett == nil || result.Jackett.Sites != nil {
		t.Fatal("Home contacted remote directory or fabricated count", err)
	}
}

type summaryJobReader struct {
	job   *indexerdefs.Job
	err   error
	reads int
}

func (r *summaryJobReader) CurrentJob(context.Context) (*indexerdefs.Job, error) {
	r.reads++
	return r.job, r.err
}
func TestHomeShowsOnlyCurrentFailedDefinitionJobWithoutRemoteActions(t *testing.T) {
	service, _, _, registry := managementFixture(t)
	reader := &summaryJobReader{}
	service.SetDefinitionJobReader(reader)
	for _, state := range []indexerdefs.JobState{indexerdefs.Failed, indexerdefs.Interrupted, indexerdefs.Cancelled, indexerdefs.Completed, indexerdefs.Downloading} {
		reader.job = &indexerdefs.Job{ID: "job", State: state, ErrorCode: "DEFINITION_SOURCE_UNAVAILABLE"}
		result, e := service.Summary(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		expected := state == indexerdefs.Failed || state == indexerdefs.Interrupted
		if (result.DefinitionJob != nil) != expected || result.Attention != map[bool]int{true: 1, false: 0}[expected] || registry.refreshes != 0 {
			t.Fatal("job outcome projected incorrectly or caused remote action", state)
		}
	}
	reader.err = errors.New("synthetic local read failure")
	result, e := service.Summary(context.Background())
	if e != nil || result.DefinitionJob != nil || result.DefinitionErrorCode != indexerdefs.ErrUnavailable.Error() || result.Jackett == nil || result.Attention != 1 {
		t.Fatal("job read failure removed other indexer state", e)
	}
	reader.err = nil
	reader.job = nil
	result, e = service.Summary(context.Background())
	if e != nil || result.DefinitionJob != nil || result.DefinitionErrorCode != "" || result.Attention != 0 {
		t.Fatal("successful current read retained old failure", e)
	}
}
