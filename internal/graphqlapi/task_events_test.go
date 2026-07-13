package graphqlapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
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
	resolver := &Resolver{TaskEventSource: bus}
	server := httptest.NewServer(NewGraphQLServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver})))
	defer server.Close()

	dialer := websocket.Dialer{Subprotocols: []string{"graphql-transport-ws"}}
	headers := map[string][]string{"Origin": {server.URL}}
	connection, response, err := dialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/graphql", headers)
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("dial GraphQL websocket (status %d): %v", status, err)
	}
	defer connection.Close()

	if err := connection.WriteJSON(map[string]any{"type": "connection_init"}); err != nil {
		t.Fatalf("write connection_init: %v", err)
	}
	var message struct {
		ID      string          `json:"id"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := connection.ReadJSON(&message); err != nil {
		t.Fatalf("read connection_ack: %v", err)
	}
	if message.Type != "connection_ack" {
		t.Fatalf("expected connection_ack, got %#v", message)
	}

	if err := connection.WriteJSON(map[string]any{
		"id":   "task-events",
		"type": "subscribe",
		"payload": map[string]any{
			"query": "subscription { taskEvents { sequence type taskId task { id } dashboardStats { total active } } }",
		},
	}); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for bus.SubscriberCount() != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if bus.SubscriberCount() != 1 {
		t.Fatal("GraphQL subscription did not register with the task event bus")
	}
	bus.Publish(&taskruntime.TaskEvent{
		Type:           taskruntime.TaskEventCreated,
		TaskID:         "task-1",
		Task:           &taskruntime.Task{ID: "task-1", Stage: taskruntime.TaskStageDownloading, StageStatus: taskruntime.TaskStageStatusRunning},
		DashboardStats: taskruntime.TaskStats{Total: 1, Active: 1},
	})

	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if err := connection.ReadJSON(&message); err != nil {
		t.Fatalf("read subscription event: %v", err)
	}
	if message.Type != "next" || message.ID != "task-events" {
		t.Fatalf("expected next task event, got %#v", message)
	}
	var payload struct {
		Data struct {
			TaskEvents struct {
				Sequence int    `json:"sequence"`
				Type     string `json:"type"`
				TaskID   string `json:"taskId"`
			} `json:"taskEvents"`
		} `json:"data"`
	}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		t.Fatalf("decode subscription payload: %v", err)
	}
	if payload.Data.TaskEvents.Sequence != 1 || payload.Data.TaskEvents.Type != "CREATED" || payload.Data.TaskEvents.TaskID != "task-1" {
		t.Fatalf("unexpected subscription payload: %#v", payload)
	}
}
