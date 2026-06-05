package client

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
)

var client *Client

func TestMain(m *testing.M) {
	if os.Getenv("MOJI_RUN_INTEGRATION") != "1" {
		os.Exit(m.Run())
	}

	host := os.Getenv("MOJI_R18DEV_HOST")
	database := os.Getenv("MOJI_R18DEV_DATABASE")
	user := os.Getenv("MOJI_R18DEV_USER")
	password := os.Getenv("MOJI_R18DEV_PASSWORD")
	if host == "" || database == "" || user == "" || password == "" {
		os.Exit(m.Run())
	}

	port := uint16(5432)
	if rawPort := os.Getenv("MOJI_R18DEV_PORT"); rawPort != "" {
		parsed, err := strconv.ParseUint(rawPort, 10, 16)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid MOJI_R18DEV_PORT: %v\n", err)
			os.Exit(1)
		}
		port = uint16(parsed)
	}

	config := Config{
		Host:     host,
		Port:     port,
		Database: database,
		User:     user,
		Password: password,
		MaxConns: 10,
	}
	var err error
	client, err = NewClient(context.Background(), config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to R18.dev database: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func requireR18DevClient(t *testing.T) *Client {
	t.Helper()
	if client == nil {
		t.Skip("set MOJI_RUN_INTEGRATION=1 and MOJI_R18DEV_* environment variables to run R18.dev integration tests")
	}
	return client
}

func TestGetActressMovies(t *testing.T) {
	client := requireR18DevClient(t)
	actress := os.Getenv("MOJI_R18DEV_TEST_ACTRESS")
	if actress == "" {
		t.Skip("set MOJI_R18DEV_TEST_ACTRESS to run this R18.dev integration test")
	}

	movies, err := client.GetActressMovies(context.Background(), actress)
	if err != nil {
		t.Fatalf("GetActressMovies failed: %v", err)
	}
	t.Logf("found %d movies for %s", len(movies), actress)
	for _, movie := range movies {
		t.Logf("content_id:%s, code:%s, title: %s, other_ids: %v\n",
			movie.ContentID,
			movie.Code,
			movie.Title,
			movie.OtherContentIDs,
		)
	}
}
