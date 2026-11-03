package graphqlapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

func TestIndexedTaskProjectionKeepsPrivateDownloadLocationsServerSide(t *testing.T) {
	magnet := "magnet:?xt=urn:btih:" + strings.Repeat("a", 40) + "&tr=https://tracker.example/announce?passkey=synthetic-private-passkey"
	task := &taskruntime.Task{ID: "selected", Code: "ABCD-123", TorrentURL: magnet, TorrentFileData: []byte("synthetic-private-bytes"), Candidate: taskruntime.Candidate{Title: "ABCD-123", Link: "https://site.example/grab?ticket=synthetic-private-ticket", MagnetURI: magnet, Source: &indexer.Provenance{Version: 1, ResourceID: "opaque"}}, IndexerResource: &indexer.ResourceRecord{}}
	data, err := json.Marshal(taskToModel(task))
	if err != nil || strings.Contains(string(data), "synthetic-private") || !strings.Contains(string(data), "indexer-resource:opaque") {
		t.Fatal("task projection exposed private acquisition data", err)
	}
	if task.TorrentURL != magnet || task.IndexerResource == nil || string(task.TorrentFileData) != "synthetic-private-bytes" {
		t.Fatal("public projection changed execution state")
	}
	task.Candidate.Source = nil
	if taskToModel(task).TorrentURL != magnet {
		t.Fatal("explicit manual source behavior changed")
	}
}
