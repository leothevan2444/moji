package graphqlapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexerdownloads"
)

type preparedDownloadsAPI struct {
	calls    int
	resource string
	err      error
}

func (d *preparedDownloadsAPI) Prepare(_ context.Context, id string) (indexerdownloads.Receipt, error) {
	d.calls++
	d.resource = id
	return indexerdownloads.Receipt{Handle: strings.Repeat("a", 64), Kind: "TORRENT", ExpiresAt: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)}, d.err
}
func TestPrepareDownloadGraphQLUsesServerMembershipAndReturnsOnlyHandle(t *testing.T) {
	source := &searchAPISource{}
	downloads := &preparedDownloadsAPI{}
	r := NewResolver(nil, nil, nil, "test")
	r.SearchRuns = source
	r.TorrentDownloads = downloads
	type responseType struct {
		Errors []struct {
			Message    string
			Extensions struct{ Code string }
		}
		Data struct {
			Download *struct{ Kind, DownloadURL, ExpiresAt string } `json:"prepareTorrentDownload"`
		}
	}
	for _, input := range []string{`searchId:"other-run",candidateId:"candidate"`, `searchId:"run",candidateId:"other-candidate"`} {
		var response responseType
		executeDefinitionGraphQLInto(t, r, `mutation{prepareTorrentDownload(input:{`+input+`}){kind downloadUrl expiresAt}}`, &response)
		if len(response.Errors) == 0 || downloads.calls != 0 || response.Data.Download != nil {
			t.Fatal("unowned candidate or client locator reached acquisition", response)
		}
	}
	// Unknown locator fields are rejected by GraphQL before the resolver runs.
	payload, _ := json.Marshal(map[string]string{"query": `mutation{prepareTorrentDownload(input:{searchId:"run",candidateId:"candidate",url:"https://private.example/grab"}){kind}}`})
	server := NewGraphQLServer(generated.NewExecutableSchema(generated.Config{Resolvers: r}))
	request := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	reply := httptest.NewRecorder()
	server.ServeHTTP(reply, request)
	if reply.Code != http.StatusUnprocessableEntity || downloads.calls != 0 {
		t.Fatal("client locator escaped schema validation", reply.Code)
	}
	var response responseType
	executeDefinitionGraphQLInto(t, r, `mutation{prepareTorrentDownload(input:{searchId:"run",candidateId:"candidate"}){kind downloadUrl expiresAt}}`, &response)
	if len(response.Errors) != 0 || downloads.calls != 1 || downloads.resource != "server-private-resource" || response.Data.Download == nil {
		t.Fatal("membership did not control acquisition", response)
	}
	result := response.Data.Download
	if result.Kind != "TORRENT" || result.DownloadURL != "/api/indexers/downloads/"+strings.Repeat("a", 64) || result.ExpiresAt != "2030-01-01T00:00:00Z" || strings.Contains(result.DownloadURL, "private") {
		t.Fatal("source metadata escaped public receipt", result)
	}
	downloads.err = indexer.ErrAuthentication
	response = responseType{}
	executeDefinitionGraphQLInto(t, r, `mutation{prepareTorrentDownload(input:{searchId:"run",candidateId:"candidate"}){kind}}`, &response)
	if len(response.Errors) != 1 || response.Errors[0].Extensions.Code != indexer.ErrAuthentication.Error() || response.Data.Download != nil {
		t.Fatal("acquisition failure was presented as prepared", response)
	}
}
func TestPrepareDownloadDoesNotWorkWithoutEitherDependency(t *testing.T) {
	r := NewResolver(nil, nil, nil, "test")
	r.TorrentDownloads = &preparedDownloadsAPI{}
	// Direct helper checks capability availability before consulting a source.
	_, err := r.prepareTorrentDownload(context.Background(), model.PrepareTorrentDownloadInput{SearchID: "run", CandidateID: "candidate"})
	if !errors.Is(err, indexer.ErrNotConfigured) {
		t.Fatal(err)
	}
	r.SearchRuns = &searchAPISource{}
	r.TorrentDownloads = nil
	_, err = r.prepareTorrentDownload(context.Background(), model.PrepareTorrentDownloadInput{SearchID: "run", CandidateID: "candidate"})
	if !errors.Is(err, indexer.ErrNotConfigured) {
		t.Fatal(err)
	}
}
