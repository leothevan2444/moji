package indexerdefs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sourceAPI = "https://api.github.com/repos/Prowlarr/Indexers/commits?path=definitions/v11&per_page=1"
const sourceArchive = "https://codeload.github.com/Prowlarr/Indexers/tar.gz/"

// SourceError deliberately omits response bodies and request URLs. Callers may
// use Status and RetryAfter to schedule bounded update checks.
type SourceError struct {
	Status     int
	RetryAfter time.Duration
}

func (e *SourceError) Error() string        { return fmt.Sprintf("%s: HTTP %d", ErrSource, e.Status) }
func (e *SourceError) Is(target error) bool { return target == ErrSource }

type GitHubSource struct{ client *http.Client }

func NewGitHubSource(client *http.Client) *GitHubSource {
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	copy := *client
	copy.Jar = nil // Rule distribution never receives a private tracker's session.
	if copy.Timeout == 0 {
		copy.Timeout = 2 * time.Minute
	}
	previous := copy.CheckRedirect
	copy.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return ErrSource
		}
		if request.URL.Scheme != "https" || request.URL.User != nil || request.URL.Port() != "" && request.URL.Port() != "443" {
			return ErrSource
		}
		switch strings.ToLower(request.URL.Hostname()) {
		case "api.github.com", "codeload.github.com", "github.com":
		default:
			return ErrSource
		}
		request.Header.Del("Authorization")
		request.Header.Del("Cookie")
		if previous != nil {
			return previous(request, via)
		}
		return nil
	}
	return &GitHubSource{client: &copy}
}

func (s *GitHubSource) Latest(ctx context.Context, etag string) (Revision, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceAPI, nil)
	if err != nil {
		return Revision{}, false, ErrSource
	}
	request.Header.Set("User-Agent", "Moji-Indexer-Definitions")
	request.Header.Set("Accept", "application/vnd.github+json")
	if len(etag) <= 1024 && !strings.ContainsAny(etag, "\r\n") {
		request.Header.Set("If-None-Match", etag)
	}
	response, err := s.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return Revision{}, false, ctx.Err()
		}
		return Revision{}, false, ErrSource
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotModified {
		return Revision{ETag: etag}, true, nil
	}
	if response.StatusCode != http.StatusOK {
		return Revision{}, false, sourceHTTPError(response)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return Revision{}, false, ErrSource
	}
	var records []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(data, &records); err != nil || len(records) != 1 || !ValidRevision(records[0].SHA) {
		return Revision{}, false, ErrSource
	}
	newETag := response.Header.Get("ETag")
	if len(newETag) > 1024 {
		newETag = ""
	}
	return Revision{Commit: records[0].SHA, ETag: newETag, PublishedAt: records[0].Commit.Committer.Date}, false, nil
}

func (s *GitHubSource) Download(ctx context.Context, revision string, writer io.Writer) error {
	if !ValidRevision(revision) {
		return ErrInvalidBundle
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceArchive+revision, nil)
	if err != nil {
		return ErrSource
	}
	request.Header.Set("User-Agent", "Moji-Indexer-Definitions")
	response, err := s.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrSource
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return sourceHTTPError(response)
	}
	if response.ContentLength > MaxArchiveBytes {
		return ErrBudget
	}
	count, err := io.Copy(writer, io.LimitReader(response.Body, MaxArchiveBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	if count > MaxArchiveBytes {
		return ErrBudget
	}
	return nil
}

func sourceHTTPError(response *http.Response) error {
	var retry time.Duration
	value := response.Header.Get("Retry-After")
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		if seconds > 86400 {
			seconds = 86400
		}
		retry = time.Duration(seconds) * time.Second
	} else if instant, err := http.ParseTime(value); err == nil {
		retry = time.Until(instant)
		if retry > 24*time.Hour {
			retry = 24 * time.Hour
		}
		if retry < 0 {
			retry = 0
		}
	}
	return &SourceError{Status: response.StatusCode, RetryAfter: retry}
}
