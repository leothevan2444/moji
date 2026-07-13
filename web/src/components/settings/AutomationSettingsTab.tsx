import { useTranslation } from "react-i18next";
import { useOutletContext } from "react-router";
import { useQuery } from "urql";
import type { AppOutletContext } from "../../app/AppLayout";
import { AutomationSettingsTabDocument, type Settings, type SettingsStatus } from "../../graphql/generated/graphql";
import { AutomationSettingsEditor } from "./SettingsPanel";
import { SettingsError, SettingsLoading } from "./SettingsTabCommon";
import "../../styles/settings.scss";

export default function AutomationSettingsTab() {
  const { t } = useTranslation();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const [{ data, fetching, error }, refresh] = useQuery({ query: AutomationSettingsTabDocument, requestPolicy: "cache-first" });
  if (fetching && !data) return <SettingsLoading title={t("settings.tabs.automation")} />;
  if (error && !data) return <SettingsError title={t("settings.tabs.automation")} error={error} onRetry={() => refresh({ requestPolicy: "network-only" })} />;
  if (!data) return null;

  const settings = {
    stash: { configured: false, url: "", apiKeyConfigured: false, apiKey: "" },
    ingest: { deliveryMode: "PATH_MAP", downloads: { qbRoot: "", mojiRoot: "" }, library: { mojiRoot: "", stashRoot: "" }, transfer: { action: "COPY" } },
    jackett: { configured: false, url: "", apiKeyConfigured: false, apiKey: "", passwordConfigured: false, password: "" },
    qbittorrent: { configured: false, url: "", username: "", usernameConfigured: false, passwordConfigured: false, password: "", defaultSavePath: "", category: "", tags: "" },
    automation: data.settings.automation,
    system: { taskDeletePolicy: "KEEP_ONLY", imageCache: { enabled: false, maxSizeMb: 0, retentionDays: 0 } }
  } as unknown as Settings;
  const status = {
    stash: { configured: false, ready: false }, jackett: { configured: false, ready: false }, qbittorrent: { configured: false, ready: false },
    automation: data.settingsStatus.automation, stashBox: data.settingsStatus.stashBox,
    ingest: { configured: false }, imageCache: { usedBytes: 0, entryCount: 0, cacheDirectory: "", lastCleanupAt: null, lastError: null },
    stashLibraries: [], stashLibrariesLoadError: null,
    stashStats: { version: null, sceneCount: null, pendingMojiScanCount: 0, lastError: null, okAt: null },
    jackettStats: { indexerCount: 0, configuredIndexerCount: 0, lastIndexerLatencyMs: 0, lastIndexerError: null, lastIndexerSearchAt: null, lastError: null, okAt: null },
    qbittorrentStats: { downloadSpeed: 0, uploadSpeed: 0, activeTorrentCount: 0, connectionStatus: "", altSpeedLimitEnabled: false, lastError: null, okAt: null }
  } as unknown as SettingsStatus;

  return <AutomationSettingsEditor runtimeSettings={settings} runtimeStatus={status} appVersion="" drawer="settings" renderedDrawer="settings" pushToast={pushToast} refreshAutomation={refresh} />;
}
