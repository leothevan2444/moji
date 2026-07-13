import { describe, expect, it } from "vitest";
import {
  DiscoverConfigDocumentDocument,
  HomePageDocumentDocument,
  HomeServiceStatusDocument,
  PerformersConfigDocumentDocument,
  SettingsPageDocumentDocument,
  StatsPageDocumentDocument,
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
    expect(rootFields(TasksOverviewDocumentDocument)).toEqual(["dashboardStats", "settings", "tasks"]);
    expect(rootFields(SettingsPageDocumentDocument)).toEqual(["settings", "settingsStatus", "version"]);
    expect(rootFields(StatsPageDocumentDocument)).toEqual(["dashboardStats"]);
    expect(rootFields(DiscoverConfigDocumentDocument)).toEqual(["settings"]);
    expect(rootFields(PerformersConfigDocumentDocument)).toEqual(["settings"]);
  });
});
