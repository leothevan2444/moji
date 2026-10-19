package taskruntime

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/pkg/jackett"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

func torrentResponse(body string, status int) Option {
	return WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: req}, nil
	})})
}

func TestSourcingValidationBlocksBeforeDownloaderSubmission(t *testing.T) {
	for _, tc := range []struct {
		name, title, torrentName string
		files                    []string
		body                     string
		status                   int
		errorCode                string
	}{
		{name: "fuzzy title", title: "OFJE667", torrentName: "OFJE657", files: []string{"OFJE657.mp4"}, errorCode: TaskStageErrorCandidateMismatch},
		{name: "wrong torrent name", title: "OFJE657", torrentName: "OFJE667", files: []string{"movie.mp4"}, errorCode: TaskStageErrorCandidateMismatch},
		{name: "wrong video despite matching name", title: "OFJE657", torrentName: "OFJE657", files: []string{"OFJE667.mp4"}, errorCode: TaskStageErrorCandidateMismatch},
		{name: "conflicting videos", title: "OFJE657", torrentName: "OFJE657", files: []string{"OFJE657.mp4", "OFJE667.mp4"}, errorCode: TaskStageErrorCandidateMismatch},
		{name: "ambiguous title", title: "OFJE657 OFJE667", torrentName: "OFJE657", files: []string{"OFJE657.mp4"}, errorCode: TaskStageErrorCandidateMismatch},
		{name: "unknown identity", title: "release", torrentName: "release", files: []string{"movie.mp4"}, errorCode: TaskStageErrorCandidateUnverified},
		{name: "no video", title: "OFJE657", torrentName: "OFJE657", files: []string{"readme.txt"}, errorCode: TaskStageErrorCandidateUnverified},
		{name: "HTTP failure", title: "OFJE657", status: http.StatusBadGateway, errorCode: TaskStageErrorTorrentMetadata},
		{name: "HTML response", title: "OFJE657", body: "<html>upstream error</html>", errorCode: TaskStageErrorTorrentMetadata},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := tc.body
			if body == "" {
				body = testTorrentFile(tc.torrentName, tc.files)
			}
			status := tc.status
			if status == 0 {
				status = http.StatusOK
			}
			qbt := &fakeTorrentAdder{}
			store := NewMemoryTaskStore()
			s, err := NewService(fakeTracker{results: []jackett.SearchResult{{Title: tc.title, Link: "http://indexer.invalid/file?apikey=secret"}}}, qbt, store, torrentResponse(body, status))
			if err != nil {
				t.Fatal(err)
			}
			task, err := s.DownloadMediaContext(context.Background(), DownloadRequest{Code: "OFJE-657"})
			if err == nil || task == nil || task.Stage != TaskStageSourcing || task.StageStatus != TaskStageStatusBlocked || task.StageErrorCode != tc.errorCode {
				t.Fatalf("expected blocked sourcing with %s, got task=%+v err=%v", tc.errorCode, task, err)
			}
			if len(qbt.options.URLs) != 0 || len(qbt.options.Torrents) != 0 {
				t.Fatal("unverified torrent reached qBittorrent")
			}
			stored, err := store.Find(context.Background(), task.ID)
			if err != nil || stored.StageErrorCode != tc.errorCode || stored.SourcingCompletedAt != nil || strings.Contains(stored.StageErrorMessage, "secret") {
				t.Fatalf("incorrect persisted validation state: %+v %v", stored, err)
			}
		})
	}
}

func TestSourcingUploadsExactVerifiedTorrentBytesAndIdentity(t *testing.T) {
	for _, alias := range []string{"OFJE657", "ofje_657", "OFJE 657", "OFJE00657", "OFJE-657", "[HD 720p] OFJE657", "OFJE657-part01"} {
		t.Run(alias, func(t *testing.T) {
			body := testTorrentFile(alias, []string{"movie/" + alias + ".mp4", "readme.txt"})
			qbt := &fakeTorrentAdder{}
			store := NewMemoryTaskStore()
			s, err := NewService(fakeTracker{results: []jackett.SearchResult{{Title: alias, Link: "http://indexer.invalid/file"}}}, qbt, store, torrentResponse(body, http.StatusOK),
				WithMediaPreparation(func() stashsync.IntegrationConfig { return stashsync.IntegrationConfig{} }, nil))
			if err != nil {
				t.Fatal(err)
			}
			task, err := s.DownloadMediaContext(context.Background(), DownloadRequest{Code: "OFJE-657", SavePath: "/downloads", Category: "moji", Tags: "test"})
			if err != nil {
				t.Fatal(err)
			}
			if len(qbt.options.URLs) != 0 || len(qbt.options.Torrents) != 1 || !bytes.Equal(qbt.options.Torrents[0].Data, []byte(body)) {
				t.Fatal("expected verified bytes uploaded without a remote URL")
			}
			metadata, _ := parseTorrentMetadata([]byte(body))
			stored, _ := store.Find(context.Background(), task.ID)
			if stored.Stage != TaskStageDownloading || stored.TorrentIdentityHash != metadata.InfoHash || stored.Candidate.InfoHash != metadata.InfoHash || stored.SourcingCompletedAt == nil {
				t.Fatalf("verified identity was not persisted: %+v", stored)
			}
			if *qbt.options.SavePath != "/downloads" || *qbt.options.Category != "moji" || *qbt.options.Tags != "test" {
				t.Fatal("download defaults lost")
			}
			if qbt.options.Paused == nil || !*qbt.options.Paused || qbt.options.StopCondition == nil || *qbt.options.StopCondition != "MetadataReceived" {
				t.Fatal("file inspection stop condition lost")
			}
		})
	}
}

