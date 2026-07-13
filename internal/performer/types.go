package performer

import "github.com/leothevan2444/moji/internal/taskruntime"

const DefaultCustomFieldKey = "moji_subscribed"

type Performer struct {
	ID, Name   string
	AliasList  []string
	Favorite   bool
	ImagePath  string
	SceneCount int
	Subscribed bool
}
type MatchedStashBox struct{ Name, Endpoint, PerformerID, PerformerName string }
type StashSceneID struct{ Endpoint, StashID string }
type SceneSource string

const (
	SceneSourceStash    SceneSource = "STASH"
	SceneSourceStashBox SceneSource = "STASHBOX"
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

type Scene struct {
	Key                                          string
	PrimarySource                                SceneSource
	SourceSceneID, Title, Code, Date, StudioName string
	PerformerCount, TagCount                     int
	Performers                                   []ScenePerson
	Tags                                         []SceneTag
	ImageURL                                     string
	ImageSource                                  SceneSource
	URL                                          string
	InLibrary                                    bool
	MatchedStashSceneID                          string
	HasStashSource, HasStashBoxSource            bool
	StashBoxSceneID, StashBoxEndpoint            string
	SourceLabels                                 []string
	StashIDs                                     []StashSceneID
	MojiTask                                     *SceneTask
}
type ScenePerson struct{ ID, Name string }
type SceneTag struct{ ID, Name string }
type SceneTask struct {
	ID                           string
	Stage                        taskruntime.TaskStage
	StageStatus                  taskruntime.TaskStageStatus
	StageLabel, StageStatusLabel string
	Progress                     float64
}
type SceneQuery struct {
	Search         string
	Source         SceneSourceFilter
	InLibrary      LibraryFilter
	Page, PageSize int
}
type ScenePage struct {
	Items                                        []Scene
	Page, PageSize, TotalCount, TotalPages       int
	HasPrevPage, HasNextPage                     bool
	StashSceneCount, StashBoxCount, DedupedCount int
}

type QueueSceneSelection struct {
	Key, SourceSceneID, StashBoxSceneID, StashBoxEndpoint, Code, Title string
	InLibrary                                                          bool
}
type QueueSceneStatus string

const (
	QueueSceneStatusQueued  QueueSceneStatus = "QUEUED"
	QueueSceneStatusSkipped QueueSceneStatus = "SKIPPED"
	QueueSceneStatusFailed  QueueSceneStatus = "FAILED"
)

type QueueSceneResult struct {
	Key                 string
	Status              QueueSceneStatus
	ReasonCode, Message string
	Task                *taskruntime.Task
	ResolvedCode        string
}
type QueueScenesSummary struct{ RequestedCount, QueuedCount, SkippedCount, FailedCount int }
type QueueScenesResult struct {
	QueuedTasks []*taskruntime.Task
	Results     []QueueSceneResult
	Summary     QueueScenesSummary
}

type Detail struct {
	Performer                                                               Performer
	Disambiguation, Birthdate, Ethnicity, Country, EyeColor                 string
	HeightCm, Rating100                                                     *int
	URLs                                                                    []string
	MatchedStashBox                                                         *MatchedStashBox
	TotalSceneCount, StashSceneCount, StashBoxSceneCount, DedupedSceneCount int
}
