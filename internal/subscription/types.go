package subscription

import (
	"time"

	"github.com/leothevan2444/moji/internal/config"
)

const DefaultCustomFieldKey = "moji_subscribed"

type Performer struct {
	ID         string
	Name       string
	AliasList  []string
	Favorite   bool
	ImagePath  string
	SceneCount int
	Subscribed bool
}

type MatchedStashBox struct {
	Name          string
	Endpoint      string
	PerformerID   string
	PerformerName string
}

type DiscoveredScene struct {
	Key              string
	SceneID          string
	StashBoxEndpoint string
	StashBoxName     string
	Title            string
	DurationSeconds  *int
	Code             string
	Date             string
	StudioName       string
	ImageURL         string
	URL              string
	PerformerNames   []string
	DerivedQuery     string
}

type DiscoverScenePage struct {
	Items         []DiscoveredScene
	UsedStashBox  *MatchedStashBox
	FallbackCount int
	SearchedQuery string
}

type StashSceneID struct {
	Endpoint string
	StashID  string
}

type SceneSource string

const (
	SceneSourceStash    SceneSource = "STASH"
	SceneSourceStashBox SceneSource = "STASHBOX"
)

// DiscoverSort orders discovered scenes returned from a StashBox search.
// RELEVANCE preserves StashBox's native ordering — the search backend
// already weighs relevance, so we don't reshuffle on the client.
type DiscoverSort string

const (
	DiscoverSortRelevance    DiscoverSort = "RELEVANCE"
	DiscoverSortDateDesc     DiscoverSort = "DATE_DESC"
	DiscoverSortDateAsc      DiscoverSort = "DATE_ASC"
	DiscoverSortDurationDesc DiscoverSort = "DURATION_DESC"
	DiscoverSortTitleAsc     DiscoverSort = "TITLE_ASC"
)

type SceneSourceFilter string

const (
	SceneSourceFilterAll      SceneSourceFilter = "ALL"
	SceneSourceFilterStash    SceneSourceFilter = "STASH"
	SceneSourceFilterStashBox SceneSourceFilter = "STASHBOX"
)

type LibraryFilter string

const (
	LibraryFilterAll          LibraryFilter = "ALL"
	LibraryFilterInLibrary    LibraryFilter = "IN_LIBRARY"
	LibraryFilterNotInLibrary LibraryFilter = "NOT_IN_LIBRARY"
)

type PerformerScene struct {
	Key                 string
	PrimarySource       SceneSource
	SourceSceneID       string
	Title               string
	Code                string
	Date                string
	StudioName          string
	ImageURL            string
	URL                 string
	InLibrary           bool
	MatchedStashSceneID string
	HasStashSource      bool
	HasStashBoxSource   bool
	StashBoxSceneID     string
	StashBoxEndpoint    string
	SourceLabels        []string
	StashIDs            []StashSceneID
}

type PerformerSceneQuery struct {
	Search    string
	Source    SceneSourceFilter
	InLibrary LibraryFilter
	Page      int
	PageSize  int
}

type PerformerScenePage struct {
	Items           []PerformerScene
	Page            int
	PageSize        int
	TotalCount      int
	TotalPages      int
	HasPrevPage     bool
	HasNextPage     bool
	StashSceneCount int
	StashBoxCount   int
	DedupedCount    int
}

type PerformerDetail struct {
	Performer          Performer
	Disambiguation     string
	Birthdate          string
	Ethnicity          string
	Country            string
	EyeColor           string
	HeightCm           *int
	Rating100          *int
	URLs               []string
	MatchedStashBox    *MatchedStashBox
	TotalSceneCount    int
	StashSceneCount    int
	StashBoxSceneCount int
	DedupedSceneCount  int
}

type Release struct {
	Key            string
	Source         string
	Title          string
	Code           string
	Date           string
	URL            string
	Query          string
	PerformerCount int
	PerformerNames []string
	Classification ReleaseClassification
	Decision       ReleaseDecision
	DecisionReason string
}

type RecordedRelease struct {
	Key            string                `json:"key"`
	Source         string                `json:"source"`
	Title          string                `json:"title"`
	Code           string                `json:"code,omitempty"`
	Date           string                `json:"date,omitempty"`
	URL            string                `json:"url,omitempty"`
	Query          string                `json:"query"`
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
	Performer             Performer
	LastCheckedAt         *time.Time
	LastError             string
	PendingReleaseCount   int
	ProcessedReleaseCount int
	RecentReleases        []RecordedRelease
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
