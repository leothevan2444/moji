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
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/stashsync"
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
		{name: "fuzzy title", title: "OFJE-667", torrentName: "OFJE-657", files: []string{"OFJE-657.mp4"}, errorCode: TaskStageErrorCandidateUnverified},
		{name: "wrong torrent name", title: "OFJE-657", torrentName: "OFJE-667", files: []string{"movie.mp4"}, errorCode: TaskStageErrorCandidateUnverified},
		{name: "longer number is not the target", title: "OFJE-6570", torrentName: "OFJE-657", files: []string{"OFJE-657.mp4"}, errorCode: TaskStageErrorCandidateUnverified},
		{name: "unknown identity", title: "release", torrentName: "release", files: []string{"movie.mp4"}, errorCode: TaskStageErrorCandidateUnverified},
		{name: "no video", title: "OFJE-657", torrentName: "OFJE-657", files: []string{"readme.txt"}, errorCode: TaskStageErrorCandidateUnverified},
		{name: "HTTP failure", title: "OFJE-657", status: http.StatusBadGateway, errorCode: TaskStageErrorTorrentMetadata},
		{name: "HTML response", title: "OFJE-657", body: "<html>upstream error</html>", errorCode: TaskStageErrorTorrentMetadata},
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
			s, err := NewService(fakeTracker{results: []indexer.SelectionCandidate{{Title: tc.title, Link: "http://indexer.invalid/file?apikey=secret"}}}, qbt, store, torrentResponse(body, status))
			if err != nil {
				t.Fatal(err)
			}
			task, err := s.runSourcingFixture(context.Background(), DownloadRequest{Film: testFilm("OFJE-657"), Code: "OFJE-657"})
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
	for _, alias := range []string{"OFJE-657", "ofje-657", "[HD 720p] OFJE-657", "OFJE-657-part01"} {
		t.Run(alias, func(t *testing.T) {
			body := testTorrentFile(alias, []string{"movie/" + alias + ".mp4", "readme.txt"})
			qbt := &fakeTorrentAdder{}
			store := NewMemoryTaskStore()
			s, err := NewService(fakeTracker{results: []indexer.SelectionCandidate{{Title: alias, Link: "http://indexer.invalid/file"}}}, qbt, store, torrentResponse(body, http.StatusOK),
				WithMediaPreparation(func() stashsync.IntegrationConfig { return stashsync.IntegrationConfig{} }, nil))
			if err != nil {
				t.Fatal(err)
			}
			task, err := s.runSourcingFixture(context.Background(), DownloadRequest{Film: testFilm("OFJE-657"), Code: "OFJE-657", SavePath: "/downloads", Category: "moji", Tags: "test"})
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
	s, err := NewService(fakeTracker{results: []indexer.SelectionCandidate{
		{Title: "OFJE-657 preferred", Link: "http://indexer.invalid/wrong", Seeders: 10},
		{Title: "OFJE-657 fallback", Link: "http://indexer.invalid/right", Seeders: 1},
	}}, qbt, NewMemoryTaskStore(), WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		code := "OFJE-657"
		if req.URL.Path == "/wrong" {
			code = "OFJE-667"
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(testTorrentFile(code, []string{code + ".mp4"}))), Request: req}, nil
	})}))
	if err != nil {
		t.Fatal(err)
	}
	task, err := s.runSourcingFixture(context.Background(), DownloadRequest{Film: testFilm("OFJE-657"), Code: "OFJE-657"})
	if err != nil || task.Candidate.Title != "OFJE-657 fallback" || len(qbt.options.Torrents) != 1 {
		t.Fatalf("failed fallback: %+v %v", task, err)
	}
}

func TestSourcingAcceptsTargetTorrentWithAdvertisementVideos(t *testing.T) {
	body := testTorrentFile("PPPE-454", []string{"4k688.com@PPPE-454.mp4", "18+游戏大全(996gg.cc).mp4", "javcall.com.mp4"})
	qbt := &fakeTorrentAdder{}
	s, err := NewService(fakeTracker{results: []indexer.SelectionCandidate{{Title: "+++ [FHD] PPPE-454", Link: "http://indexer.invalid/file"}}}, qbt, NewMemoryTaskStore(), torrentResponse(body, http.StatusOK))
	if err != nil {
		t.Fatal(err)
	}
	task, err := s.runSourcingFixture(context.Background(), DownloadRequest{Film: testFilm("PPPE-454"), Code: "PPPE-454"})
	if err != nil || task.Stage != TaskStageDownloading || len(qbt.options.Torrents) != 1 || !bytes.Equal(qbt.options.Torrents[0].Data, []byte(body)) {
		t.Fatalf("target torrent with advertisement videos rejected: task=%+v err=%v", task, err)
	}
}

