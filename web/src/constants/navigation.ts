import { faHouse, faListCheck, faUserGroup, faCompass } from "@fortawesome/free-solid-svg-icons";
import type { SettingsTab } from "../types";

export const NAV_ITEMS = [
  { labelKey: "navigation.home", icon: faHouse, to: "/", end: true },
  { labelKey: "navigation.tasks", icon: faListCheck, to: "/tasks", end: false },
  { labelKey: "navigation.performers", icon: faUserGroup, to: "/performers", end: false },
  { labelKey: "navigation.discover", icon: faCompass, to: "/discover", end: false }
] as const;

export const SETTINGS_TABS: SettingsTab[] = ["indexers", "downloader", "stash", "tasks", "system", "logs", "about"];
