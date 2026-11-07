package main

import (
	"github.com/leothevan2444/moji/internal/config"
	"net/url"
)

func acceptanceNetworkPatch(proxy, solver string) config.IndexerNetworkPatch {
	enabledProxy, enabledSolver := proxy != "", solver != ""
	patch := config.IndexerNetworkPatch{ProxyEnabled: &enabledProxy, ProxyURL: &proxy, FlareSolverrEnabled: &enabledSolver, FlareSolverrURL: &solver}
	if u, err := url.Parse(proxy); err == nil && u.User != nil {
		user := u.User.Username()
		password, _ := u.User.Password()
		u.User = nil
		raw := u.String()
		patch.ProxyURL = &raw
		patch.ProxyUsername = &user
		patch.ProxyPassword = &password
	}
	return patch
}
