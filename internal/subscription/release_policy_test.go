package subscription

import (
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func TestEvaluateReleasePolicySoloBehavior(t *testing.T) {
	target := &stashboxgraphql.PerformerFragment{ID: "p1", Name: "Actor A"}
	scene := releasePolicyScene("scene-1", target, &stashboxgraphql.PerformerFragment{ID: "p1", Name: "Actor A"})

	evaluation, matched := evaluateReleasePolicy(config.SubscriptionReleasePolicyConfig{
		SoloBehavior:           config.SubscriptionReleaseBehaviorBlock,
		GroupBehavior:          config.SubscriptionReleaseBehaviorDownload,
		CompilationBehavior:    config.SubscriptionReleaseBehaviorDownload,
		MaxGroupPerformerCount: 3,
	}, time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC), target, scene)
	if !matched {
		t.Fatalf("expected target performer to match")
	}
	if evaluation.Classification != ReleaseClassificationSolo {
		t.Fatalf("expected solo classification, got %s", evaluation.Classification)
	}
	if evaluation.Decision != ReleaseDecisionBlocked || evaluation.DecisionReason != "solo_behavior_block" {
		t.Fatalf("unexpected solo evaluation: %+v", evaluation)
	}
}

func TestEvaluateReleasePolicyGroupBehaviorCoversLargeGroup(t *testing.T) {
	target := &stashboxgraphql.PerformerFragment{ID: "p1", Name: "Actor A"}
	scene := releasePolicyScene(
		"scene-2",
		target,
		&stashboxgraphql.PerformerFragment{ID: "p1", Name: "Actor A"},
		&stashboxgraphql.PerformerFragment{ID: "p2", Name: "Actor B"},
		&stashboxgraphql.PerformerFragment{ID: "p3", Name: "Actor C"},
		&stashboxgraphql.PerformerFragment{ID: "p4", Name: "Actor D"},
	)

	evaluation, matched := evaluateReleasePolicy(config.SubscriptionReleasePolicyConfig{
		SoloBehavior:           config.SubscriptionReleaseBehaviorDownload,
		GroupBehavior:          config.SubscriptionReleaseBehaviorReview,
		CompilationBehavior:    config.SubscriptionReleaseBehaviorBlock,
		MaxGroupPerformerCount: 3,
	}, time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC), target, scene)
	if !matched {
		t.Fatalf("expected target performer to match")
	}
	if evaluation.Classification != ReleaseClassificationLargeGroup {
		t.Fatalf("expected large-group classification, got %s", evaluation.Classification)
	}
	if evaluation.Decision != ReleaseDecisionQueued || evaluation.DecisionReason != "group_behavior_review" {
		t.Fatalf("unexpected group evaluation: %+v", evaluation)
	}
}

func TestEvaluateReleasePolicyCompilationBehavior(t *testing.T) {
	target := &stashboxgraphql.PerformerFragment{ID: "p1", Name: "Actor A"}
	title := "BEST anthology"
	scene := releasePolicyScene("scene-3", target, &stashboxgraphql.PerformerFragment{ID: "p1", Name: "Actor A"})
	scene.Title = &title

	evaluation, matched := evaluateReleasePolicy(config.SubscriptionReleasePolicyConfig{
		SoloBehavior:           config.SubscriptionReleaseBehaviorDownload,
		GroupBehavior:          config.SubscriptionReleaseBehaviorDownload,
		CompilationBehavior:    config.SubscriptionReleaseBehaviorBlock,
		MaxGroupPerformerCount: 3,
	}, time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC), target, scene)
	if !matched {
		t.Fatalf("expected target performer to match")
	}
	if evaluation.Classification != ReleaseClassificationCompilationLike {
		t.Fatalf("expected compilation classification, got %s", evaluation.Classification)
	}
	if evaluation.Decision != ReleaseDecisionBlocked || evaluation.DecisionReason != "compilation_behavior_block" {
		t.Fatalf("unexpected compilation evaluation: %+v", evaluation)
	}
}

