package taskruntime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/resourceselection"
)

func selectionService(t *testing.T) *Service {
	t.Helper()
	s, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, NewMemoryTaskStore())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func magnetic(title, hash string) indexer.SelectionCandidate {
	return indexer.SelectionCandidate{Title: title, MagnetURI: "magnet:?xt=urn:btih:" + strings.Repeat(hash, 40)}
}
func TestRoundKeepsFallbackAcrossSitesAndStopsOnPreferred(t *testing.T) {
	p := config.DefaultCandidateSelectionConfig()
	p.Rules = []resourceselection.Rule{{ID: "plain", Name: "User plain names", Conditions: []resourceselection.Condition{{Scope: "TITLE", Value: "/^ABCD-123$/"}}}, {ID: "sub", Name: "User subtitle names", Conditions: []resourceselection.Condition{{Scope: "TITLE", Value: "中字"}}}}
	p.ActiveGroup = "main"
	p.Groups = []resourceselection.Group{{Fallback: true, ID: "main", Name: "Main", Tiers: []resourceselection.Tier{{ID: "plain", Name: "First", Rules: []string{"plain"}}, {ID: "sub", Name: "Second", Rules: []string{"sub"}}}}}
	round := newSelectionRound(selectionService(t).now, "ABCD-123", p)
	ctx := context.Background()
	stop, err := round.add(ctx, []indexer.SelectionCandidate{magnetic("ABCD-123 无码破解", "1")})
	if err != nil || stop {
		t.Fatal("modified resource stopped search", err)
	}
	stop, err = round.add(ctx, []indexer.SelectionCandidate{magnetic("ABCD-123 中字", "2")})
	if err != nil || stop {
		t.Fatal("hard-subtitle resource stopped search", err)
	}
	got, err := round.finish(ctx)
	if err != nil || !strings.Contains(got.Title, "中字") {
		t.Fatal("cross-site fallback lost", got, err)
	}
	round = newSelectionRound(selectionService(t).now, "ABCD-123", p)
	stop, err = round.add(ctx, []indexer.SelectionCandidate{magnetic("ABCD-123", "3")})
	if err != nil || !stop {
		t.Fatal("preferred valid resource did not stop", err)
	}
}
func TestSelectionCannotTreatMissingIndexerDataAsSatisfied(t *testing.T) {
	p := config.DefaultCandidateSelectionConfig()
	p.Rules = []resourceselection.Rule{{ID: "size", Name: "Size", Conditions: []resourceselection.Condition{{Scope: "SIZE", Minimum: "1"}}}}
	p.ActiveGroup = "main"
	p.Groups = []resourceselection.Group{{ID: "main", Name: "Main", Fallback: true, Require: []string{"size"}}}
	round := newSelectionRound(selectionService(t).now, "ABCD-123", p)
	row := magnetic("ABCD-123", "1")
	row.Size = -1
	if _, err := round.add(context.Background(), []indexer.SelectionCandidate{row}); err != nil {
		t.Fatal(err)
	}
	if _, err := round.finish(context.Background()); err == nil {
		t.Fatal("missing size matched numeric rule")
	}
}
func TestSelectionReportPersistsWithCASAndKeepsPrivateLocatorsOut(t *testing.T) {
	store, _ := relationalStore(t)
	task := richRelationalTask()
	round := newSelectionRound(selectionService(t).now, "ABCD-123", config.DefaultCandidateSelectionConfig())
	round.add(context.Background(), []indexer.SelectionCandidate{magnetic("ABCD-123", "1")})
	round.finish(context.Background())
	task.SelectionReportJSON = round.json()
	if strings.Contains(task.SelectionReportJSON, "magnet:") {
		t.Fatal("report leaked acquisition locator")
	}
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	saved, err := store.Find(context.Background(), task.ID)
	if err != nil || saved.SelectionReportJSON != task.SelectionReportJSON {
		t.Fatal("report not persisted", err)
	}
	saved.SelectionReportJSON = `{"version":1,"end":"INCOMPLETE"}`
	saved.StageErrorMessage = "new sourcing result"
	if err = store.Update(context.Background(), saved); err != nil {
		t.Fatal(err)
	}
	if err = store.Update(context.Background(), task); err != ErrTaskRevisionConflict {
		t.Fatalf("stale decision replaced report: %v", err)
	}
}
func TestReportTrimsRepresentativesWithoutLosingSelectedDecision(t *testing.T) {
	r := newSelectionRound(selectionService(t).now, "ABCD-123", config.DefaultCandidateSelectionConfig())
	for i := 0; i < 32; i++ {
		r.report.Policy.Rules = append(r.report.Policy.Rules, resourceselection.Rule{ID: "test", Name: strings.Repeat("x", 160), Conditions: []resourceselection.Condition{{Scope: "TITLE", Value: strings.Repeat("x", 512)}}})
	}
	for i := 0; i < 25; i++ {
		title := strings.Repeat("長", 3000)
		e := resourceselection.Evaluation{Facts: resourceselection.Facts{Title: title}, Status: "REJECTED"}
		for j := 0; j < 32; j++ {
			e.Reasons = append(e.Reasons, strings.Repeat("x", 160))
		}
		r.items = append(r.items, &selectionItem{evaluation: e})
	}
	r.selected = r.items[0]
	r.report.Selected = r.items[0].evaluation.Facts.Title
	raw := r.json()
	if len(raw) == 0 || len(raw) > 256<<10 || !json.Valid([]byte(raw)) {
		t.Fatal("bounded report discarded decision", len(raw))
	}
}

