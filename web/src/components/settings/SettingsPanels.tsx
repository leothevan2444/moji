import { createContext, useContext, type ReactNode } from "react";
import type { SettingsTab } from "../../types";

export const SETTINGS_PANELS = {
  indexers: ["connection", "queryOrder"],
  downloader: ["connection", "speed", "cleanup"],
  stash: ["connection", "delivery", "metadata"],
  tasks: ["schedule", "releases", "resources", "deletion"],
  system: ["general", "imageCache"],
  logs: [],
  about: []
} as const satisfies Record<SettingsTab, readonly string[]>;

export const SettingsPanelContext = createContext<string | null>(null);
export const useSettingsPanel = () => useContext(SettingsPanelContext);

// Keep each form mounted while changing subpages so local edits, errors and
// in-flight autosaves retain their original ownership and lifecycle.
export function SettingsSubpanel({ id, children }: { id: string | string[]; children: ReactNode }) {
  const active = useSettingsPanel();
  const visible = active === null || (Array.isArray(id) ? id.includes(active) : active === id);
  return <div className="settings-subpanel" hidden={!visible}>{children}</div>;
}

export function SettingsGroup({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  return <fieldset className="settings-group"><legend>{title}</legend>{description ? <p className="settings-note">{description}</p> : null}<div className="settings-group__fields">{children}</div></fieldset>;
}
