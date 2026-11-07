package config

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"
)

type IndexerNetworkPatch struct {
	FlareSolverrMaxTimeoutMS                                *int
	RequestTimeoutSeconds, SiteTimeoutSeconds               *int
	ProxyEnabled, FlareSolverrEnabled                       *bool
	ProxyURL, ProxyUsername, ProxyPassword, FlareSolverrURL *string
	ClearProxyPassword                                      bool
}

func (s *Store) UpdateIndexerNetwork(ctx context.Context, expected string, patch IndexerNetworkPatch) (*Config, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if expected == "" || configDigest(s.saved) != expected {
		return nil, "", ErrConfigRevision
	}
	if patch.ClearProxyPassword && patch.ProxyPassword != nil {
		return nil, "", ErrInvalidSettings
	}
	cfg := s.cfg.Indexers.Clone()
	if patch.RequestTimeoutSeconds != nil {
		cfg.Builtin.RequestTimeoutSeconds = *patch.RequestTimeoutSeconds
	}
	if patch.SiteTimeoutSeconds != nil {
		cfg.Builtin.SiteTimeoutSeconds = *patch.SiteTimeoutSeconds
	}
	network := &cfg.Builtin.Network
	if patch.FlareSolverrMaxTimeoutMS != nil {
		network.FlareSolverr.MaxTimeoutMS = *patch.FlareSolverrMaxTimeoutMS
	}
	if patch.ProxyEnabled != nil {
		network.Proxy.Enabled = *patch.ProxyEnabled
	}
	if patch.FlareSolverrEnabled != nil {
		network.FlareSolverr.Enabled = *patch.FlareSolverrEnabled
	}
	if patch.ProxyURL != nil {
		network.Proxy.URL = *patch.ProxyURL
	}
	if patch.ProxyUsername != nil {
		network.Proxy.Username = *patch.ProxyUsername
	}
	if patch.ProxyPassword != nil && *patch.ProxyPassword != "" {
		network.Proxy.Password = *patch.ProxyPassword
	}
	if patch.ClearProxyPassword {
		network.Proxy.Password = ""
	}
	if patch.FlareSolverrURL != nil {
		network.FlareSolverr.URL = *patch.FlareSolverrURL
	}
	if err := cfg.Validate(); err != nil {
		return nil, "", fmt.Errorf("%w: invalid indexer network settings", ErrInvalidSettings)
	}
	next, err := s.mutateIndexerNodeLocked(func(indexers *yaml.Node) error {
		builtin := ensureMapValue(mapValue(indexers, "builtin"))
		setNodeValue(builtin, "network", network)
		if patch.RequestTimeoutSeconds != nil {
			setNodeValue(builtin, "request_timeout_seconds", *patch.RequestTimeoutSeconds)
		}
		if patch.SiteTimeoutSeconds != nil {
			setNodeValue(builtin, "site_timeout_seconds", *patch.SiteTimeoutSeconds)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return next, configDigest(s.saved), nil
}
