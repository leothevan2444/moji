package subscription

import "time"

type ReviewStatus string

const (
	ReviewPending    ReviewStatus = "PENDING"
	ReviewDownloaded ReviewStatus = "DOWNLOADED"
	ReviewRejected   ReviewStatus = "REJECTED"
	ReviewSkipped    ReviewStatus = "SKIPPED"
	ReviewFailed     ReviewStatus = "FAILED"
	ReviewInLibrary  ReviewStatus = "IN_LIBRARY"
)

type ReviewPerformer struct {
	ID, Name     string
	PendingCount int
}

type ReleaseReview struct {
	RecordedRelease
	SceneID    string
	ImageURL   string
	Status     ReviewStatus
	Performers []ReviewPerformer
	ReviewedAt *time.Time
	Error      string
}

type ReleaseReviewQuery struct {
	Search, PerformerID string
	Status              ReviewStatus
	VR                  *bool
	Page, PageSize      int
}

type ReleaseReviewPage struct {
	Items                                                             []ReleaseReview
	Performers                                                        []ReviewPerformer
	TotalCount, PendingCount, FailedCount, Page, PageSize, TotalPages int
}

type ReviewAction string

const (
	ReviewDownload   ReviewAction = "DOWNLOAD"
	ReviewReject     ReviewAction = "REJECT"
	ReviewReconsider ReviewAction = "RECONSIDER"
)

type ReleaseReviewResult struct {
	Key, Outcome, ReasonCode string
	Release                  *ReleaseReview
}

type ReleaseReviewPayload struct {
	Results                                   []ReleaseReviewResult
	SucceededCount, SkippedCount, FailedCount int
}
