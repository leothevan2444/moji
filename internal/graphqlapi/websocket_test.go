package graphqlapi

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
)

type graphQLWebSocketTestClient struct {
	connection *websocket.Conn
}

func newGraphQLWebSocketTestClient(t *testing.T, resolver *Resolver) *graphQLWebSocketTestClient {
	t.Helper()
	server := httptest.NewServer(NewGraphQLServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver})))
	dialer := websocket.Dialer{Subprotocols: []string{"graphql-transport-ws"}}
	connection, response, err := dialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/graphql", map[string][]string{"Origin": {server.URL}})
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		server.Close()
		t.Fatalf("dial GraphQL websocket (status %d): %v", status, err)
	}
	t.Cleanup(func() {
		_ = connection.Close()
		server.Close()
	})
	if err := connection.WriteJSON(map[string]any{"type": "connection_init"}); err != nil {
		t.Fatalf("write connection_init: %v", err)
	}
	var message graphQLWebSocketMessage
	if err := connection.ReadJSON(&message); err != nil {
		t.Fatalf("read connection_ack: %v", err)
	}
	if message.Type != "connection_ack" {
		t.Fatalf("expected connection_ack, got %#v", message)
	}
	return &graphQLWebSocketTestClient{connection: connection}
}

type graphQLWebSocketMessage struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (c *graphQLWebSocketTestClient) subscribe(t *testing.T, id, query string) {
	t.Helper()
	if err := c.connection.WriteJSON(map[string]any{
		"id": id, "type": "subscribe", "payload": map[string]any{"query": query},
	}); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}
}

func (c *graphQLWebSocketTestClient) readNext(t *testing.T, id string, target any) {
	t.Helper()
	if err := c.connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var message graphQLWebSocketMessage
	if err := c.connection.ReadJSON(&message); err != nil {
		t.Fatalf("read subscription event: %v", err)
	}
	if message.Type != "next" || message.ID != id {
		t.Fatalf("expected next event %q, got %#v", id, message)
	}
	if err := json.Unmarshal(message.Payload, target); err != nil {
		t.Fatalf("decode subscription payload: %v", err)
	}
}

func waitForSubscriber(t *testing.T, count func() int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for count() != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if count() != 1 {
		t.Fatal("GraphQL subscription did not register with the event source")
	}
}
