package subscription

import (
	"context"
	"errors"
	"github.com/Yamashou/gqlgenc/clientv2"
	"github.com/leothevan2444/moji/internal/film"
	"testing"
	"time"
)

func TestCatalogSuccessSeparatesNewFailuresAndHistoricalRetryFailures(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	tasks := &reviewTestTasks{failCode: "FAIL-123"}
	s := refreshTaskService(t, store, tasks, "FAIL-123")
	for check := 0; check < 2; check++ {
		result, err := s.RefreshSubscribedPerformer(ctx, "p1")
		if err != nil || result.LastError != "" {
			t.Fatalf("release failure contaminated catalog result: %+v %v", result, err)
		}
		runs, err := store.LoadCheckRuns(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var latest *CheckRun
		for i := range runs {
			if latest == nil || runs[i].StartedAt.After(latest.StartedAt) {
				latest = &runs[i]
			}
		}
		if latest.Status != "SUCCEEDED" || len(latest.Items) != 1 || latest.Items[0].Details == nil {
			t.Fatalf("missing successful check diagnostics: %+v", latest)
		}
		d := latest.Items[0].Details
		if d.CatalogStatus != "SUCCEEDED" || d.CatalogPages < 1 || d.CatalogScenes < 1 {
			t.Fatalf("missing catalog observations: %+v", d)
		}
		if check == 0 && (d.CreationFailedCount != 1 || d.RetryAttempted != 0) {
			t.Fatalf("new failure retried within same run: %+v", d)
		}
		if check == 1 && (d.CreationFailedCount != 0 || d.RetryTotal != 1 || d.RetryAttempted != 1 || d.RetryFailed != 1 || d.Results[0].Phase != "HISTORICAL_RETRY") {
			t.Fatalf("historical failure not retried and separated: %+v", d)
		}
	}
	if tasks.calls != 2 {
		t.Fatal("expected one attempt per check", tasks.calls)
	}
}

func TestRetryClassificationDoesNotLoopOnIdentityOrAuthenticationFailures(t *testing.T) {
	for _, tc := range []struct {
		err       error
		temporary bool
	}{
		{context.DeadlineExceeded, true}, {context.Canceled, false}, {film.ErrRequired, false}, {errors.New("invalid metadata"), false},
		{&clientv2.ErrorResponse{NetworkError: &clientv2.HTTPError{Code: 503, Message: "private response body"}}, true},
		{&clientv2.ErrorResponse{NetworkError: &clientv2.HTTPError{Code: 401, Message: "private response body"}}, false},
	} {
		if temporaryReleaseFailure(tc.err) != tc.temporary {
			t.Fatalf("wrong retry classification: %v", tc.err)
		}
	}
}

func TestManagedRetryDeadlineIsDurableAndDueQueryIsBounded(t *testing.T) {
	store, _ := managedSubscriptionStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	past := now.Add(-time.Minute)
	future := now.Add(time.Hour)
	for i, key := range []string{"one", "two", "future", "rejected"} {
		r := reviewTestRecord(key, ReviewFailed)
		r.NextRetryAt = &past
		r.FailureCode = "UPSTREAM_TEMPORARY"
		r.RetryAttempts = 2
		if i == 2 {
			r.NextRetryAt = &future
		}
		if i == 3 {
			r.Status = ReviewRejected
		}
		if err := store.PutReleaseReviews(ctx, []ReleaseReview{r}); err != nil {
			t.Fatal(err)
		}
	}
	loaded, err := store.GetReleaseReview(ctx, "one")
	if err != nil || loaded.NextRetryAt == nil || !loaded.NextRetryAt.Equal(past) || loaded.RetryAttempts != 2 {
		t.Fatalf("retry checkpoint lost: %+v %v", loaded, err)
	}
	rows, err := store.DueReleaseRetries(ctx, now, 1)
	if err != nil || len(rows) != 1 || rows[0].Status != ReviewFailed {
		t.Fatalf("due query not bounded: %+v %v", rows, err)
	}
	rows, err = store.DueReleaseRetries(ctx, now, 8)
	if err != nil || len(rows) != 2 {
		t.Fatalf("future or rejected retry selected: %+v %v", rows, err)
	}
}

func TestTransientReleaseRetriesBackOffAndReconcileAcceptedFilm(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	tasks := &racingRefreshTasks{queueErr: context.DeadlineExceeded}
	s := refreshTaskService(t, store, tasks, "URE-114")
	clock := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return clock }
	if _, err := s.RefreshSubscribedPerformer(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	rows, _ := store.ListReleaseReviews(ctx)
	r := rows[0]
	if r.NextRetryAt == nil || !r.NextRetryAt.Equal(clock.Add(time.Minute)) || r.FailureCode != "UPSTREAM_TEMPORARY" {
		t.Fatalf("transient failure has no durable retry: %+v", r)
	}
	if err := s.ProcessDueReviewRetries(ctx); err != nil || tasks.calls != 1 {
		t.Fatal("retried before deadline", err, tasks.calls)
	}
	clock = clock.Add(time.Minute)
	if err := s.ProcessDueReviewRetries(ctx); err != nil {
		t.Fatal(err)
	}
	rows, _ = store.ListReleaseReviews(ctx)
	r = rows[0]
	if tasks.calls != 2 || r.RetryAttempts != 2 || r.NextRetryAt == nil || !r.NextRetryAt.Equal(clock.Add(5*time.Minute)) {
		t.Fatalf("backoff did not grow: %+v calls=%d", r, tasks.calls)
	}
	clock = clock.Add(5 * time.Minute)
	tasks.queueErr = nil
	if err := s.ProcessDueReviewRetries(ctx); err != nil {
		t.Fatal(err)
	}
	rows, _ = store.ListReleaseReviews(ctx)
	r = rows[0]
	if tasks.calls != 3 || r.Status != ReviewDownloaded || r.TaskID != "concurrent-task" || r.NextRetryAt != nil || r.Error != "" {
		t.Fatalf("retry did not reconcile accepted task: %+v", r)
	}
	if err := s.ProcessDueReviewRetries(ctx); err != nil || tasks.calls != 3 {
		t.Fatal("accepted film was retried", err)
	}
}

func TestManualFailuresRejectionsAndUnsubscribeDoNotRetryInBackground(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	tasks := &reviewTestTasks{}
	s := refreshTaskService(t, store, tasks, "URE-114")
	for _, status := range []ReviewStatus{ReviewFailed, ReviewRejected, ReviewSkipped} {
		r := reviewTestRecord(string(status), status)
		r.Code = "URE-114"
		if status == ReviewFailed {
			s.failReleaseReview(&r, context.DeadlineExceeded)
		}
		if status != ReviewFailed {
			due := s.now().Add(-time.Hour)
			r.NextRetryAt = &due
		}
		if err := store.PutReleaseReviews(ctx, []ReleaseReview{r}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.ProcessDueReviewRetries(ctx); err != nil || tasks.calls != 0 {
		t.Fatal("retried rejected, skipped, or not-due records", err)
	}
	rows, _ := store.ListReleaseReviews(ctx)
	for _, r := range rows {
		if r.Status == ReviewFailed {
			due := s.now().Add(-time.Hour)
			r.NextRetryAt = &due
			store.PutReleaseReviews(ctx, []ReleaseReview{r})
		}
	}
	s.stash.(*fakeStashClient).performers["p1"].CustomFields[DefaultCustomFieldKey] = false
	if err := s.ProcessDueReviewRetries(ctx); err != nil || tasks.calls != 0 {
		t.Fatal("created task for unsubscribed actor", err)
	}
}