func TestSourcingFallsBackAfterMetadataMismatch(t *testing.T) {
	qbt := &fakeTorrentAdder{}
	s, err := NewService(fakeTracker{results: []jackett.SearchResult{
		{Title: "OFJE657 preferred", Link: "http://indexer.invalid/wrong", Seeders: 10},
		{Title: "OFJE657 fallback", Link: "http://indexer.invalid/right", Seeders: 1},
	}}, qbt, NewMemoryTaskStore(), WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		code := "OFJE657"
		if req.URL.Path == "/wrong" {
			code = "OFJE667"
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(testTorrentFile(code, []string{code + ".mp4"}))), Request: req}, nil
	})}))
	if err != nil {
		t.Fatal(err)
	}
	task, err := s.DownloadMediaContext(context.Background(), DownloadRequest{Code: "OFJE-657"})
	if err != nil || task.Candidate.Title != "OFJE657 fallback" || len(qbt.options.Torrents) != 1 {
		t.Fatalf("failed fallback: %+v %v", task, err)
	}
}

func TestMagnetCodeValidation(t *testing.T) {
	for _, tc := range []struct {
		title, dn string
		wantError bool
	}{
		{"OFJE657", "ofje_00657", false},
		{"OFJE657", "", false},
		{"release", "OFJE657", false},
		{"OFJE667", "OFJE657", true},
		{"OFJE657", "OFJE667", true},
		{"release", "", true},
	} {
		t.Run(tc.title+"/"+tc.dn, func(t *testing.T) {
			qbt := &fakeTorrentAdder{}
			s, _ := NewService(fakeTracker{results: []jackett.SearchResult{{Title: tc.title, MagnetURI: "magnet:?xt=urn:btih:123&dn=" + tc.dn}}}, qbt, NewMemoryTaskStore())
			task, err := s.DownloadMediaContext(context.Background(), DownloadRequest{Code: "OFJE-657"})
			if (err != nil) != tc.wantError {
				t.Fatalf("unexpected result: %+v %v", task, err)
			}
			if tc.wantError && len(qbt.options.URLs) != 0 {
				t.Fatal("mismatched magnet submitted")
			}
			if !tc.wantError && len(qbt.options.URLs) != 1 {
				t.Fatal("matching magnet not submitted")
			}
		})
	}
}

func TestResolveAndRetryCannotBypassSourcingValidation(t *testing.T) {
	for _, mode := range []string{"resolve", "retry"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			store := NewMemoryTaskStore()
			task := &Task{ID: mode, Code: "OFJE-657", Stage: TaskStageSourcing, StageStatus: TaskStageStatusBlocked, TorrentURL: "http://indexer.invalid/wrong", Candidate: Candidate{Title: "OFJE657"}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
			if mode == "retry" {
				task.Stage = TaskStageDownloading
				task.StageErrorCode = TaskStageErrorTorrentSubmit
			}
			if err := store.Create(ctx, task); err != nil {
				t.Fatal(err)
			}
			qbt := &fakeTorrentAdder{}
			s, _ := NewService(fakeTracker{}, qbt, store, torrentResponse(testTorrentFile("OFJE667", []string{"OFJE667.mp4"}), 200))
			var err error
			if mode == "resolve" {
				task, err = s.ResolveBlockedSourcingTask(ctx, task.ID, ResolveBlockedSourcingRequest{URL: task.TorrentURL, Title: "OFJE657"})
			} else {
				task, err = s.RetryTask(ctx, task.ID, nil)
			}
			if err == nil || task.Stage != TaskStageSourcing || task.StageStatus != TaskStageStatusBlocked || task.StageErrorCode != TaskStageErrorCandidateMismatch || len(qbt.options.Torrents) != 0 {
				t.Fatalf("validation bypassed: %+v %v", task, err)
			}
		})
	}
}

