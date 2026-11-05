package indexer

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"

	"github.com/leothevan2444/moji/internal/cardigann"
)

// CheckConnection makes a bounded connectivity request without authenticating,
// searching or acquiring resources. Jackett caps verifies its per-site API only.
func (n *Network) CheckConnection(ctx context.Context, instance InstanceSnapshot, certificates []string) error {
	var session *HTTPSession
	var err error
	target := instance.BaseURL
	if instance.Kind == Jackett {
		if instance.RemoteID == "" || instance.JackettAPIKey == "" {
			return ErrNotConfigured
		}
		session, err = n.newExternalSession(instance)
		target = strings.TrimRight(instance.BaseURL, "/") + "/api/v2.0/indexers/" + url.PathEscape(instance.RemoteID) + "/results/torznab/api"
		values := url.Values{"t": {"caps"}, "apikey": {instance.JackettAPIKey}}
		target += "?" + values.Encode()
	} else {
		session, err = n.NewSession(instance, certificates)
	}
	if err != nil {
		return err
	}
	defer session.Close()
	response, err := session.Do(ctx, cardigann.Request{Method: http.MethodGet, URL: target})
	if err != nil {
		return err
	}
	if response.Status == http.StatusUnauthorized || response.Status == http.StatusForbidden {
		return ErrAuthentication
	}
	if response.Status < 200 || response.Status >= 300 {
		return ErrTransport
	}
	if instance.Kind == Jackett {
		var document struct{ XMLName xml.Name }
		if xml.Unmarshal(response.Body, &document) != nil || document.XMLName.Local != "caps" {
			return ErrTransport
		}
	}
	return nil
}
