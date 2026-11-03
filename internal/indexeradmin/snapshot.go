package indexeradmin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
)

// BuiltinSnapshot owns the conversion shared by production sourcing and draft
// diagnostics. Its revision remains independent of temporary tests and jobs.
func BuiltinSnapshot(cfg *config.Config, id string, source config.BuiltinIndexerInstance) indexer.InstanceSnapshot {
	profileID := source.NetworkProfile
	if profileID == "" {
		profileID = "direct"
	}
	profile := cfg.Indexers.Builtin.NetworkProfiles[profileID]
	instance := indexer.InstanceSnapshot{ID: id, Kind: indexer.Builtin, Name: source.Name, Enabled: source.Enabled, DefinitionID: source.Definition, BaseURL: source.BaseURL, TrustedOrigins: append([]string(nil), source.TrustedOrigins...), Settings: map[string]indexer.SettingValue{}, ChallengeTimeout: time.Duration(cfg.Indexers.Builtin.ChallengeTimeoutSeconds) * time.Second, RequestTimeout: time.Duration(cfg.Indexers.Builtin.RequestTimeoutSeconds) * time.Second, SiteTimeout: time.Duration(cfg.Indexers.Builtin.SiteTimeoutSeconds) * time.Second, Network: indexer.NetworkProfile{ID: profileID, Revision: snapshotFingerprint(profile), ProxyURL: profile.ProxyURL, FlareSolverrURL: profile.FlareSolverrURL}}
	if instance.Name == "" {
		instance.Name = source.Definition
	}
	for name, value := range source.Settings {
		instance.Settings[name] = indexer.SettingValue{String: value.String, Boolean: value.Boolean, Number: value.Number}
	}
	instance.ConfigRevision = snapshotFingerprint(instance)
	return instance
}

func snapshotFingerprint(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