func TestManualConfirmationCannotOverrideTargetIdentity(t *testing.T) {
	s := selectionService(t)
	task := &Task{Code: "ABCD-123", ResourceConfirmed: true, Candidate: Candidate{Title: "ABCD-123"}}
	metadata, err := parseTorrentMetadata([]byte(testTorrentFile("ABCD-999", []string{"ABCD-999.mp4"})))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.prepareConfirmedResource(context.Background(), task, "https://example.test/wrong", &metadata); SourcingValidationErrorCode(err) != TaskStageErrorCandidateUnverified {
		t.Fatal("manual choice bypassed actual film identity", err)
	}
	if _, err = s.prepareConfirmedResource(context.Background(), task, "magnet:?xt=urn:btih:"+strings.Repeat("1", 40), nil); err != nil {
		t.Fatal("matching explicit title should establish magnet identity", err)
	}
	task.Candidate.Title = "unrelated"
	if _, err = s.prepareConfirmedResource(context.Background(), task, "magnet:?xt=urn:btih:"+strings.Repeat("1", 40), nil); err == nil {
		t.Fatal("manual magnet without target evidence accepted")
	}
}

func TestReportNamesTheActualCandidateWhenSourcesShareALocator(t *testing.T) {
	r := newSelectionRound(selectionService(t).now, "ABCD-123", config.DefaultCandidateSelectionConfig())
	first := magnetic("WRONG-123", "1")
	second := magnetic("ABCD-123", "1")
	r.add(context.Background(), []indexer.SelectionCandidate{first, second})
	winner, err := r.finish(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var report selectionReport
	if err = json.Unmarshal([]byte(r.json()), &report); err != nil || report.Selected != winner.Title {
		t.Fatal("report describes the wrong source title", report.Selected, winner.Title, err)
	}
}
func TestAutomaticAndManualPreparationRejectMalformedMagnets(t *testing.T) {
	s := selectionService(t)
	raw := "magnet:?xt=urn:btih:not-a-valid-hash&dn=ABCD-123"
	if _, err := s.prepareTorrent(context.Background(), "ABCD-123", Candidate{Title: "ABCD-123"}, raw); err == nil {
		t.Fatal("malformed automatic magnet accepted")
	}
	if _, err := s.prepareConfirmedResource(context.Background(), &Task{Code: "ABCD-123", ResourceConfirmed: true, Candidate: Candidate{Title: "ABCD-123"}}, raw, nil); err == nil {
		t.Fatal("manual confirmation bypassed URI validation")
	}
}

func TestLegacyManualReportIsNotReinterpretedUnderNewRules(t *testing.T) {
	s := selectionService(t)
	for _, raw := range []string{
		`{"version":2,"end":"USER_SELECTED","policy":{"version":6,"activeGroup":"main","groups":[{"id":"main","tiers":[{"id":"empty"}]}]},"selected":"ABCD-123"}`,
		`{"version":2,"end":"USER_SELECTED","policy":{"version":6,"activeGroup":"main","groups":[{"id":"main","tiers":[{"id":"first","rules":[{"ruleID":"label","negate":true}]}]}]},"selected":"ABCD-123"}`,
		`{"version":2,"end":"USER_SELECTED","policy":{"version":5,"perSite":5,"maxFetches":30},"selected":"ABCD-123"}`,
		`{"version":1,"end":"USER_SELECTED","policy":{"version":4,"groups":[{"id":"old","requireMode":"ALL","excludeMode":"ANY","tiers":[{"id":"fallback","mode":"ALL","matchAll":true,"rules":[]}]}]},"selected":"ABCD-123"}`,
		`{"version":1,"end":"USER_SELECTED","policy":{"version":1,"original":true},"selected":"ABCD-123"}`,
		`{"version":1,"end":"USER_SELECTED","policy":{"version":3,"rules":[{"id":"old","scope":"TITLE","value":"historic"}]},"selected":"ABCD-123"}`,
		`{"version":1,"end":"USER_SELECTED","policy":{"version":2,"rules":[{"id":"old","mode":"ANY","conditions":[{"scope":"TITLE","operator":"CONTAINS","value":"historic"}]}]},"selected":"ABCD-123"}`,
	} {
		task := &Task{Code: "ABCD-123", SelectionReportJSON: raw}
		if got := s.manualSelectionReport(task); got != raw {
			t.Fatal("historical manual decision was reinterpreted", got)
		}
	}
}
