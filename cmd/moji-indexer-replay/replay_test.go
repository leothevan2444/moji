package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/cardigann"
)

func TestReplayManualChallengesResumeOriginalForm(t *testing.T) {
	for _, kind := range []string{"text", "image"} {
		t.Run(kind, func(t *testing.T) {
			root := filepath.Join("../../scripts/indexer-diff/cases", "captcha-"+kind)
			definition, err := os.ReadFile(filepath.Join(root, "definition.yml"))
			if err != nil {
				t.Fatal(err)
			}
			metadata, err := os.ReadFile(filepath.Join(root, "scenario.json"))
			if err != nil {
				t.Fatal(err)
			}
			var input scenario
			if err := json.Unmarshal(metadata, &input); err != nil {
				t.Fatal(err)
			}
			var bodies struct {
				Responses []struct {
					BodyText string
					BodyFile string
				}
			}
			if err := json.Unmarshal(metadata, &bodies); err != nil {
				t.Fatal(err)
			}
			input.Definition = string(definition)
			for i, body := range bodies.Responses {
				input.Responses[i].Body = []byte(body.BodyText)
				if body.BodyFile != "" {
					input.Responses[i].Body, err = os.ReadFile(filepath.Join(root, body.BodyFile))
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			report := replay(context.Background(), input)
			if report.Error != "" || report.Challenge == nil || report.Challenge.Type != kind || report.Artifact == nil {
				t.Fatalf("challenge did not resume: %+v", report)
			}
			landing, submissions := 0, 0
			for _, request := range report.Requests {
				if request.URL == "https://fixture.invalid/login" {
					landing++
					if request.Headers["Referer"] != "https://fixture.invalid/" {
						t.Fatalf("login landing lost site referer: %+v", request)
					}
				}
				if request.URL == "https://fixture.invalid/captcha.png" {
					if request.Headers["Referer"] != "https://fixture.invalid/login" || request.Headers["Content-Type"] != "" || request.Headers["Cookie"] != "landing=synthetic" {
						t.Fatalf("captcha request inherited submission headers or lost context: %+v", request)
					}
				}
				if request.Method == "POST" {
					submissions++
					if string(request.Body) != "csrf=synthetic-token&username=synthetic-user&password=synthetic-password&answer=7+%2B+reply" || request.Headers["Cookie"] != "landing=synthetic" {
						t.Fatalf("pending form or cookies lost: %+v", request)
					}
				}
			}
			if landing != 1 || submissions != 1 {
				t.Fatalf("repeated login: %+v", report.Requests)
			}
			input.ChallengeAnswer = nil
			report = replay(context.Background(), input)
			if report.Error != "interactive challenge requires a scripted answer" || report.Challenge == nil {
				t.Fatalf("unanswered challenge did not stop: %+v", report)
			}
			for _, request := range report.Requests {
				if request.Method == "POST" {
					t.Fatal("submitted without an answer")
				}
			}
		})
	}
}

func TestReplayUnexpectedAnswerDoesNotStartSearch(t *testing.T) {
	for _, authenticate := range []bool{false, true} {
		input := publicScenario(t)
		input.Authenticate = authenticate
		answer := "synthetic-answer"
		input.ChallengeAnswer = &answer
		report := replay(context.Background(), input)
		if report.Error == "" || len(report.Requests) != 0 {
			t.Fatalf("unexpected answer did not stop before search: %+v", report)
		}
	}
}

func publicScenario(t *testing.T) scenario {
	t.Helper()
	root := "../../scripts/indexer-diff/cases/public-html"
	definition, err := os.ReadFile(filepath.Join(root, "definition.yml"))
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "search.html"))
	if err != nil {
		t.Fatal(err)
	}
	limit := 100
	return scenario{Definition: string(definition), Query: cardigann.Query{Keywords: "ABCD-123", Categories: []int{2000}, Limit: &limit}, Fetch: true,
		Responses: []responseFixture{
			{Method: "GET", Path: "/search", Status: 200, Headers: http.Header{"Content-Type": {"text/html"}}, Body: body},
			{Method: "GET", Path: "/file.torrent", Status: 200, Body: []byte("d4:infod4:name8:ABCD-1236:lengthi1eee")},
		}}
}

func TestReplayPublicSearchAndAcquisitionMatchReference(t *testing.T) {
	report := replay(context.Background(), publicScenario(t))
	if report.Error != "" || len(report.Results) != 1 || report.Results[0].Title != "ABCD-123" || report.Artifact == nil {
		t.Fatalf("replay did not complete search and fetch: %+v", report)
	}
	if report.Artifact.SHA256 != "05453e95d2e4b252277e10495ec01480f4a0c584476acaa335b8d57a83805216" {
		t.Fatal("torrent bytes differ from reference")
	}
	if len(report.Requests) != 2 || report.Requests[0].URL != "https://fixture.invalid/search?q=ABCD-123&category=film" || report.Requests[1].Headers["Referer"] != "https://fixture.invalid/file.torrent" {
		t.Fatalf("request sequence differs from reference: %+v", report.Requests)
	}
}

func TestReplayUnscriptedRequestFailsWithoutFallbackTransport(t *testing.T) {
	input := publicScenario(t)
	input.Responses = nil
	report := replay(context.Background(), input)
	if !strings.Contains(report.Error, "unscripted request GET /search") || len(report.Requests) != 1 {
		t.Fatalf("unscripted request was not rejected: %+v", report)
	}
}

func TestReplayFileRejectsUnknownProtocolFields(t *testing.T) {
	root := t.TempDir()
	input, output := filepath.Join(root, "case.json"), filepath.Join(root, "report.json")
	if err := os.WriteFile(input, []byte(`{"remoteEndpoint":"https://example.invalid"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := replayFile(input, output); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatal("unknown replay field accepted", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("invalid input wrote a result")
	}
}
