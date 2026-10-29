package graphqlapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/pkg/jackett"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
	"github.com/leothevan2444/moji/pkg/stash"
)

// Tests use a fresh client and a credential snapshot. They never mutate the
// configured clients, sessions, settings or runtime readiness observations.
func testServiceConnection(ctx context.Context, input model.TestServiceConnectionInput) *model.ServiceConnectionTestResult {
	parsed, err := url.Parse(strings.TrimSpace(input.URL))
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return &model.ServiceConnectionTestResult{Code: model.ServiceConnectionTestCodeInvalidURL}
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	endpoint := strings.TrimRight(parsed.String(), "/")
	value := func(v *string) string {
		if v == nil {
			return ""
		}
		return *v
	}
	switch input.Service {
	case model.SettingsConnectionServiceStash:
		cfg := config.StashConfig{URL: endpoint, APIKey: value(input.APIKey)}
		client := stash.NewClient(func() stash.Config { return stash.Config{URL: cfg.GraphQLEndpoint(), APIKey: cfg.APIKey} })
		defer client.CloseIdleConnections()
		version, e := client.GetVersion(ctx)
		err = e
		if err == nil && (version == nil || (version.Version == nil && version.Hash == "")) {
			err = errors.New("invalid service response")
		}
	case model.SettingsConnectionServiceJackett:
		// The management password is for Jackett's dashboard; the API uses its key.
		client := jackett.NewClient(endpoint, value(input.APIKey), "")
		defer client.CloseIdleConnections()
		sites, e := client.GetAPIIndexersContext(ctx)
		err = e
		if err == nil && sites == nil {
			err = errors.New("invalid service response")
		}
	case model.SettingsConnectionServiceQbittorrent:
		cfg := qbittorrent.Config{URL: endpoint, Username: value(input.Username), Password: value(input.Password)}
		client := qbittorrent.NewClient(func() qbittorrent.Config { return cfg })
		defer client.CloseIdleConnections()
		if cfg.Username != "" || cfg.Password != "" {
			err = client.Login(ctx, cfg.Username, cfg.Password)
		}
		if err == nil {
			version, e := client.GetApplicationVersion(ctx)
			err = e
			if err == nil && !qbVersionPattern.MatchString(strings.TrimSpace(version)) {
				err = errors.New("invalid service response")
			}
		}
	default:
		err = fmt.Errorf("unsupported service")
	}
	code := model.ServiceConnectionTestCodeOk
	if err != nil {
		code = model.ServiceConnectionTestCodeFailed
		var timeout net.Error
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.As(err, &timeout) && timeout.Timeout() {
			code = model.ServiceConnectionTestCodeTimeout
		}
	}
	// No upstream body, URL, credential or cookie is returned or logged.
	return &model.ServiceConnectionTestResult{Code: code}
}

var qbVersionPattern = regexp.MustCompile(`^v?\d+\.\d+(\.\d+)?[A-Za-z0-9.+_-]*$`)
