package jackett

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var testURL = "http://homeserver0.local:9118"
var testAPIKey = "yivm2eqtrspwajjwhi33cawxzclcagxe"

func TestClient_Search(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Received request: %s %s", r.Method, r.URL)
		if r.URL.Path != "/api/v2.0/indexers/all/results" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if r.URL.Query().Get("apikey") != "test-api-key" {
			t.Errorf("unexpected api key: %s", r.URL.Query().Get("apikey"))
		}

		response := `{
    "Results": [
        {
            "FirstSeen": "0001-01-01T00:00:00",
            "Tracker": "sukebei.nyaa.si",
            "TrackerId": "sukebeinyaasi",
            "TrackerType": "public",
            "CategoryDesc": "XXX",
            "BlackholeLink": null,
            "Title": "[HD/720p] NAAC-019B Best naked/彩月七緒",
            "Guid": "https://sukebei.nyaa.si/download/4279375.torrent",
            "Link": "http://homeserver0.local:9118/dl/sukebeinyaasi/?jackett_apikey=yivm2eqtrspwajjwhi33cawxzclcagxe&path=Q2ZESjhDcWNCYzNMUmx4RG9sdVRDTjcwZlh6MTBzeXB0alNOdi12Z2kxVDFBNXBMUl9Ib3JoN2FyeTNqb3psc3FCNDBiS0ZOTUVzZTZndnBxNVA0ZjRQbER6Zl9aUGpTbUUwRGZOem1wYjUwOHJiYmczMDFkTWpjcVRDNGZkN3dfcGFsVWc1Z2dRVVNHZDJCT0Rua0VFRXVWTHJiWW9rc1V2QzFlck1Pb0ItbDFScVlSdFlFaVhublRFM1BqWmlIYWtXSmp3&file=%5BHD_720p%5D+NAAC-019B+Best+naked_%E5%BD%A9%E6%9C%88%E4%B8%83%E7%B7%92",
            "Details": "https://sukebei.nyaa.si/view/4279375",
            "PublishDate": "2025-03-27T16:57:00+08:00",
            "Category": [
                6000,
                155285
            ],
            "Size": 922117760,
            "Files": null,
            "Grabs": 534,
            "Description": null,
            "RageID": null,
            "TVDBId": null,
            "Imdb": null,
            "TMDb": null,
            "TVMazeId": null,
            "TraktId": null,
            "DoubanId": null,
            "Genres": null,
            "Languages": [

            ],
            "Subs": [

            ],
            "Year": null,
            "Author": null,
            "BookTitle": null,
            "Publisher": null,
            "Artist": null,
            "Album": null,
            "Label": null,
            "Track": null,
            "Seeders": 1,
            "Peers": 1,
            "Poster": null,
            "InfoHash": "82c0d4480e151d31d7cc4421a0b5d678d588b478",
            "MagnetUri": "magnet:?xt=urn:btih:82c0d4480e151d31d7cc4421a0b5d678d588b478&dn=%5BHD%2F720p%5D%20NAAC-019B%20Best%20naked%2F%E5%BD%A9%E6%9C%88%E4%B8%83%E7%B7%92&tr=http%3A%2F%2Fsukebei.tracker.wf%3A8888%2Fannounce&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce&tr=udp%3A%2F%2Fexodus.desync.com%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.torrent.eu.org%3A451%2Fannounce",
            "MinimumRatio": null,
            "MinimumSeedTime": null,
            "DownloadVolumeFactor": 0,
            "UploadVolumeFactor": 1,
            "Gain": 0.8587890863418579
        },
        {
            "FirstSeen": "0001-01-01T00:00:00",
            "Tracker": "sukebei.nyaa.si",
            "TrackerId": "sukebeinyaasi",
            "TrackerType": "public",
            "CategoryDesc": "XXX",
            "BlackholeLink": null,
            "Title": "+++ [FHD] NAAC-019 Best naked/彩月七緒",
            "Guid": "https://sukebei.nyaa.si/download/4279072.torrent",
            "Link": "http://homeserver0.local:9118/dl/sukebeinyaasi/?jackett_apikey=yivm2eqtrspwajjwhi33cawxzclcagxe&path=Q2ZESjhDcWNCYzNMUmx4RG9sdVRDTjcwZlh4WDBuY3Znem9nbHVXNEFHRUtNQ1dWMU1MSFRBUDNTbVgzc1JucWgzc01rS2J3S01lTEpqSGZDSzgzRVRael9nczF5N2VYUWhKYzd6TjlrRVp6R1BwNFBzMFdiWXJvVElxa1F0dHE0YlhLUjdtbF9zV3hXN0hfU0tIbFFpRGFRUHBLQmphdDlQRzBnd2ZHZDRnTUR0SDNwdV94V3ViclNFak9lYW1tNEtoN21R&file=%2B%2B%2B+%5BFHD%5D+NAAC-019+Best+naked_%E5%BD%A9%E6%9C%88%E4%B8%83%E7%B7%92",
            "Details": "https://sukebei.nyaa.si/view/4279072",
            "PublishDate": "2025-03-27T01:50:00+08:00",
            "Category": [
                6000,
                155285
            ],
            "Size": 4617090048,
            "Files": null,
            "Grabs": 3110,
            "Description": null,
            "RageID": null,
            "TVDBId": null,
            "Imdb": null,
            "TMDb": null,
            "TVMazeId": null,
            "TraktId": null,
            "DoubanId": null,
            "Genres": null,
            "Languages": [

            ],
            "Subs": [

            ],
            "Year": null,
            "Author": null,
            "BookTitle": null,
            "Publisher": null,
            "Artist": null,
            "Album": null,
            "Label": null,
            "Track": null,
            "Seeders": 9,
            "Peers": 2,
            "Poster": null,
            "InfoHash": "0f057700feac449de6ab93fe4ed2be16d22ff9e9",
            "MagnetUri": "magnet:?xt=urn:btih:0f057700feac449de6ab93fe4ed2be16d22ff9e9&dn=%2B%2B%2B%20%5BFHD%5D%20NAAC-019%20Best%20naked%2F%E5%BD%A9%E6%9C%88%E4%B8%83%E7%B7%92&tr=http%3A%2F%2Fsukebei.tracker.wf%3A8888%2Fannounce&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce&tr=udp%3A%2F%2Fexodus.desync.com%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.torrent.eu.org%3A451%2Fannounce",
            "MinimumRatio": null,
            "MinimumSeedTime": null,
            "DownloadVolumeFactor": 0,
            "UploadVolumeFactor": 1,
            "Gain": 38.70000171661377
        }
    ],
    "Indexers": [
        {
            "ID": "sukebeinyaasi",
            "Name": "sukebei.nyaa.si",
            "Status": 2,
            "Results": 2,
            "Error": null,
            "ElapsedTime": 0
        }
    ]
}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "test-api-key")
	results, err := client.Search(SearchRequest{
		Query:   "test query",
		Tracker: []string{"test-tracker"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result, got none")
	}
	for _, result := range results {
		t.Logf("Tracker: %s, Title: %s, PublishDate: %s, Details: %s, Category: %v, Size: %d, InfoHash: %s, MagnetURI: %s",
			result.Tracker, result.Title, result.PublishDate, result.Details,
			result.Category, result.Size, result.InfoHash, result.MagnetURI)
	}
}

func TestClient_SearchReal(t *testing.T) {
	// This test requires a real Jackett instance to be running.
	// Replace with your actual Jackett URL and API key.
	client := NewClient(testURL, testAPIKey)
	results, err := client.Search(SearchRequest{
		Query:    "SONE-786",
		Trackers: []string{"sukebeinyaasi", "onejav", "u3c3"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result, got none")
	}
	for _, result := range results {
		t.Logf("Tracker: %s, Title: %s, PublishDate: %s, Details: %s, Category: %v, Size: %d, InfoHash: %s, MagnetURI: %s",
			result.Tracker, result.Title, result.PublishDate, result.Details,
			result.Category, result.Size, result.InfoHash, result.MagnetURI)
	}
	t.Logf("Search completed successfully with %d results", len(results))
}

func TestClient_GetIndexersReal(t *testing.T) {
	// This test requires a real Jackett instance to be running.
	// Replace with your actual Jackett URL and API key.
	client := NewClient(testURL, testAPIKey)
	client.SetPassword("010728")
	indexers, err := client.GetIndexers()
	if err != nil {
		t.Fatal(err)
	}
	if len(indexers) == 0 {
		t.Fatal("expected at least one indexer, got none")
	}
	for _, indexer := range indexers {
		t.Logf("Indexer ID: %s, Name: %s, Description: %s, Type: %s, Configured: %t, SiteLink: %s, Language: %s",
			indexer.ID, indexer.Name, indexer.Description, indexer.Type,
			indexer.Configured, indexer.SiteLink, indexer.Language)
	}
	t.Logf("GetIndexers completed successfully with %d indexers", len(indexers))
}
