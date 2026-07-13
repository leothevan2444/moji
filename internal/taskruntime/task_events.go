package taskruntime

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leothevan2444/moji/internal/logging"
)

type TaskEventType string

const (
	TaskEventCreated TaskEventType = "CREATED"
	TaskEventUpdated TaskEventType = "UPDATED"
	TaskEventDeleted TaskEventType = "DELETED"
)

type TaskStats struct {
	Total        int
	Active       int
	Completed    int
	Downloading  int
	PendingScans int
	Failed       int
}

type TaskEvent struct {
	Sequence       int
	Type           TaskEventType
	TaskID         string
	Task           *Task
	DashboardStats TaskStats
}

type TaskEventSource interface {
	Subscribe(ctx context.Context) <-chan *TaskEvent
}

type TaskEventPublisher interface {
	Publish(event *TaskEvent)
}

type TaskEventBus struct {
	mu          sync.RWMutex
	subscribers map[uint64]taskEventSubscriber
	nextID      atomic.Uint64
	sequence    atomic.Uint64
	published   atomic.Uint64
	dropped     atomic.Uint64
	closed      bool
	bufferSize  int
}

type TaskEventBusStats struct {
	Subscribers int
	Published   uint64
	Dropped     uint64
	Sequence    uint64
}

type taskEventSubscriber struct {
	channel chan *TaskEvent
	done    chan struct{}
}

func NewTaskEventBus(bufferSize int) *TaskEventBus {
	if bufferSize <= 0 {
		bufferSize = 32
	}
	return &TaskEventBus{
		subscribers: make(map[uint64]taskEventSubscriber),
		bufferSize:  bufferSize,
	}
}

func (b *TaskEventBus) Subscribe(ctx context.Context) <-chan *TaskEvent {
	channel := make(chan *TaskEvent, b.bufferSize)
	if ctx == nil {
		ctx = context.Background()
	}

	b.mu.Lock()
	if b.closed {
		close(channel)
		b.mu.Unlock()
		return channel
	}
	id := b.nextID.Add(1)
	subscriber := taskEventSubscriber{channel: channel, done: make(chan struct{})}
	b.subscribers[id] = subscriber
	b.mu.Unlock()
	slog.Info("task events: subscriber connected", "subscriber_id", id)

	go func() {
		select {
		case <-ctx.Done():
			b.unsubscribe(id)
		case <-subscriber.done:
		}
	}()
	return channel
}

func (b *TaskEventBus) unsubscribe(id uint64) {
	b.mu.Lock()
	subscriber, ok := b.subscribers[id]
	if ok {
		delete(b.subscribers, id)
		close(subscriber.done)
		close(subscriber.channel)
	}
	b.mu.Unlock()
	if ok {
		slog.Info("task events: subscriber disconnected", "subscriber_id", id)
	}
}

func (b *TaskEventBus) Publish(event *TaskEvent) {
	if event == nil {
		return
	}
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return
	}
	next := cloneTaskEvent(event)
	next.Sequence = int(b.sequence.Add(1))
	b.published.Add(1)
	for id, subscriber := range b.subscribers {
		select {
		case subscriber.channel <- cloneTaskEvent(next):
		default:
			b.dropped.Add(1)
			slog.Warn("task events: event dropped because subscriber buffer is full", "subscriber_id", id, "sequence", next.Sequence, "task_id", next.TaskID)
		}
	}
	b.mu.RUnlock()

	if next.Type == TaskEventUpdated {
		slog.Debug("task events: event published", "sequence", next.Sequence, "type", next.Type, "task_id", next.TaskID)
	} else {
		slog.Info("task events: event published", "sequence", next.Sequence, "type", next.Type, "task_id", next.TaskID)
	}
}

func (b *TaskEventBus) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	disconnected := len(b.subscribers)
	for id, subscriber := range b.subscribers {
		delete(b.subscribers, id)
		close(subscriber.done)
		close(subscriber.channel)
	}
	b.mu.Unlock()
	stats := b.Stats()
	slog.Info("task events: shutdown summary", "published", stats.Published, "dropped", stats.Dropped, "sequence", stats.Sequence, "disconnected", disconnected)
}