func TestTorrentCodeRequiresTargetEvidenceButAllowsUnidentifiedVideos(t *testing.T) {
	for _, tc := range []struct {
		name, torrentName string
		files             []string
		wantError         bool
	}{
		{name: "target in torrent name", torrentName: "PPPE-454", files: []string{"movie.mp4", "javcall.com.mp4"}},
		{name: "target in video name", torrentName: "release", files: []string{"PPPE-454.mp4", "movie.mp4"}},
		{name: "quality and part labels", torrentName: "PPPE-454", files: []string{"HD720.mp4", "PART01.mp4"}},
		{name: "compact number is not target evidence", torrentName: "release", files: []string{"PPPE454.mp4"}, wantError: true},
		{name: "longer number is not target evidence", torrentName: "release", files: []string{"PPPE-4540.mp4"}, wantError: true},
		{name: "different number is not target evidence", torrentName: "release", files: []string{"PPPE-455.mp4"}, wantError: true},
		{name: "no target evidence", torrentName: "release", files: []string{"movie.mp4", "javcall.com.mp4"}, wantError: true},
		{name: "different video code", torrentName: "PPPE-454", files: []string{"PPPE-454.mp4", "PPPE-455.mp4"}},
		{name: "different compact code", torrentName: "PPPE-454", files: []string{"PPPE455.mp4"}},
		{name: "longer number", torrentName: "PPPE-454", files: []string{"PPPE-4540.mp4"}},
		{name: "different torrent code", torrentName: "PPPE-455", files: []string{"PPPE-454.mp4"}},
		{name: "no video", torrentName: "PPPE-454", files: []string{"readme.txt"}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metadata, err := parseTorrentMetadata([]byte(testTorrentFile(tc.torrentName, tc.files)))
			if err != nil {
				t.Fatal(err)
			}
			if err := validateTorrentCode("PPPE-454", metadata); (err != nil) != tc.wantError {
				t.Fatalf("validation error=%v, wantError=%v", err, tc.wantError)
			}
		})
	}
}

