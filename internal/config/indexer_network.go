package config

import (
	"context"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

var ErrIndexerNetworkInUse = errors.New("INDEXER_NETWORK_PROFILE_IN_USE")

type IndexerNetworkTimeoutPatch struct{ RequestTimeoutSeconds, SiteTimeoutSeconds, ChallengeTimeoutSeconds *int }
type IndexerNetworkProfilePatch struct {
	ProxyURL          *string
	ClearProxy        bool
	FlareSolverrURL   *string
	ClearFlareSolverr bool
}

// Network changes share the config writer and return the revision of the exact
// acknowledged write, rather than a later independent configuration snapshot.
func (s *Store) UpdateIndexerNetworkTimeouts(ctx context.Context, expected string, patch IndexerNetworkTimeoutPatch) (*Config, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if expected == "" || configDigest(s.saved) != expected {
		return nil, "", ErrConfigRevision
	}
	cfg := s.cfg.Indexers.Clone()
	if patch.RequestTimeoutSeconds != nil {
		cfg.Builtin.RequestTimeoutSeconds = *patch.RequestTimeoutSeconds
	}
	if patch.SiteTimeoutSeconds != nil {
		cfg.Builtin.SiteTimeoutSeconds = *patch.SiteTimeoutSeconds
	}
	if patch.ChallengeTimeoutSeconds != nil {
		cfg.Builtin.ChallengeTimeoutSeconds = *patch.ChallengeTimeoutSeconds
	}
	if err := cfg.Validate(); err != nil {
		return nil, "", fmt.Errorf("%w: invalid network timeout", ErrInvalidSettings)
	}
	next, err := s.mutateIndexerNodeLocked(func(indexers *yaml.Node) error {
		builtin := ensureMapValue(mapValue(indexers, "builtin"))
		if patch.RequestTimeoutSeconds != nil {
			setNodeValue(builtin, "request_timeout_seconds", *patch.RequestTimeoutSeconds)
		}
		if patch.SiteTimeoutSeconds != nil {
			setNodeValue(builtin, "site_timeout_seconds", *patch.SiteTimeoutSeconds)
		}
		if patch.ChallengeTimeoutSeconds != nil {
			setNodeValue(builtin, "challenge_timeout_seconds", *patch.ChallengeTimeoutSeconds)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return next, configDigest(s.saved), nil
}

func (s *Store) SaveIndexerNetworkProfile(ctx context.Context, expected, id string, create bool, patch IndexerNetworkProfilePatch) (*Config, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if expected == "" || configDigest(s.saved) != expected {
		return nil, "", ErrConfigRevision
	}
	if id == "direct" || !indexerName.MatchString(id) || patch.ClearProxy && patch.ProxyURL != nil || patch.ClearFlareSolverr && patch.FlareSolverrURL != nil {
		return nil, "", ErrInvalidSettings
	}
	cfg := s.cfg.Indexers.Clone()
	profile, exists := cfg.Builtin.NetworkProfiles[id]
	if create == exists {
		return nil, "", ErrInvalidSettings
	}
	if patch.ClearProxy {
		profile.ProxyURL = ""
	}
	// Blank credential input retains the configured proxy; clearing is explicit.
	if patch.ProxyURL != nil && *patch.ProxyURL != "" {
		profile.ProxyURL = *patch.ProxyURL
	}
	if patch.ClearFlareSolverr {
		profile.FlareSolverrURL = ""
	}
	if patch.FlareSolverrURL != nil && *patch.FlareSolverrURL != "" {
		profile.FlareSolverrURL = *patch.FlareSolverrURL
	}
	if create && profile.ProxyURL == "" && profile.FlareSolverrURL == "" {
		return nil, "", ErrInvalidSettings
	}
	cfg.Builtin.NetworkProfiles[id] = profile
	if err := cfg.Validate(); err != nil {
		return nil, "", fmt.Errorf("%w: invalid network profile", ErrInvalidSettings)
	}
	next, err := s.mutateIndexerNodeLocked(func(indexers *yaml.Node) error {
		setNodeValue(ensureMapValue(mapValue(ensureMapValue(mapValue(indexers, "builtin")), "network_profiles")), id, profile)
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return next, configDigest(s.saved), nil
}

func (s *Store) RemoveIndexerNetworkProfile(ctx context.Context, expected, id string) (*Config, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if expected == "" || configDigest(s.saved) != expected {
		return nil, "", ErrConfigRevision
	}
	if id == "direct" {
		return nil, "", ErrInvalidSettings
	}
	if _, ok := s.cfg.Indexers.Builtin.NetworkProfiles[id]; !ok {
		return nil, "", ErrInvalidSettings
	}
	for _, instance := range s.cfg.Indexers.Builtin.Instances {
		if instance.NetworkProfile == id {
			return nil, "", ErrIndexerNetworkInUse
		}
	}
	next, err := s.mutateIndexerNodeLocked(func(indexers *yaml.Node) error {
		deleteMapKey(ensureMapValue(mapValue(ensureMapValue(mapValue(indexers, "builtin")), "network_profiles")), id)
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return next, configDigest(s.saved), nil
}
