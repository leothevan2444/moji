package taskruntime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTaskEventBusPublishesToMultipleSubscribers(t *testing.T) {
	bus := NewTaskEventBus(2)
	defer bus.Close()
	first := bus.Subscribe(context.Background())
	second := bus.Subscribe(context.Background())

	bus.Publish(&TaskEvent{Type: TaskEventCreated, TaskID: "task-1"})
	for index, channel := range []<-chan *TaskEvent{first, second} {
		select {
		case event := <-channel:
			if event.Sequence != 1 || event.Type != TaskEventCreated || event.TaskID != "task-1" {
				t.Fatalf("subscriber %d received unexpected event: %#v", index, event)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d did not receive event", index)
		}
	}
}

func TestTaskEventBusCancellationUnsubscribesAndClosesChannel(t *testing.T) {
	bus := NewTaskEventBus(1)
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	channel := bus.Subscribe(ctx)
	cancel()

	select {
	case _, ok := <-channel:
		if ok {
			t.Fatal("expected subscriber channel to close")
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber was not removed after cancellation")
	}
}

func TestTaskEventBusDropsForSlowSubscriberWithoutBlocking(t *testing.T) {
	bus := NewTaskEventBus(1)
	defer bus.Close()
	channel := bus.Subscribe(context.Background())
	bus.Publish(&TaskEvent{Type: TaskEventUpdated, TaskID: "task-1"})

	done := make(chan struct{})
	go func() {
		bus.Publish(&TaskEvent{Type: TaskEventUpdated, TaskID: "task-1"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Publish blocked on a full subscriber channel")
	}

	first := <-channel
	bus.Publish(&TaskEvent{Type: TaskEventUpdated, TaskID: "task-1"})
	third := <-channel
	if first.Sequence != 1 || third.Sequence != 3 {
		t.Fatalf("expected monotonic sequence with a visible gap, got %d then %d", first.Sequence, third.Sequence)
	}
}

func TestTaskEventBusCloseIsIdempotentAndPublishSafe(t *testing.T) {
	bus := NewTaskEventBus(1)
	channel := bus.Subscribe(context.Background())
	bus.Close()
	bus.Close()
	bus.Publish(&TaskEvent{Type: TaskEventDeleted, TaskID: "task-1"})
	if _, ok := <-channel; ok {
		t.Fatal("expected channel to be closed")
	}
}

type collectingTaskEventPublisher struct {
	events []*TaskEvent
}

func (p *collectingTaskEventPublisher) Publish(event *TaskEvent) {
	p.events = append(p.events, cloneTaskEvent(event))
}

func TestEventingTaskStorePublishesCommittedChangesAndThrottlesProgress(t *testing.T) {
	now := time.Unix(100, 0)
	base := NewMemoryTaskStore()
	publisher := &collectingTaskEventPublisher{}
	store, err := NewEventingTaskStore(base, publisher, WithTaskEventClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewEventingTaskStore: %v", err)
	}
	task := &Task{ID: "task-1", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, CreatedAt: now, UpdatedAt: now}
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(publisher.events) != 1 || publisher.events[0].Type != TaskEventCreated || publisher.events[0].DashboardStats.Total != 1 {
		t.Fatalf("unexpected create event: %#v", publisher.events)
	}

	task.Progress = 0.005
	task.UpdatedAt = now.Add(100 * time.Millisecond)
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatalf("Update small progress: %v", err)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("small progress update should be throttled, got %d events", len(publisher.events))
	}

	task.Progress = 0.02
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatalf("Update significant progress: %v", err)
	}
	if len(publisher.events) != 2 || publisher.events[1].Type != TaskEventUpdated {
		t.Fatalf("expected significant progress event, got %#v", publisher.events)
	}

	task.StageStatus = TaskStageStatusBlocked
	task.StageErrorCode = "FAILED"
	if err := store.Update(context.Background(), task); err != nil {
		t.Fatalf("Update critical state: %v", err)
	}
	if len(publisher.events) != 3 || publisher.events[2].DashboardStats.Failed != 1 {
		t.Fatalf("expected immediate critical update with stats, got %#v", publisher.events)
	}

	deleted, err := store.Delete(context.Background(), task.ID)
	if err != nil || deleted == nil {
		t.Fatalf("Delete: task=%#v err=%v", deleted, err)
	}
	last := publisher.events[len(publisher.events)-1]
	if last.Type != TaskEventDeleted || last.Task != nil || last.TaskID != task.ID || last.DashboardStats.Total != 0 {
		t.Fatalf("unexpected delete event: %#v", last)
	}
}

func TestEventingTaskStorePublishesProgressAfterOneSecond(t *testing.T) {
	now := time.Unix(100, 0)
	publisher := &collectingTaskEventPublisher{}
	store, _ := NewEventingTaskStore(NewMemoryTaskStore(), publisher, WithTaskEventClock(func() time.Time { return now }))
	task := &Task{ID: "task-1", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, CreatedAt: now, UpdatedAt: now}
	_ = store.Create(context.Background(), task)
	now = now.Add(time.Second)
	task.Progress = 0.001
	task.UpdatedAt = now
	_ = store.Update(context.Background(), task)
	if len(publisher.events) != 2 {
		t.Fatalf("expected elapsed-time progress event, got %d", len(publisher.events))
	}
}

type failingTaskStore struct{ TaskStore }

func (f failingTaskStore) Create(context.Context, *Task) error { return errors.New("persist failed") }

func TestEventingTaskStoreDoesNotPublishFailedPersistence(t *testing.T) {
	publisher := &collectingTaskEventPublisher{}
	store, err := NewEventingTaskStore(failingTaskStore{}, publisher)
	if err != nil {
		t.Fatalf("NewEventingTaskStore: %v", err)
	}
	if err := store.Create(context.Background(), &Task{ID: "task-1"}); err == nil {
		t.Fatal("expected persistence failure")
	}
	if len(publisher.events) != 0 {
		t.Fatalf("failed persistence published %d events", len(publisher.events))
	}
}
