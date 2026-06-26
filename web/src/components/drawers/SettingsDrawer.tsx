import type { OperationContext } from "urql";
import { SETTINGS_TABS } from "../../constants";
import type { DrawerKey, SettingsTab, ToastTone } from "../../types";
import { SettingsPanel } from "../settings/SettingsPanel";
import type { DashboardDocumentQuery } from "../../graphql/generated/graphql";

type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;
type RuntimeSettingsStatus = NonNullable<DashboardDocumentQuery["settingsStatus"]>;

interface SettingsDrawerProps {
  settingsTab: SettingsTab;
  onSettingsTabChange: (tab: SettingsTab) => void;
  runtimeSettings: RuntimeSettings | null;
  runtimeStatus: RuntimeSettingsStatus | null;
  appVersion: string;
  drawer: DrawerKey;
  renderedDrawer: Exclude<DrawerKey, null> | null;
  pushToast: (tone: ToastTone, message: string) => void;
  refreshDashboard: (opts?: Partial<OperationContext>) => void;
}

export function SettingsDrawer({
  settingsTab,
  onSettingsTabChange,
  runtimeSettings,
  runtimeStatus,
  appVersion,
  drawer,
  renderedDrawer,
  pushToast,
  refreshDashboard
}: SettingsDrawerProps) {
  return (
    <div className="drawer-stack">
      <div className="settings-tabs">
        {SETTINGS_TABS.map((item) => (
          <button
            key={item}
            type="button"
            className={`chip ${settingsTab === item ? "is-active" : ""}`}
            onClick={() => onSettingsTabChange(item)}
          >
            {item}
          </button>
        ))}
      </div>

      <SettingsPanel
        settingsTab={settingsTab}
        runtimeSettings={runtimeSettings}
        runtimeStatus={runtimeStatus}
        appVersion={appVersion}
        drawer={drawer}
        renderedDrawer={renderedDrawer}
        pushToast={pushToast}
        refreshDashboard={(opts) => { refreshDashboard(opts as Partial<OperationContext>); }}
      />
    </div>
  );
}
