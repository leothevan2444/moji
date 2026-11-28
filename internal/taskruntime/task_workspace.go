package taskruntime

import (
	"context"
	"errors"
	"fmt"
	"github.com/leothevan2444/moji/internal/ordering"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrTaskNotFound = errors.New("taskruntime: task not found")

type TaskPageInput struct {
	Search, Status, Sort, Direction string
	Page, PageSize                  int
	CompletedWithin24Hours          bool
	IDs                             []string
}
type TaskFilterCounts struct {
	Total     int `db:"total"`
	Active    int `db:"active"`
	Waiting   int `db:"waiting"`
	Completed int `db:"completed"`
	Attention int `db:"attention"`
}

type TaskPage struct {
	Items                                  []*Task
	TotalCount, Page, PageSize, TotalPages int
	Sequence                               int
	StreamID                               string
	FilterCounts                           TaskFilterCounts
	Stats                                  TaskStats
}
type HomeTaskSnapshot struct {
	HomeOverview
	Sequence int
	StreamID string
}
type TaskDetailSnapshot struct {
	Task     *Task
	Sequence int
	StreamID string
}

func (s *EventingTaskStore) TaskDetailSnapshot(ctx context.Context, id string) (TaskDetailSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, err := s.store.Find(ctx, id)
	if errors.Is(err, ErrTaskNotFound) {
		err = nil
	}
	result := TaskDetailSnapshot{Task: task}
	if bus, ok := s.publisher.(*TaskEventBus); ok {
		result.Sequence, result.StreamID = bus.Cursor()
	}
	return result, err
}
func (s *Service) TaskDetailSnapshot(ctx context.Context, id string) (TaskDetailSnapshot, error) {
	if p, ok := s.store.(interface {
		TaskDetailSnapshot(context.Context, string) (TaskDetailSnapshot, error)
	}); ok {
		return p.TaskDetailSnapshot(ctx, id)
	}
	task, err := s.FindTask(ctx, id)
	if errors.Is(err, ErrTaskNotFound) {
		err = nil
	}
	return TaskDetailSnapshot{Task: task}, err
}

type taskPageStore interface {
	TaskPage(context.Context, TaskPageInput) (TaskPage, error)
}

func attentionHeaders(ctx context.Context, store TaskStore) ([]*Task, error) {
	if relational, ok := store.(*RelationalTaskStore); ok {
		var rows []sqliteTaskRow
		if err := relational.core.Reader.SelectContext(ctx, &rows, taskSummarySelect+` WHERE t.stage<>'COMPLETED' AND t.stage_status='BLOCKED' ORDER BY t.created_at,t.id`); err != nil {
			return nil, err
		}
		result := make([]*Task, 0, len(rows))
		for _, row := range rows {
			task, err := row.toTask()
			if err != nil {
				return nil, err
			}
			task = relational.overlay(task)
			refreshTaskStageFields(task)
			result = append(result, task)
		}
		return result, nil
	}
	tasks, err := store.List(ctx)
	if err != nil {
		return nil, err
	}
	result := []*Task{}
	for _, task := range tasks {
		if taskWorkspaceGroup(task) == "ATTENTION" {
			result = append(result, task)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}
func (s *EventingTaskStore) AttentionQueue(ctx context.Context) (TaskPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := attentionHeaders(ctx, s.store)
	result := TaskPage{Items: items}
	if bus, ok := s.publisher.(*TaskEventBus); ok {
		result.Sequence, result.StreamID = bus.Cursor()
	}
	return result, err
}
func (s *Service) AttentionQueue(ctx context.Context) (TaskPage, error) {
	if p, ok := s.store.(interface {
		AttentionQueue(context.Context) (TaskPage, error)
	}); ok {
		return p.AttentionQueue(ctx)
	}
	items, err := attentionHeaders(ctx, s.store)
	return TaskPage{Items: items}, err
}

func normalizeTaskPage(in TaskPageInput) (TaskPageInput, error) {
	if in.Page < 1 {
		in.Page = 1
	}
	if in.PageSize == 0 {
		in.PageSize = 24
	}
	if in.PageSize < 1 || in.PageSize > 100 || len(in.IDs) > 100 {
		return in, fmt.Errorf("taskruntime: invalid task page size")
	}
	in.Status = strings.ToUpper(in.Status)
	in.Sort = strings.ToUpper(in.Sort)
	if in.Status == "" {
		in.Status = "ALL"
	}
	if in.Sort == "" {
		in.Sort = "CREATED"
	}
	switch in.Status {
	case "ALL", "ATTENTION", "ACTIVE", "WAITING", "COMPLETED":
	default:
		return in, fmt.Errorf("taskruntime: invalid task status")
	}
	switch in.Sort {
	case "TITLE", "RELEASE_DATE", "CREATED", "UPDATED", "PROGRESS", "OLDEST", "INGESTED":
	default:
		return in, fmt.Errorf("taskruntime: invalid task sort")
	}
	in.Direction = strings.ToUpper(in.Direction)
	if in.Direction == "" {
		in.Direction = "DESC"
	}
	if in.Direction != "ASC" && in.Direction != "DESC" {
		return in, fmt.Errorf("taskruntime: invalid sort direction")
	}
	in.Search = strings.ToLower(strings.TrimSpace(in.Search))
	return in, nil
}

func taskWorkspaceGroup(t *Task) string {
	if t.Stage == TaskStageCompleted {
		return "COMPLETED"
	}
	if t.StageStatus == TaskStageStatusBlocked {
		return "ATTENTION"
	}
	if t.Stage == TaskStageSourcing && t.StageStatus == TaskStageStatusPending && t.SourcingNextCheckAt != nil {
		return "WAITING"
	}
	return "ACTIVE"
}

// The production projection never reads attachments, file plans or checkpoint JSON.
const taskSummarySelect = `SELECT t.id,t.source,t.code,t.stage,t.stage_status,t.stage_error_code,t.stage_error_message,
 json_extract(t.film_json,'$.Date') AS film_release_date,t.revision,t.created_at,t.updated_at,k.sourcing_last_checked_at,k.sourcing_next_check_at,
 k.sourcing_completed_at,k.download_completed_at,k.ingest_completed_at,i.stash_ingest_phase,CASE WHEN json_extract(i.stash_scene_ids_json,'$[0]') IS NOT NULL THEN json_array(json_extract(i.stash_scene_ids_json,'$[0]')) END AS stash_scene_ids_json,d.torrent_url,d.torrent_hash,d.content_path,i.stash_scan_job_id
 FROM tasks t LEFT JOIN task_checkpoints k ON k.task_id=t.id LEFT JOIN task_ingest i ON i.task_id=t.id LEFT JOIN task_downloads d ON d.task_id=t.id`

func (s *RelationalTaskStore) FindSummary(ctx context.Context, id string) (*Task, error) {
	var row sqliteTaskRow
	if err := s.core.Reader.GetContext(ctx, &row, taskSummarySelect+" WHERE t.id=?", id); err != nil {
		return nil, err
	}
	task, err := row.toTask()
	if err != nil {
		return nil, err
	}
	task = s.overlay(task)
	refreshTaskStageFields(task)
	return task, nil
}

func (s *RelationalTaskStore) TaskPage(ctx context.Context, input TaskPageInput) (TaskPage, error) {
	in, err := normalizeTaskPage(input)
	if err != nil {
		return TaskPage{}, err
	}
	where := ` WHERE 1=1`
	args := []any{}
	waiting := `t.stage='SOURCING' AND t.stage_status='PENDING' AND k.sourcing_next_check_at IS NOT NULL`
	if in.Sort == "INGESTED" {
		where += ` AND k.ingest_completed_at IS NOT NULL`
	}
	if in.CompletedWithin24Hours {
		where += ` AND t.stage='COMPLETED' AND julianday(k.ingest_completed_at) BETWEEN julianday(?) AND julianday(?)`
		now := time.Now().UTC()
		args = append(args, now.Add(-24*time.Hour).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	}
	if in.IDs != nil {
		if len(in.IDs) == 0 {
			where += ` AND 0=1`
		} else {
			var e error
			where, args, e = sqlx.In(where+` AND t.id IN (?)`, append(args, in.IDs)...)
			if e != nil {
				return TaskPage{}, e
			}
		}
	}
	scopeWhere := where
	switch in.Status {
	case "ATTENTION":
		where += ` AND t.stage<>'COMPLETED' AND t.stage_status='BLOCKED'`
	case "WAITING":
		where += ` AND (` + waiting + `)`
	case "ACTIVE":
		where += ` AND t.stage<>'COMPLETED' AND t.stage_status<>'BLOCKED' AND NOT (` + waiting + `)`
	case "COMPLETED":
		where += ` AND t.stage='COMPLETED'`
	}
	result := TaskPage{Items: []*Task{}, Page: in.Page, PageSize: in.PageSize}
	result.Stats, err = s.TaskStats(ctx)
	if err != nil {
		return result, err
	}
	result.FilterCounts = TaskFilterCounts{Total: result.Stats.Total, Completed: result.Stats.Completed, Attention: result.Stats.Attention, Waiting: result.Stats.Waiting}
	result.FilterCounts.Active = result.FilterCounts.Total - result.FilterCounts.Completed - result.FilterCounts.Attention - result.FilterCounts.Waiting
	if in.Search == "" && (in.CompletedWithin24Hours || in.IDs != nil || in.Sort == "INGESTED") {
		// Only aggregates are needed here; keep ordinary SQL-paginated reads bounded.
		countSQL := `SELECT COUNT(*) AS total,
   COALESCE(SUM(CASE WHEN t.stage='COMPLETED' THEN 1 ELSE 0 END),0) AS completed,
   COALESCE(SUM(CASE WHEN t.stage<>'COMPLETED' AND t.stage_status='BLOCKED' THEN 1 ELSE 0 END),0) AS attention,
   COALESCE(SUM(CASE WHEN ` + waiting + ` THEN 1 ELSE 0 END),0) AS waiting
   FROM tasks t LEFT JOIN task_checkpoints k ON k.task_id=t.id` + scopeWhere
		if err = s.core.Reader.GetContext(ctx, &result.FilterCounts, countSQL, args...); err != nil {
			return result, err
		}
		result.FilterCounts.Active = result.FilterCounts.Total - result.FilterCounts.Completed - result.FilterCounts.Attention - result.FilterCounts.Waiting
	}
	// Observation-driven search/sorts use only lightweight headers and the current
	// in-memory observation overlay, never stale persisted progress or full tasks.
	dynamic := in.Search != "" || in.Sort == "UPDATED" || in.Sort == "PROGRESS" || in.Sort == "TITLE"
	var rows []sqliteTaskRow
	if dynamic {
		queryWhere := where
		if in.Search != "" {
			queryWhere = scopeWhere
		}
		err = s.core.Reader.SelectContext(ctx, &rows, taskSummarySelect+queryWhere, args...)
	} else {
		err = s.core.Reader.GetContext(ctx, &result.TotalCount, `SELECT count(*) FROM tasks t LEFT JOIN task_checkpoints k ON k.task_id=t.id`+where, args...)
		if err != nil {
			return result, err
		}
		result.TotalPages = max(1, (result.TotalCount+in.PageSize-1)/in.PageSize)
		result.Page = min(in.Page, result.TotalPages)
		direction := in.Direction
		order := `t.created_at ` + direction + `,t.id ASC`
		if in.Sort == "RELEASE_DATE" {
			order = `(NULLIF(TRIM(json_extract(t.film_json,'$.Date')),'') IS NULL) ASC,NULLIF(TRIM(json_extract(t.film_json,'$.Date')),'') ` + direction + `,t.created_at DESC,t.id ASC`
		}
		if in.Sort == "OLDEST" {
			order = `t.created_at ASC,t.id ASC`
		}
		if in.Sort == "INGESTED" {
			order = `k.ingest_completed_at DESC,t.id ASC`
		}
		pageArgs := append(append([]any{}, args...), in.PageSize, (result.Page-1)*in.PageSize)
		err = s.core.Reader.SelectContext(ctx, &rows, taskSummarySelect+where+` ORDER BY `+order+` LIMIT ? OFFSET ?`, pageArgs...)
	}
	if err != nil {
		return result, err
	}
	for _, row := range rows {
		task, e := row.toTask()
		if e != nil {
			return result, e
		}
		task = s.overlay(task)
		refreshTaskStageFields(task)
		result.Items = append(result.Items, task)
	}
	if dynamic {
		page := paginateTaskHeaders(result.Items, in, result.Stats)
		if in.Search == "" {
			page.FilterCounts = result.FilterCounts
		}
		return page, nil
	}
	return result, nil
}

// Scope predicates are shared by the result list and the counts, excluding only status.
func taskMatchesPageScope(task *Task, in TaskPageInput, ids map[string]bool, now time.Time) bool {
	if in.Sort == "INGESTED" && task.IngestCompletedAt == nil {
		return false
	}
	if in.IDs != nil && !ids[task.ID] {
		return false
	}
	if in.CompletedWithin24Hours && (task.Stage != TaskStageCompleted || task.IngestCompletedAt == nil || task.IngestCompletedAt.Before(now.Add(-24*time.Hour)) || task.IngestCompletedAt.After(now)) {
		return false
	}
	return in.Search == "" || strings.Contains(strings.ToLower(strings.Join([]string{task.ID, task.Code, task.TorrentName, string(task.Stage), string(task.StageStatus), task.StageLabel, task.StageStatusLabel, task.StageErrorCode, task.StageErrorMessage, task.QBittorrentState, task.TorrentHash, task.ContentPath, task.TorrentURL, task.StashScanJobID}, " ")), in.Search)
}

func paginateTaskHeaders(tasks []*Task, in TaskPageInput, stats TaskStats) TaskPage {
	items := make([]*Task, 0, len(tasks))
	ids := map[string]bool{}
	for _, id := range in.IDs {
		ids[id] = true
	}
	now := time.Now()
	counts := TaskFilterCounts{}
	for _, task := range tasks {
		if !taskMatchesPageScope(task, in, ids, now) {
			continue
		}
		counts.Total++
		switch taskWorkspaceGroup(task) {
		case "ACTIVE":
			counts.Active++
		case "WAITING":
			counts.Waiting++
		case "COMPLETED":
			counts.Completed++
		case "ATTENTION":
			counts.Attention++
		}
		if in.Status == "ALL" || taskWorkspaceGroup(task) == in.Status {
			items = append(items, task)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		comparison := 0
		switch in.Sort {
		case "TITLE":
			comparison = ordering.Compare(taskSortTitle(a), taskSortTitle(b))
		case "PROGRESS":
			if a.Progress < b.Progress {
				comparison = -1
			} else if a.Progress > b.Progress {
				comparison = 1
			}
		case "UPDATED":
			comparison = a.UpdatedAt.Compare(b.UpdatedAt)
		case "RELEASE_DATE":
			left, right := a.FilmReleaseDate(), b.FilmReleaseDate()
			if (left == "") != (right == "") {
				return left != ""
			}
			comparison = strings.Compare(left, right)
		case "INGESTED":
			if a.IngestCompletedAt != nil && b.IngestCompletedAt != nil {
				comparison = a.IngestCompletedAt.Compare(*b.IngestCompletedAt)
			}
		default:
			comparison = a.CreatedAt.Compare(b.CreatedAt)
		}
		if comparison != 0 {
			if in.Direction == "ASC" || in.Sort == "OLDEST" {
				return comparison < 0
			}
			return comparison > 0
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}
		return a.ID < b.ID
	})

	total := len(items)
	pages := max(1, (total+in.PageSize-1)/in.PageSize)
	page := min(in.Page, pages)
	start := (page - 1) * in.PageSize
	return TaskPage{Items: items[start:min(total, start+in.PageSize)], TotalCount: total, Page: page, PageSize: in.PageSize, TotalPages: pages, Stats: stats, FilterCounts: counts}
}

func readTaskPage(ctx context.Context, store TaskStore, in TaskPageInput) (TaskPage, error) {
	in, err := normalizeTaskPage(in)
	if err != nil {
		return TaskPage{}, err
	}
	if p, ok := store.(taskPageStore); ok {
		return p.TaskPage(ctx, in)
	}
	tasks, err := store.List(ctx)
	if err != nil {
		return TaskPage{}, err
	}
	return paginateTaskHeaders(tasks, in, CalculateTaskStats(tasks)), nil
}
func (s *EventingTaskStore) TaskPage(ctx context.Context, in TaskPageInput) (TaskPage, error) {
	// Serialize the snapshot boundary with commits and publication, including progress.
	s.mu.Lock()
	defer s.mu.Unlock()
	page, err := readTaskPage(ctx, s.store, in)
	if bus, ok := s.publisher.(*TaskEventBus); ok {
		page.Sequence, page.StreamID = bus.Cursor()
	}
	return page, err
}
func (s *Service) TaskPage(ctx context.Context, in TaskPageInput) (TaskPage, error) {
	return readTaskPage(ctx, s.store, in)
}
func (s *Service) TaskStats(ctx context.Context) (TaskStats, error) {
	if p, ok := s.store.(TaskStatsProvider); ok {
		return p.TaskStats(ctx)
	}
	tasks, err := s.store.List(ctx)
	return CalculateTaskStats(tasks), err
}

func readHomeTaskSnapshot(ctx context.Context, store TaskStore) (HomeTaskSnapshot, error) {
	result := HomeTaskSnapshot{}
	stats, err := readTaskPage(ctx, store, TaskPageInput{PageSize: 3, Status: "ATTENTION", Sort: "OLDEST"})
	if err != nil {
		return result, err
	}
	result.Total, result.Attention, result.Waiting = stats.Stats.Total, stats.Stats.Attention, stats.Stats.Waiting
	result.Active = stats.Stats.Total - stats.Stats.Completed - stats.Stats.Attention - stats.Stats.Waiting
	result.AttentionTasks = stats.Items
	active, err := readTaskPage(ctx, store, TaskPageInput{PageSize: 5, Status: "ACTIVE", Sort: "OLDEST"})
	if err != nil {
		return result, err
	}
	result.ActiveTasks = active.Items
	recent, err := readTaskPage(ctx, store, TaskPageInput{PageSize: 3, Status: "COMPLETED", Sort: "INGESTED"})
	if err != nil {
		return result, err
	}
	result.RecentTasks = recent.Items
	completed, err := readTaskPage(ctx, store, TaskPageInput{PageSize: 1, CompletedWithin24Hours: true})
	result.CompletedLast24Hours = completed.TotalCount
	return result, err
}
func (s *EventingTaskStore) HomeTaskSnapshot(ctx context.Context) (HomeTaskSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := readHomeTaskSnapshot(ctx, s.store)
	if bus, ok := s.publisher.(*TaskEventBus); ok {
		result.Sequence, result.StreamID = bus.Cursor()
	}
	return result, err
}
func (s *Service) HomeTaskSnapshot(ctx context.Context) (HomeTaskSnapshot, error) {
	if p, ok := s.store.(interface {
		HomeTaskSnapshot(context.Context) (HomeTaskSnapshot, error)
	}); ok {
		return p.HomeTaskSnapshot(ctx)
	}
	return readHomeTaskSnapshot(ctx, s.store)
}

// FilmReleaseDate reads the persisted snapshot or its lightweight header projection.
func (t *Task) FilmReleaseDate() string {
	if t.Film != nil {
		return strings.TrimSpace(t.Film.Date)
	}
	return strings.TrimSpace(t.filmReleaseDate)
}
func taskSortTitle(t *Task) string {
	if title := strings.TrimSpace(t.Code); title != "" {
		return title
	}
	if title := strings.TrimSpace(t.TorrentName); title != "" {
		return title
	}
	return t.ID
}
