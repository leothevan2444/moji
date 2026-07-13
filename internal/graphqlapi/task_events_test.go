package graphqlapi

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/taskruntime"
)

func TestTaskEventsResolverMapsEventsAndStopsOnCancellation(t *testing.T) {
	bus := taskruntime.NewTaskEventBus(2)
	defer bus.Close()
	resolver := &subscriptionResolver{Resolver: &Resolver{TaskEventSource: bus}}
	ctx, cancel := context.WithCancel(context.Background())
	channel, err := resolver.TaskEvents(ctx)
	if err != nil {
		t.Fatalf("TaskEvents: %v", err)
	}
	task := &taskruntime.Task{ID: "task-1", Stage: taskruntime.TaskStageDownloading, StageStatus: taskruntime.TaskStageStatusRunning}
	bus.Publish(&taskruntime.TaskEvent{
		Type:           taskruntime.TaskEventUpdated,
		TaskID:         task.ID,
		Task:           task,
		DashboardStats: taskruntime.TaskStats{Total: 1, Active: 1, Downloading: 1},
	})
	select {
	case event := <-channel:
		if event.Sequence != 1 || event.TaskID != task.ID || event.Task == nil || event.Task.ID != task.ID || event.DashboardStats.Downloading != 1 {
			t.Fatalf("unexpected mapped event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("resolver did not forward event")
	}
	cancel()
	select {
	case _, ok := <-channel:
		if ok {
			t.Fatal("expected resolver channel to close")
		}
	case <-time.After(time.Second):
		t.Fatal("resolver did not stop after cancellation")
	}
}

func TestSameOriginWebSocketRequest(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		origin string
		want   bool
	}{
		{name: "same origin", host: "moji.example:8443", origin: "https://moji.example:8443", want: true},
		{name: "case insensitive", host: "MOJI.EXAMPLE", origin: "https://moji.example", want: true},
		{name: "cross origin", host: "moji.example", origin: "https://attacker.example", want: false},
		{name: "different port", host: "moji.example:8443", origin: "https://moji.example", want: false},
		{name: "missing origin", host: "moji.example", origin: "", want: true},
		{name: "invalid origin", host: "moji.example", origin: "://bad", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "http://"+test.host+"/graphql", nil)
			request.Host = test.host
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			if got := SameOriginWebSocketRequest(request); got != test.want {
				t.Fatalf("SameOriginWebSocketRequest() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestTaskEventsOverGraphQLTransportWebSocket(t *testing.T) {
	bus := taskruntime.NewTaskEventBus(2)
	defer bus.Close()
	client := newGraphQLWebSocketTestClient(t, &Resolver{TaskEventSource: bus})
	client.subscribe(t, "task-events", `subscription { taskEvents { sequence type taskId task { id } dashboardStats { total active } } }`)
	waitForSubscriber(t, bus.SubscriberCount)
	bus.Publish(&taskruntime.TaskEvent{
		Type:           taskruntime.TaskEventCreated,
		TaskID:         "task-1",
		Task:           &taskruntime.Task{ID: "task-1", Stage: taskruntime.TaskStageDownloading, StageStatus: taskruntime.TaskStageStatusRunning},
		DashboardStats: taskruntime.TaskStats{Total: 1, Active: 1},
	})

	var payload struct {
		Data struct {
			TaskEvents struct {
				Sequence int    `json:"sequence"`
				Type     string `json:"type"`
				TaskID   string `json:"taskId"`
			} `json:"taskEvents"`
		} `json:"data"`
	}
	client.readNext(t, "task-events", &payload)
	if payload.Data.TaskEvents.Sequence != 1 || payload.Data.TaskEvents.Type != "CREATED" || payload.Data.TaskEvents.TaskID != "task-1" {
		t.Fatalf("unexpected subscription payload: %#v", payload)
	}
}
