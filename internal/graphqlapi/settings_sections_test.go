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
	if _, err := settingsSectionInput(model.UpdateSettingsSectionInput{Section: model.SettingsSectionIndexers, QueryOrder: []string{"b", "a"}}); err == nil {
		t.Fatal("legacy indexer order accepted through resource settings")
	}
	if _, err := settingsSectionInput(model.UpdateSettingsSectionInput{Section: model.SettingsSectionMetadata, MetadataPriority: []string{}}); err != nil {
		t.Fatal("cannot clear priority", err)
	}
}