func TestTorrentHashMismatchAndMalformedMetadataAreRejected(t *testing.T) {
	s, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, NewMemoryTaskStore(), torrentResponse(testTorrentFile("OFJE657", []string{"OFJE657.mp4"}), 200))
	_, err := s.prepareTorrent(context.Background(), "OFJE-657", Candidate{Title: "OFJE657", InfoHash: "different"}, "http://indexer.invalid/file")
	var validation *candidateValidationError
	if !errors.As(err, &validation) || validation.code != TaskStageErrorTorrentMetadata {
		t.Fatalf("hash mismatch accepted: %v", err)
	}
	for _, body := range []string{"de", "d4:infod4:name99999999999999999999999999999999:xe", testTorrentFile("OFJE657", []string{"OFJE657.mp4"}) + "junk"} {
		if _, err := parseTorrentMetadata([]byte(body)); err == nil {
			t.Fatalf("accepted malformed torrent %q", body)
		}
	}
	deep := "d5:extra" + strings.Repeat("l", 100) + strings.Repeat("e", 100) + "e"
	if _, err := parseTorrentMetadata([]byte(deep)); err == nil {
		t.Fatal("unbounded nesting accepted")
	}
	s.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(strings.Repeat("x", (8<<20)+1)))}, nil
	})}
	if _, err := s.prepareTorrent(context.Background(), "OFJE-657", Candidate{Title: "OFJE657"}, "http://indexer.invalid/oversize"); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversize torrent accepted: %v", err)
	}
}

func TestTorrentUTF8NamesCannotHideConflictingCodes(t *testing.T) {
	for _, key := range []string{"name.utf-8", "path.utf-8"} {
		t.Run(key, func(t *testing.T) {
			body := testTorrentFile("OFJE657", []string{"OFJE657.mp4"})
			if key == "name.utf-8" {
				needle := bencodeString("name") + bencodeString("OFJE657")
				body = strings.Replace(body, needle, needle+bencodeString(key)+bencodeString("OFJE667"), 1)
			} else {
				needle := bencodeString("path") + "l" + bencodeString("OFJE657.mp4") + "e"
				body = strings.Replace(body, needle, needle+bencodeString(key)+"l"+bencodeString("OFJE667.mp4")+"e", 1)
			}
			metadata, err := parseTorrentMetadata([]byte(body))
			if err != nil {
				t.Fatal(err)
			}
			if err := validateTorrentCode("OFJE-657", metadata); err == nil {
				t.Fatalf("conflicting %s was ignored", key)
			}
		})
	}
}

func TestTitleSimilarityRecognizesCodeAliases(t *testing.T) {
	selector := defaultCandidateSelector{}
	result, err := selector.Select(context.Background(), "OFJE-657", []jackett.SearchResult{
		{Title: "OFJE-667", MagnetURI: "magnet:?xt=urn:btih:wrong"},
		{Title: "ofje00657", MagnetURI: "magnet:?xt=urn:btih:right"},
	}, candidateSelectionConfig(true, 0, []config.CandidateSelectionRule{{Type: config.CandidateSelectionRuleTypeTitleSimilarity, Enabled: true}}))
	if err != nil || result.Title != "ofje00657" {
		t.Fatalf("alias did not win: %+v %v", result, err)
	}
}

func TestSyncRecoversUnstartedFuzzySubmissionsAndKeepsSourcingBlocked(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTaskStore()
	for _, pair := range [][2]string{{"OFJE-657", "OFJE667"}, {"IPOK-004", "IPOK006"}} {
		task := &Task{ID: pair[0], Code: pair[0], Candidate: Candidate{Title: pair[1]}, TorrentURL: "http://indexer.invalid/file", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		completed := time.Now()
		task.SourcingCompletedAt = &completed
		if err := store.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	qbt := &fakeTorrentAdder{torrents: []qbittorrent.Torrent{{Name: "OFJE667", Hash: "wrong", Progress: 0.5}}}
	s, _ := NewService(fakeTracker{}, qbt, store)
	for i := 0; i < 2; i++ {
		if _, err := s.SyncProgress(ctx); err != nil {
			t.Fatal(err)
		}
		for _, id := range []string{"OFJE-657", "IPOK-004"} {
			task, err := store.Find(ctx, id)
			if err != nil || task.Stage != TaskStageSourcing || task.StageStatus != TaskStageStatusBlocked || task.StageErrorCode != TaskStageErrorCandidateMismatch || task.SourcingCompletedAt != nil || task.TorrentHash != "" {
				t.Fatalf("invalid historical submission was not blocked: %+v %v", task, err)
			}
		}
	}
}