func TestEvaluateReleasePolicyUnknownMetadataAlwaysReviews(t *testing.T) {
	target := &stashboxgraphql.PerformerFragment{ID: "p1", Name: "Actor A"}
	scene := &stashboxgraphql.SceneFragment{ID: "scene-4"}

	evaluation, matched := evaluateReleasePolicy(config.SubscriptionReleasePolicyConfig{
		SoloBehavior:           config.SubscriptionReleaseBehaviorDownload,
		GroupBehavior:          config.SubscriptionReleaseBehaviorBlock,
		CompilationBehavior:    config.SubscriptionReleaseBehaviorBlock,
		MaxGroupPerformerCount: 3,
	}, time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC), target, scene)
	if matched {
		t.Fatalf("expected unmatched target when performer metadata is absent")
	}

	scene.Performers = []*stashboxgraphql.PerformerAppearanceFragment{{Performer: &stashboxgraphql.PerformerFragment{ID: "p1"}}}
	evaluation, matched = evaluateReleasePolicy(config.SubscriptionReleasePolicyConfig{
		SoloBehavior:           config.SubscriptionReleaseBehaviorDownload,
		GroupBehavior:          config.SubscriptionReleaseBehaviorBlock,
		CompilationBehavior:    config.SubscriptionReleaseBehaviorBlock,
		MaxGroupPerformerCount: 3,
	}, time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC), target, scene)
	if !matched {
		t.Fatalf("expected target performer to match")
	}
	if evaluation.Classification != ReleaseClassificationUnknown {
		t.Fatalf("expected unknown classification, got %s", evaluation.Classification)
	}
	if evaluation.Decision != ReleaseDecisionQueued || evaluation.DecisionReason != "metadata_unknown_review" {
		t.Fatalf("unexpected unknown evaluation: %+v", evaluation)
	}
}

func releasePolicyScene(id string, target *stashboxgraphql.PerformerFragment, performers ...*stashboxgraphql.PerformerFragment) *stashboxgraphql.SceneFragment {
	scene := &stashboxgraphql.SceneFragment{ID: id}
	if len(performers) == 0 && target != nil {
		performers = []*stashboxgraphql.PerformerFragment{target}
	}
	scene.Performers = make([]*stashboxgraphql.PerformerAppearanceFragment, 0, len(performers))
	for _, performer := range performers {
		scene.Performers = append(scene.Performers, &stashboxgraphql.PerformerAppearanceFragment{Performer: performer})
	}
	return scene
}

func TestEvaluateReleasePolicyReleaseDateOutOfRangeReviews(t *testing.T) {
	target := &stashboxgraphql.PerformerFragment{ID: "p1", Name: "Actor A"}
	date := "2020-06-01"
	scene := releasePolicyScene("scene-5", target, target)
	scene.Date = &date

	evaluation, matched := evaluateReleasePolicy(config.SubscriptionReleasePolicyConfig{
		SoloBehavior:           config.SubscriptionReleaseBehaviorDownload,
		GroupBehavior:          config.SubscriptionReleaseBehaviorDownload,
		CompilationBehavior:    config.SubscriptionReleaseBehaviorDownload,
		MaxGroupPerformerCount: 3,
		ReleaseDateRange:       config.SubscriptionReleaseDateRangeOneYear,
	}, time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC), target, scene)
	if !matched {
		t.Fatalf("expected target performer to match")
	}
	if evaluation.Decision != ReleaseDecisionQueued || evaluation.DecisionReason != "release_date_out_of_range_review" {
		t.Fatalf("unexpected date-range evaluation: %+v", evaluation)
	}
}

func TestEvaluateReleasePolicyReleaseDateUnknownReviews(t *testing.T) {
	target := &stashboxgraphql.PerformerFragment{ID: "p1", Name: "Actor A"}
	scene := releasePolicyScene("scene-6", target, target)

	evaluation, matched := evaluateReleasePolicy(config.SubscriptionReleasePolicyConfig{
		SoloBehavior:           config.SubscriptionReleaseBehaviorDownload,
		GroupBehavior:          config.SubscriptionReleaseBehaviorDownload,
		CompilationBehavior:    config.SubscriptionReleaseBehaviorDownload,
		MaxGroupPerformerCount: 3,
		ReleaseDateRange:       config.SubscriptionReleaseDateRangeOneYear,
	}, time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC), target, scene)
	if !matched {
		t.Fatalf("expected target performer to match")
	}
	if evaluation.Decision != ReleaseDecisionQueued || evaluation.DecisionReason != "release_date_unknown_review" {
		t.Fatalf("unexpected unknown-date evaluation: %+v", evaluation)
	}
}
