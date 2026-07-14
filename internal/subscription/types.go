package subscription

import (
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/performer"
)

const DefaultCustomFieldKey = performer.DefaultCustomFieldKey

type Release struct {
	SceneID, Key, Source, Title, Code, Date, URL string
	PerformerCount                               int
	PerformerNames                               []string
	Classification                               ReleaseClassification
	Decision                                     ReleaseDecision
	DecisionReason                               string
}
type RecordedRelease struct {
	Key            string                `json:"key"`
	Source         string                `json:"source"`
	Title          string                `json:"title"`
	Code           string                `json:"code,omitempty"`
	Date           string                `json:"date,omitempty"`
	URL            string                `json:"url,omitempty"`
	TaskID         string                `json:"task_id,omitempty"`
	SeenAt         time.Time             `json:"seen_at"`
	PerformerCount int                   `json:"performer_count,omitempty"`
	PerformerNames []string              `json:"performer_names,omitempty"`
	Classification ReleaseClassification `json:"classification,omitempty"`
	Decision       ReleaseDecision       `json:"decision,omitempty"`
	DecisionReason string                `json:"decision_reason,omitempty"`
}
type PerformerState struct {
	PerformerID       string            `json:"performer_id"`
	LastCheckedAt     *time.Time        `json:"last_checked_at,omitempty"`
	LastError         string            `json:"last_error,omitempty"`
	ProcessedReleases []RecordedRelease `json:"processed_releases,omitempty"`
	PendingReleases   []RecordedRelease `json:"pending_releases,omitempty"`
}
type SubscribedPerformer struct {
	Performer                                  performer.Performer
	LastCheckedAt                              *time.Time
	LastError                                  string
	PendingReleaseCount, ProcessedReleaseCount int
	RecentReleases                             []RecordedRelease
}

type PerformerBatchStatus string

const (
	PerformerBatchStatusSucceeded PerformerBatchStatus = "SUCCEEDED"
	PerformerBatchStatusSkipped   PerformerBatchStatus = "SKIPPED"
	PerformerBatchStatusFailed    PerformerBatchStatus = "FAILED"
)

const (
	PerformerBatchReasonSubscribed        = "SUBSCRIBED"
	PerformerBatchReasonUnsubscribed      = "UNSUBSCRIBED"
	PerformerBatchReasonRefreshed         = "REFRESHED"
	PerformerBatchReasonAlreadySubscribed = "ALREADY_SUBSCRIBED"
	PerformerBatchReasonNotSubscribed     = "NOT_SUBSCRIBED"
	PerformerBatchReasonPerformerNotFound = "PERFORMER_NOT_FOUND"
	PerformerBatchReasonStashUpdateFailed = "STASH_UPDATE_FAILED"
	PerformerBatchReasonRefreshFailed     = "REFRESH_FAILED"
	PerformerBatchReasonCancelled         = "CANCELLED"
)

type PerformerBatchResult struct {
	PerformerID string
	Status      PerformerBatchStatus
	ReasonCode  string
	Performer   *performer.Performer
	State       *SubscribedPerformer
}

type PerformerBatchSummary struct {
	RequestedCount int
	SucceededCount int
	SkippedCount   int
	FailedCount    int
}

type PerformerBatchPayload struct {
	BatchID string
	Summary PerformerBatchSummary
	Results []PerformerBatchResult
}
type ReleaseClassification string

const (
	ReleaseClassificationSolo            ReleaseClassification = "SOLO"
	ReleaseClassificationSmallGroup      ReleaseClassification = "SMALL_GROUP"
	ReleaseClassificationLargeGroup      ReleaseClassification = "LARGE_GROUP"
	ReleaseClassificationCompilationLike ReleaseClassification = "COMPILATION_LIKE"
	ReleaseClassificationUnknown         ReleaseClassification = "UNKNOWN"
)

type ReleaseDecision string

const (
	ReleaseDecisionDownloaded ReleaseDecision = "DOWNLOADED"
	ReleaseDecisionQueued     ReleaseDecision = "QUEUED"
	ReleaseDecisionBlocked    ReleaseDecision = "BLOCKED"
)

type ReleaseEvaluation struct {
	PerformerCount int
	PerformerNames []string
	Classification ReleaseClassification
	Decision       ReleaseDecision
	DecisionReason string
}
type ReleasePolicyConfig = config.SubscriptionReleasePolicyConfig
