package qbittorrent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fileSelectionTransport func(*http.Request) (*http.Response, error)

func (f fileSelectionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestFileSelectionUsesOfficialParametersAndReportsHTTPFailures(t *testing.T) {
	client := NewClient(nil)
	client.baseURL = "http://qbt"
	status := 200
	client.httpClient = &http.Client{Transport: fileSelectionTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/v2/torrents/filePrio" {
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("id") != "7|12" || r.Form.Get("priority") != "0" {
				t.Fatalf("wrong file selection parameters: %v", r.Form)
			}
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("[]")), Header: make(http.Header)}, nil
	})}
	if err := client.SetTorrentFilePriority(context.Background(), "hash", []int{7, 12}, TorrentFilePriorityDontDownload); err != nil {
		t.Fatal(err)
	}
	status = 403
	if err := client.SetTorrentFilePriority(context.Background(), "hash", []int{7, 12}, TorrentFilePriorityDontDownload); err == nil {
		t.Fatal("HTTP failure was ignored")
	}
	if _, err := client.GetTorrentContents(context.Background(), "hash", nil); err == nil {
		t.Fatal("HTTP failure was ignored")
	}
	if err := client.PauseTorrents(context.Background(), []string{"hash"}); err == nil {
		t.Fatal("stop failure was ignored")
	}
	if err := client.ResumeTorrents(context.Background(), []string{"hash"}); err == nil {
		t.Fatal("start failure was ignored")
	}
}
func TestAddTorrentSupportsMetadataStopAndStoppedState(t *testing.T) {
	client := NewClient(nil)
	client.baseURL = "http://qbt"
	client.httpClient = &http.Client{Transport: fileSelectionTransport(func(r *http.Request) (*http.Response, error) {
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("stopCondition") != "MetadataReceived" || r.FormValue("stopped") != "true" {
			t.Fatalf("metadata acquisition options lost: %v", r.MultipartForm.Value)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("Ok.")), Header: make(http.Header)}, nil
	})}
	stopped := true
	condition := "MetadataReceived"
	if err := client.AddNewTorrent(context.Background(), AddTorrentOptions{URLs: []string{"magnet:?xt=urn:btih:hash"}, Paused: &stopped, StopCondition: &condition}); err != nil {
		t.Fatal(err)
	}
}
