package graphqlapi

import (
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"testing"
)

func TestSettingsSectionRejectsUnrelatedPayloads(t *testing.T) {
	for _, input := range []model.UpdateSettingsSectionInput{
		{Section: model.SettingsSectionMetadata},
		{Section: model.SettingsSectionMetadata, MetadataPriority: []string{}, QueryOrder: []string{}},
		{Section: model.SettingsSectionResources, MetadataPriority: []string{}},
		{Section: model.SettingsSectionAll, MetadataPriority: []string{}},
	} {
		if _, err := settingsSectionInput(input); err == nil {
			t.Fatalf("accepted invalid section payload: %+v", input)
		}
	}
	update, err := settingsSectionInput(model.UpdateSettingsSectionInput{Section: model.SettingsSectionIndexers, QueryOrder: []string{"b", "a"}})
	if err != nil || update.Section != "INDEXERS" || update.TorrentSelection.FastRules[0].IndexerPreference.TrackerIDs[0] != "b" {
		t.Fatalf("incorrect site order mapping: %+v %v", update, err)
	}
	if _, err := settingsSectionInput(model.UpdateSettingsSectionInput{Section: model.SettingsSectionMetadata, MetadataPriority: []string{}}); err != nil {
		t.Fatal("cannot clear priority", err)
	}
}
