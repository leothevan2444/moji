package qbittorrent

import (
	"context"
	"testing"
)

func TestGetTorrentList(t *testing.T) {
	torrents, err := client.GetTorrentList(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetTorrentList failed: %v", err)
	}
	t.Logf("Retrieved %d torrents", len(torrents))
	for _, torrent := range torrents {
		t.Logf("Torrent: %s, Hash: %s", torrent.Name, torrent.Hash)
	}
}

func TestGetTorrentGenericProperties(t *testing.T) {
	properties, err := client.GetTorrentGenericProperties(context.Background(), "e056dd321979cee0829fba40ace570aa6c61d9b8")
	if err != nil {
		t.Fatalf("GetTorrentGenericProperties failed: %v", err)
	}
	t.Logf("Generic Properties: %+v", properties)
}

func TestGetTorrentTrackers(t *testing.T) {
	trackers, err := client.GetTorrentTrackers(context.Background(), "e056dd321979cee0829fba40ace570aa6c61d9b8")
	if err != nil {
		t.Fatalf("GetTorrentTrackers failed: %v", err)
	}
	t.Logf("Retrieved %d trackers", len(trackers))
	for _, tracker := range trackers {
		t.Logf("Tracker: %s, Status: %d", tracker.URL, tracker.Status)
	}
}

func TestGetTorrentWebSeeds(t *testing.T) {
	webSeeds, err := client.GetTorrentWebSeeds(context.Background(), "e056dd321979cee0829fba40ace570aa6c61d9b8")
	if err != nil {
		t.Fatalf("GetTorrentWebSeeds failed: %v", err)
	}
	t.Logf("Retrieved %d web seeds", len(webSeeds))
	for _, webSeed := range webSeeds {
		t.Logf("Web Seed: %s", webSeed)
	}
}

func TestGetTorrentContents(t *testing.T) {
	contents, err := client.GetTorrentContents(context.Background(), "e056dd321979cee0829fba40ace570aa6c61d9b8", nil)
	if err != nil {
		t.Fatalf("GetTorrentContents failed: %v", err)
	}
	t.Logf("Retrieved %d contents", len(contents))
	for _, content := range contents {
		t.Logf("Content: %s, Size: %d", content.Name, content.Size)
	}
}

func TestGetTorrentPiecesStates(t *testing.T) {
	piecesStates, err := client.GetTorrentPiecesStates(context.Background(), "e056dd321979cee0829fba40ace570aa6c61d9b8")
	if err != nil {
		t.Fatalf("GetTorrentPiecesStates failed: %v", err)
	}
	t.Logf("Pieces States: %v", piecesStates)
}

func TestGetTorrentPiecesHashes(t *testing.T) {
	piecesHashes, err := client.GetTorrentPiecesHashes(context.Background(), "e056dd321979cee0829fba40ace570aa6c61d9b8")
	if err != nil {
		t.Fatalf("GetTorrentPiecesHashes failed: %v", err)
	}
	t.Logf("Pieces Hashes: %v", piecesHashes)
}

func TestPauseTorrents(t *testing.T) {
	err := client.PauseTorrents(context.Background(), []string{"e056dd321979cee0829fba40ace570aa6c61d9b8"})
	if err != nil {
		t.Fatalf("PauseTorrents failed: %v", err)
	}
	t.Log("Paused torrents successfully")
}

func TestResumeTorrents(t *testing.T) {
	err := client.ResumeTorrents(context.Background(), []string{"e056dd321979cee0829fba40ace570aa6c61d9b8"})
	if err != nil {
		t.Fatalf("ResumeTorrents failed: %v", err)
	}
	t.Log("Resumed torrents successfully")
}

func TestDeleteTorrents(t *testing.T) {
	err := client.DeleteTorrents(context.Background(), []string{"e056dd321979cee0829fba40ace570aa6c61d9b8"}, false)
	if err != nil {
		t.Fatalf("DeleteTorrents failed: %v", err)
	}
	t.Log("Deleted torrents successfully")
}