func TestMagnetCodeValidation(t *testing.T) {
	for _, tc := range []struct {
		title, dn string
		wantError bool
	}{
		{"OFJE-657", "ofje-657", false},
		{"OFJE-657", "", false},
		{"release", "OFJE-657", false},
		{"OFJE-667", "OFJE-657", false},
		{"OFJE-657", "OFJE-667", false},
		{"release", "", true},
	} {
		t.Run(tc.title+"/"+tc.dn, func(t *testing.T) {
			qbt := &fakeTorrentAdder{}
			s, _ := NewService(fakeTracker{results: []indexer.SelectionCandidate{{Title: tc.title, MagnetURI: "magnet:?xt=urn:btih:123&dn=" + tc.dn}}}, qbt, NewMemoryTaskStore())
			task, err := s.runSourcingFixture(context.Background(), DownloadRequest{Film: testFilm("OFJE-657"), Code: "OFJE-657"})
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
	for _, mode := range []string{"retry"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			store := NewMemoryTaskStore()
			task := &Task{ID: mode, Code: "OFJE-657", Stage: TaskStageSourcing, StageStatus: TaskStageStatusBlocked, TorrentURL: "http://indexer.invalid/wrong", Candidate: Candidate{Title: "OFJE-657"}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
			if mode == "retry" {
				task.Stage = TaskStageDownloading
				task.StageErrorCode = TaskStageErrorTorrentSubmit
			}
			if err := store.Create(ctx, task); err != nil {
				t.Fatal(err)
			}
			qbt := &fakeTorrentAdder{}
			s, _ := NewService(fakeTracker{}, qbt, store, torrentResponse(testTorrentFile("OFJE-667", []string{"OFJE-667.mp4"}), 200))
			var err error
			if mode == "resolve" {
				task, err = s.ResolveBlockedSourcingTask(ctx, task.ID, ResolveBlockedSourcingRequest{URL: task.TorrentURL, Title: "OFJE-657"})
			} else {
				task, err = s.RetryTask(ctx, task.ID, nil)
			}
			if err == nil || task.Stage != TaskStageSourcing || task.StageStatus != TaskStageStatusBlocked || task.StageErrorCode != TaskStageErrorCandidateUnverified || len(qbt.options.Torrents) != 0 {
				t.Fatalf("validation bypassed: %+v %v", task, err)
			}
		})
	}
}

func TestTorrentHashMismatchAndMalformedMetadataAreRejected(t *testing.T) {
	s, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, NewMemoryTaskStore(), torrentResponse(testTorrentFile("OFJE-657", []string{"OFJE-657.mp4"}), 200))
	_, err := s.prepareTorrent(context.Background(), "OFJE-657", Candidate{Title: "OFJE-657", InfoHash: "different"}, "http://indexer.invalid/file")
	var validation *candidateValidationError
	if !errors.As(err, &validation) || validation.code != TaskStageErrorTorrentMetadata {
		t.Fatalf("hash mismatch accepted: %v", err)
	}
	for _, body := range []string{"de", "d4:infod4:name99999999999999999999999999999999:xe", testTorrentFile("OFJE-657", []string{"OFJE-657.mp4"}) + "junk"} {
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
	if _, err := s.prepareTorrent(context.Background(), "OFJE-657", Candidate{Title: "OFJE-657"}, "http://indexer.invalid/oversize"); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversize torrent accepted: %v", err)
	}
}

func TestTorrentUTF8NamesRetainTargetEvidence(t *testing.T) {
	for _, key := range []string{"name.utf-8", "path.utf-8"} {
		t.Run(key, func(t *testing.T) {
			body := testTorrentFile("OFJE-657", []string{"OFJE-657.mp4"})
			if key == "name.utf-8" {
				needle := bencodeString("name") + bencodeString("OFJE-657")
				body = strings.Replace(body, needle, needle+bencodeString(key)+bencodeString("OFJE-667"), 1)
			} else {
				needle := bencodeString("path") + "l" + bencodeString("OFJE-657.mp4") + "e"
				body = strings.Replace(body, needle, needle+bencodeString(key)+"l"+bencodeString("OFJE-667.mp4")+"e", 1)
			}
			metadata, err := parseTorrentMetadata([]byte(body))
			if err != nil {
				t.Fatal(err)
			}
			if err := validateTorrentCode("OFJE-657", metadata); err != nil {
				t.Fatalf("target evidence in original encoding lost for %s: %v", key, err)
			}
			body = strings.ReplaceAll(body, "OFJE-657", "OFJE-667")
			metadata, err = parseTorrentMetadata([]byte(body))
			if err != nil {
				t.Fatal(err)
			}
			if err := validateTorrentCode("OFJE-657", metadata); err == nil {
				t.Fatalf("metainfo without target evidence accepted for %s", key)
			}
		})
	}
}

func TestTitleSimilarityRecognizesTheCompleteTarget(t *testing.T) {
	selector := defaultCandidateSelector{}
	result, err := selector.Select(context.Background(), "OFJE-657", []indexer.SelectionCandidate{
		{Title: "OFJE-667", MagnetURI: "magnet:?xt=urn:btih:wrong"},
		{Title: "ofje-657", MagnetURI: "magnet:?xt=urn:btih:right"},
	}, candidateSelectionConfig(true, 0, []config.CandidateSelectionRule{{Type: config.CandidateSelectionRuleTypeTitleSimilarity, Enabled: true}}))
	if err != nil || result.Title != "ofje-657" {
		t.Fatalf("alias did not win: %+v %v", result, err)
	}
}

func TestSyncRecoversUnstartedFuzzySubmissionsAndKeepsSourcingBlocked(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTaskStore()
	for _, pair := range [][2]string{{"OFJE-657", "OFJE-667"}, {"IPOK-004", "IPOK006"}} {
		task := &Task{ID: pair[0], Code: pair[0], Candidate: Candidate{Title: pair[1]}, TorrentURL: "http://indexer.invalid/file", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		completed := time.Now()
		task.SourcingCompletedAt = &completed
		if err := store.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	qbt := &fakeTorrentAdder{torrents: []qbittorrent.Torrent{{Name: "OFJE-667", Hash: "wrong", Progress: 0.5}}}
	s, _ := NewService(fakeTracker{}, qbt, store)
	for i := 0; i < 2; i++ {
		if _, err := s.SyncProgress(ctx); err != nil {
			t.Fatal(err)
		}
		for _, id := range []string{"OFJE-657", "IPOK-004"} {
			task, err := store.Find(ctx, id)
			if err != nil || task.Stage != TaskStageSourcing || task.StageStatus != TaskStageStatusBlocked || task.StageErrorCode != TaskStageErrorCandidateUnverified || task.SourcingCompletedAt != nil || task.TorrentHash != "" {
				t.Fatalf("invalid historical submission was not blocked: %+v %v", task, err)
			}
		}
	}
}
