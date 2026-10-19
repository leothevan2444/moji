package qbittorrent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTorrentUploadIncludesFileBytesWithoutRemoteURL(t *testing.T) {
	client := NewClient(nil)
	client.baseURL = "http://qbt"
	data := "d4:infod4:name7:OFJE657ee"
	client.httpClient = &http.Client{Transport: fileSelectionTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/v2/torrents/add" {
			t.Fatalf("wrong path %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatal(err)
		}
		defer r.MultipartForm.RemoveAll()
		if r.FormValue("urls") != "" {
			t.Fatal("remote download URL must not be sent")
		}
		files := r.MultipartForm.File["torrents"]
		if len(files) != 1 || files[0].Filename != "OFJE-657.torrent" {
			t.Fatalf("missing torrent upload: %+v", files)
		}
		file, err := files[0].Open()
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		actual, err := io.ReadAll(file)
		if err != nil || string(actual) != data {
			t.Fatalf("upload changed: %q %v", actual, err)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("Ok.")), Header: make(http.Header)}, nil
	})}
	if err := client.AddNewTorrent(context.Background(), AddTorrentOptions{Torrents: []TorrentFile{{Filename: "OFJE-657.torrent", Data: []byte(data)}}}); err != nil {
		t.Fatal(err)
	}
}

func TestTorrentSubmissionRejectsFailureBodyEvenWithHTTP200(t *testing.T) {
	for _, body := range []string{"Fails.", "", "<html>login required</html>"} {
		t.Run(body, func(t *testing.T) {
			client := NewClient(nil)
			client.baseURL = "http://qbt"
			client.httpClient = &http.Client{Transport: fileSelectionTransport(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}
			if err := client.AddNewTorrent(context.Background(), AddTorrentOptions{URLs: []string{"magnet:?xt=urn:btih:123"}}); err == nil {
				t.Fatal("failure body was treated as successful submission")
			}
		})
	}
}
