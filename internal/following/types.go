package following

import "time"

const DefaultCustomFieldKey = "moji_following"

type Performer struct {
	ID         string
	Name       string
	AliasList  []string
	Favorite   bool
	SceneCount int
	Followed   bool
}

type Release struct {
	Key    string
	Source string
	Title  string
	Code   string
	Date   string
	URL    string
	Query  string
}

type RecordedRelease struct {
	Key    string    `json:"key"`
	Source string    `json:"source"`
	Title  string    `json:"title"`
	Code   string    `json:"code,omitempty"`
	Date   string    `json:"date,omitempty"`
	URL    string    `json:"url,omitempty"`
	Query  string    `json:"query"`
	TaskID string    `json:"task_id,omitempty"`
	SeenAt time.Time `json:"seen_at"`
}

type PerformerState struct {
	PerformerID       string            `json:"performer_id"`
	LastCheckedAt     *time.Time        `json:"last_checked_at,omitempty"`
	LastError         string            `json:"last_error,omitempty"`
	ProcessedReleases []RecordedRelease `json:"processed_releases,omitempty"`
	PendingReleases   []RecordedRelease `json:"pending_releases,omitempty"`
}

type FollowingPerformer struct {
	Performer             Performer
	LastCheckedAt         *time.Time
	LastError             string
	PendingReleaseCount   int
	ProcessedReleaseCount int
	RecentReleases        []RecordedRelease
}
