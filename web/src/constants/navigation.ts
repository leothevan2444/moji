import type { SettingsTab } from "../types";

export const NAV_ITEMS = [
  { labelKey: "navigation.home", to: "/", end: true },
  { labelKey: "navigation.tasks", to: "/tasks", end: false },
  { labelKey: "navigation.performers", to: "/performers", end: false },
  { labelKey: "navigation.discover", to: "/discover", end: false }
] as const;

export const SETTINGS_TABS: SettingsTab[] = ["connections", "ingest", "automation", "system", "logs", "about"];