func TestRecheckTorrents(t *testing.T) {
	err := client.RecheckTorrents(context.Background(), []string{"e056dd321979cee0829fba40ace570aa6c61d9b8"})
	if err != nil {
		t.Fatalf("RecheckTorrents failed: %v", err)
	}
	t.Log("Rechecked torrents successfully")
}

func TestReannounceTorrents(t *testing.T) {
	err := client.ReannounceTorrents(context.Background(), []string{"e056dd321979cee0829fba40ace570aa6c61d9b8"})
	if err != nil {
		t.Fatalf("ReannounceTorrents failed: %v", err)
	}
	t.Log("Reannounced torrents successfully")
}

func TestAddNewTorrent(t *testing.T) {
	err := client.AddNewTorrent(context.Background(), AddTorrentOptions{
		URLs: []string{"magnet:?xt=urn:btih:1582003eebbf6302b1a5ca18c6afd7f287f66643&dn=%2B%2B%2B%20%5BFHD%5D%20FNS-182%20%E3%82%A4%E3%83%B3%E3%82%B8%E3%83%A3%E3%83%B3%E5%8F%A4%E6%B2%B3%E3%83%97%E3%83%AC%E3%82%BC%E3%83%B3%E3%83%84%E3%80%80%E5%A4%A7%E4%BA%BA%E3%81%AE%E3%82%A8%E3%83%B3%E3%82%BF%E3%83%BC%E3%83%86%E3%82%A4%E3%83%A1%E3%83%B3%E3%83%88%20%E5%90%89%E9%AB%98%E5%AF%A7%E3%80%85%E3%81%AE%E6%9C%AC%E5%BD%93%E3%81%AE%E3%82%AA%E3%83%8A%E3%83%8B%E3%83%BC%E3%82%92%E8%A6%8B%E3%81%A6%E3%81%BF%E3%81%9F%E3%81%84%20%E5%90%89%E9%AB%98%E5%AF%A7%E3%80%85&tr=http%3A%2F%2Fsukebei.tracker.wf%3A8888%2Fannounce&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce&tr=udp%3A%2F%2Fexodus.desync.com%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.torrent.eu.org%3A451%2Fannounce"},
	})
	if err != nil {
		t.Fatalf("AddNewTorrent failed: %v", err)
	}
	t.Log("Added new torrent successfully")
}

func TestGetTorrentDownloadLimit(t *testing.T) {
	limit, err := client.GetTorrentDownloadLimit(context.Background(), []string{"e056dd321979cee0829fba40ace570aa6c61d9b8"})
	if err != nil {
		t.Fatalf("GetTorrentDownloadLimit failed: %v", err)
	}
	t.Logf("Download Limit: %v", limit)
}

func TestGetTorrentUploadLimit(t *testing.T) {
	limit, err := client.GetTorrentUploadLimit(context.Background(), []string{"e056dd321979cee0829fba40ace570aa6c61d9b8"})
	if err != nil {
		t.Fatalf("GetTorrentUploadLimit failed: %v", err)
	}
	t.Logf("Upload Limit: %v", limit)
}

func TestGetAllCategories(t *testing.T) {
	categories, err := client.GetAllCategories(context.Background())
	if err != nil {
		t.Fatalf("GetAllCategories failed: %v", err)
	}
	t.Logf("Retrieved %d categories", len(categories))
	for name, category := range categories {
		t.Logf("Category: %s, Properties: %+v", name, category)
	}
}

func TestGetAllTags(t *testing.T) {
	tags, err := client.GetAllTags(context.Background())
	if err != nil {
		t.Fatalf("GetAllTags failed: %v", err)
	}
	t.Logf("Retrieved %d tags", len(tags))
	for _, tag := range tags {
		t.Logf("Tag: %s", tag)
	}
}
