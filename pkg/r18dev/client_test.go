package client

import (
	"context"
	"testing"
)

var client *Client

// jack:secret@pg.example.com:5432/mydb
func TestMain(m *testing.M) {
	config := Config{
		Host:     "homeserver0.local",
		Port:     5432,
		Database: "r18dotdev_20250603",
		User:     "postgres_user",
		Password: "010728",
		MaxConns: 10,
	}
	var err error
	client, err = NewClient(context.Background(), config)
	if err != nil {
		panic(err)
	}
	m.Run()
}

func TestGetActressMovies(t *testing.T) {
	movies, err := client.GetActressMovies(context.Background(), "田野憂")
	if err != nil {
		t.Fatalf("GetActressMovies failed: %v", err)
	}
	t.Logf("found %d movies for 田野憂", len(movies))
	for _, movie := range movies {
		t.Logf("content_id:%s, code:%s, title: %s, other_ids: %v\n",
			movie.ContentID,
			movie.Code,
			movie.Title,
			movie.OtherContentIDs,
		)
	}
}
