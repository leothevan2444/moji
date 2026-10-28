import { describe, expect, it } from "vitest";
import {
  DiscoverConfigDocumentDocument,
  AboutSettingsTabDocument,
  SettingsPolicySectionDocument,
  ServiceConnectionSettingsDocument,
  ServiceConnectionStatusDocument,
  HomePageDocumentDocument,
  HomeServiceStatusDocument,
  LibraryDeliverySettingsDocument,
  PerformersConfigDocumentDocument,
  StatsPageDocumentDocument,
  SystemSettingsTabDocument,
  TasksWorkspaceDocument
} from "../graphql/generated/graphql";

function rootFields(document: { definitions: readonly unknown[] }) {
  const operation = document.definitions.find((definition: any) => definition.kind === "OperationDefinition") as any;
  return operation.selectionSet.selections.map((selection: any) => selection.name.value).sort();
}

describe("route-owned GraphQL operations", () => {
  it("keeps each route query scoped to its surface", () => {
    expect(rootFields(HomePageDocumentDocument)).toEqual(["homeTaskSnapshot", "settings"]);
    expect(rootFields(HomeServiceStatusDocument)).toEqual(["settingsStatus"]);
    expect(rootFields(TasksWorkspaceDocument)).toEqual(["settings", "settingsStatus", "taskPage"]);
    expect(rootFields(ServiceConnectionSettingsDocument)).toEqual(["settings"]);
    expect(rootFields(ServiceConnectionStatusDocument)).toEqual(["settingsStatus"]);
    expect(rootFields(LibraryDeliverySettingsDocument)).toEqual(["settings", "settingsStatus"]);
    expect(rootFields(SettingsPolicySectionDocument)).toEqual(["settings", "settingsStatus"]);
    expect(rootFields(SystemSettingsTabDocument)).toEqual(["settings", "settingsStatus"]);
    expect(rootFields(AboutSettingsTabDocument)).toEqual(["version"]);
    expect(rootFields(StatsPageDocumentDocument)).toEqual(["dashboardStats"]);
    expect(rootFields(DiscoverConfigDocumentDocument)).toEqual(["settings"]);
    expect(rootFields(PerformersConfigDocumentDocument)).toEqual(["settings"]);
  });
});