func (b *TaskEventBus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

func (b *TaskEventBus) Stats() TaskEventBusStats {
	b.mu.RLock()
	subscribers := len(b.subscribers)
	b.mu.RUnlock()
	return TaskEventBusStats{Subscribers: subscribers, Published: b.published.Load(), Dropped: b.dropped.Load(), Sequence: b.sequence.Load()}
}

func cloneTaskEvent(event *TaskEvent) *TaskEvent {
	if event == nil {
		return nil
	}
	next := *event
	next.Task = cloneTask(event.Task)
	return &next
}

type EventingTaskStore struct {
	store         TaskStore
	publisher     TaskEventPublisher
	now           func() time.Time
	mu            sync.Mutex
	lastPublished map[string]publishedTaskState
}

type publishedTaskState struct {
	task *Task
	at   time.Time
}

type EventingTaskStoreOption func(*EventingTaskStore)

func WithTaskEventClock(now func() time.Time) EventingTaskStoreOption {
	return func(store *EventingTaskStore) {
		if now != nil {
			store.now = now
		}
	}
}

func NewEventingTaskStore(store TaskStore, publisher TaskEventPublisher, options ...EventingTaskStoreOption) (*EventingTaskStore, error) {
	if store == nil {
		return nil, errors.New("taskruntime: task event store requires a task store")
	}
	if publisher == nil {
		return nil, errors.New("taskruntime: task event store requires a publisher")
	}
	next := &EventingTaskStore{
		store:         store,
		publisher:     publisher,
		now:           time.Now,
		lastPublished: make(map[string]publishedTaskState),
	}
	for _, option := range options {
		option(next)
	}
	return next, nil
}

func (s *EventingTaskStore) Create(ctx context.Context, task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Create(ctx, task); err != nil {
		return err
	}
	s.publishLocked(ctx, TaskEventCreated, task.ID, task)
	return nil
}

func (s *EventingTaskStore) Update(ctx context.Context, task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Update(ctx, task); err != nil {
		return err
	}
	if s.shouldPublishUpdateLocked(task) {
		s.publishLocked(ctx, TaskEventUpdated, task.ID, task)
	}
	return nil
}

func (s *EventingTaskStore) Delete(ctx context.Context, id string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, err := s.store.Delete(ctx, id)
	if err != nil {
		return task, err
	}
	delete(s.lastPublished, id)
	s.publishLocked(ctx, TaskEventDeleted, id, nil)
	return task, nil
}

func (s *EventingTaskStore) Find(ctx context.Context, id string) (*Task, error) {
	return s.store.Find(ctx, id)
}

func (s *EventingTaskStore) FindByCode(ctx context.Context, code string) (*Task, error) {
	return s.store.FindByCode(ctx, code)
}

func (s *EventingTaskStore) FindByTorrentIdentity(ctx context.Context, infoHash string, magnetURI string) (*Task, error) {
	return s.store.FindByTorrentIdentity(ctx, infoHash, magnetURI)
}

func (s *EventingTaskStore) List(ctx context.Context) ([]*Task, error) {
	return s.store.List(ctx)
}

func (s *EventingTaskStore) shouldPublishUpdateLocked(task *Task) bool {
	if task == nil {
		return false
	}
	previous, ok := s.lastPublished[task.ID]
	if !ok || previous.task == nil {
		return true
	}
	if taskChangedBeyondProgress(previous.task, task) {
		return true
	}
	if task.Progress-previous.task.Progress >= 0.01 || previous.task.Progress-task.Progress >= 0.01 {
		return true
	}
	return s.now().Sub(previous.at) >= time.Second
}

func taskChangedBeyondProgress(previous, next *Task) bool {
	if previous == nil || next == nil {
		return previous != next
	}
	left := cloneTask(previous)
	right := cloneTask(next)
	left.Progress, right.Progress = 0, 0
	left.UpdatedAt, right.UpdatedAt = time.Time{}, time.Time{}
	return !reflect.DeepEqual(left, right)
}

func (s *EventingTaskStore) publishLocked(ctx context.Context, eventType TaskEventType, taskID string, task *Task) {
	tasks, err := s.store.List(ctx)
	if err != nil {
		logging.Errorf("task events: calculate dashboard stats after %s task_id=%s: %v", eventType, taskID, err)
		return
	}
	now := s.now()
	if task != nil {
		s.lastPublished[taskID] = publishedTaskState{task: cloneTask(task), at: now}
	}
	s.publisher.Publish(&TaskEvent{
		Type:           eventType,
		TaskID:         taskID,
		Task:           cloneTask(task),
		DashboardStats: CalculateTaskStats(tasks),
	})
}

func CalculateTaskStats(tasks []*Task) TaskStats {
	stats := TaskStats{Total: len(tasks)}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if task.Stage != TaskStageCompleted {
			stats.Active++
		} else {
			stats.Completed++
		}
		if task.Stage == TaskStageDownloading && task.StageStatus == TaskStageStatusRunning {
			stats.Downloading++
		}
		if task.Stage == TaskStagePendingIngest || task.Stage == TaskStageTransferring || task.Stage == TaskStageScanning {
			stats.PendingScans++
		}
		if task.StageStatus == TaskStageStatusBlocked {
			stats.Failed++
		}
	}
	return stats
}
