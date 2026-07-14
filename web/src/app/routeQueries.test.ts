import { describe, expect, it } from "vitest";
import {
  DiscoverConfigDocumentDocument,
  AboutSettingsTabDocument,
  AutomationSettingsTabDocument,
  ConnectionsSettingsDocument,
  ConnectionsStatusDocument,
  HomePageDocumentDocument,
  HomeServiceStatusDocument,
  IngestSettingsTabDocument,
  PerformersConfigDocumentDocument,
  StatsPageDocumentDocument,
  SystemSettingsTabDocument,
  TasksOverviewDocumentDocument
} from "../graphql/generated/graphql";

function rootFields(document: { definitions: readonly unknown[] }) {
  const operation = document.definitions.find((definition: any) => definition.kind === "OperationDefinition") as any;
  return operation.selectionSet.selections.map((selection: any) => selection.name.value).sort();
}

describe("route-owned GraphQL operations", () => {
  it("keeps each route query scoped to its surface", () => {
    expect(rootFields(HomePageDocumentDocument)).toEqual(["settings", "tasks"]);
    expect(rootFields(HomeServiceStatusDocument)).toEqual(["settingsStatus"]);
    expect(rootFields(TasksOverviewDocumentDocument)).toEqual(["dashboardStats", "settings", "settingsStatus", "tasks"]);
    expect(rootFields(ConnectionsSettingsDocument)).toEqual(["settings"]);
    expect(rootFields(ConnectionsStatusDocument)).toEqual(["settingsStatus"]);
    expect(rootFields(IngestSettingsTabDocument)).toEqual(["settings", "settingsStatus"]);
    expect(rootFields(AutomationSettingsTabDocument)).toEqual(["settings", "settingsStatus"]);
    expect(rootFields(SystemSettingsTabDocument)).toEqual(["settings", "settingsStatus"]);
    expect(rootFields(AboutSettingsTabDocument)).toEqual(["version"]);
    expect(rootFields(StatsPageDocumentDocument)).toEqual(["dashboardStats"]);
    expect(rootFields(DiscoverConfigDocumentDocument)).toEqual(["settings"]);
    expect(rootFields(PerformersConfigDocumentDocument)).toEqual(["settings"]);
  });
});
