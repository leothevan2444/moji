package subscription

import (
	"strings"

	"github.com/leothevan2444/moji/internal/config"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

const compilationLikePerformerThreshold = 10

var compilationLikeKeywords = []string{
	"総集編",
	"ベスト",
	"best",
	"compilation",
	"anthology",
	"omnibus",
	"highlight",
	"digest",
	"collection",
	"clip",
}

func DefaultReleasePolicyConfig() ReleasePolicyConfig {
	return config.DefaultSubscriptionReleasePolicyConfig()
}

func evaluateReleasePolicy(policy ReleasePolicyConfig, targetPerformer *stashboxgraphql.PerformerFragment, scene *stashboxgraphql.SceneFragment) (ReleaseEvaluation, bool) {
	policy = policy.Effective()
	names, matched := releasePerformerNames(targetPerformer, scene)
	if !matched {
		return ReleaseEvaluation{}, false
	}

	evaluation := ReleaseEvaluation{
		PerformerCount: len(names),
		PerformerNames: append([]string(nil), names...),
		Classification: classifyRelease(scene, names, policy.MaxGroupPerformerCount),
	}

	switch evaluation.Classification {
	case ReleaseClassificationCompilationLike:
		applyBehavior(&evaluation, policy.CompilationBehavior, "compilation_behavior")
		return evaluation, true
	case ReleaseClassificationUnknown:
		evaluation.Decision = ReleaseDecisionQueued
		evaluation.DecisionReason = "metadata_unknown_review"
		return evaluation, true
	default:
		if evaluation.Classification == ReleaseClassificationSolo {
			applyBehavior(&evaluation, policy.SoloBehavior, "solo_behavior")
			return evaluation, true
		}
		applyBehavior(&evaluation, policy.GroupBehavior, "group_behavior")
		return evaluation, true
	}
}

func applyBehavior(evaluation *ReleaseEvaluation, behavior config.SubscriptionReleaseBehavior, reasonPrefix string) {
	switch behavior {
	case config.SubscriptionReleaseBehaviorBlock:
		evaluation.Decision = ReleaseDecisionBlocked
		evaluation.DecisionReason = reasonPrefix + "_block"
	case config.SubscriptionReleaseBehaviorReview:
		evaluation.Decision = ReleaseDecisionQueued
		evaluation.DecisionReason = reasonPrefix + "_review"
	default:
		evaluation.Decision = ReleaseDecisionDownloaded
		evaluation.DecisionReason = reasonPrefix + "_download"
	}
}

func releasePerformerNames(targetPerformer *stashboxgraphql.PerformerFragment, scene *stashboxgraphql.SceneFragment) ([]string, bool) {
	if scene == nil {
		return nil, false
	}
	targetID := ""
	if targetPerformer != nil {
		targetID = strings.TrimSpace(targetPerformer.ID)
	}

	names := make([]string, 0, len(scene.Performers))
	matchedTarget := targetID == ""
	for _, appearance := range scene.Performers {
		if appearance == nil || appearance.Performer == nil {
			continue
		}
		name := strings.TrimSpace(appearance.Performer.Name)
		if name != "" {
			names = append(names, name)
		}
		if targetID != "" && strings.TrimSpace(appearance.Performer.ID) == targetID {
			matchedTarget = true
		}
	}
	return names, matchedTarget
}

func classifyRelease(scene *stashboxgraphql.SceneFragment, performerNames []string, maxGroupPerformerCount int) ReleaseClassification {
	count := len(performerNames)
	if isCompilationLikeScene(scene, count) {
		return ReleaseClassificationCompilationLike
	}
	switch {
	case count == 0:
		return ReleaseClassificationUnknown
	case count == 1:
		return ReleaseClassificationSolo
	case count <= config.NormalizeSubscriptionReleaseMaxGroupPerformerCount(maxGroupPerformerCount):
		return ReleaseClassificationSmallGroup
	default:
		return ReleaseClassificationLargeGroup
	}
}

func isCompilationLikeScene(scene *stashboxgraphql.SceneFragment, performerCount int) bool {
	if scene == nil {
		return false
	}
	if performerCount >= compilationLikePerformerThreshold {
		return true
	}
	fields := []string{
		normalizeForKeywordMatch(stringValue(scene.Title)),
		normalizeForKeywordMatch(stringValue(scene.Details)),
	}
	for _, tag := range scene.Tags {
		if tag == nil {
			continue
		}
		fields = append(fields, normalizeForKeywordMatch(tag.Name))
	}
	for _, field := range fields {
		for _, keyword := range compilationLikeKeywords {
			if strings.Contains(field, normalizeForKeywordMatch(keyword)) {
				return true
			}
		}
	}
	return false
}

func normalizeForKeywordMatch(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
