// moji-config validates and migrates configuration without initializing clients
// or opening the application databases. It never prints configuration values.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/durablefile"
	"os"
	"path/filepath"
)

func main() {
	source := flag.String("config", config.DefaultPath(), "configuration to validate")
	output := flag.String("output", "", "write migrated copy to this new file; source is unchanged")
	flag.Parse()
	if err := run(*source, *output); err != nil {
		fmt.Fprintln(os.Stderr, "configuration operation failed:", err)
		os.Exit(1)
	}
	fmt.Println("configuration valid; effective settings preserved")
}

func effective(cfg *config.Config) []byte {
	copy := *cfg
	copy.Indexers = copy.Indexers.Clone()
	copy.Indexers.QueryOrder = append([]string{}, copy.Indexers.QueryOrder...)
	copy.Indexers.Jackett.ExcludedIndexers = append([]string{}, copy.Indexers.Jackett.ExcludedIndexers...)
	copy.System.Language = copy.System.EffectiveLanguage()
	copy.System.ImageCache = copy.System.ImageCache.Normalize()
	enabled := copy.System.ImageCache.EffectiveEnabled()
	copy.System.ImageCache.Enabled = &enabled
	copy.System.StashBoxDataCache = copy.System.StashBoxDataCache.Normalize()
	copy.Automation.SourcingRetryIntervalHours = copy.Automation.EffectiveSourcingRetryIntervalHours()
	// Site order is independent of the old candidate-ranking enable switch.
	copy.Automation.TorrentSelection.FastRules.IndexerPreference.Enabled = true
	order := []config.TorrentSelectionRuleType{}
	for _, rule := range copy.Automation.TorrentSelection.FastRuleOrder {
		if rule != config.TorrentSelectionRuleTypeIndexerPreference {
			order = append(order, rule)
		}
	}
	copy.Automation.TorrentSelection.FastRuleOrder = order
	data, _ := json.Marshal(copy)
	return data
}
func run(source, output string) error {
	before, err := config.LoadFromPath(source)
	if err != nil {
		return err
	}
	if output == "" {
		return nil
	}
	sourceAbs, _ := filepath.Abs(source)
	outputAbs, _ := filepath.Abs(output)
	if sourceAbs == outputAbs {
		return fmt.Errorf("output must differ from source; validate a copy before replacing the actual config")
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = file.Write(raw); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	store, err := config.OpenStore(output)
	if err != nil {
		return err
	}
	if err = store.Migrate(); err != nil {
		return err
	}
	after, err := config.LoadFromPath(output)
	if err != nil {
		return err
	}
	if !bytes.Equal(effective(before), effective(after)) {
		return fmt.Errorf("effective values differ after migration; source has not been changed")
	}
	migrated, err := os.ReadFile(output)
	if err != nil {
		return err
	}
	return durablefile.Write(output, migrated, 0600)
}
