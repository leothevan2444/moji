import { FormEvent, useEffect, useState } from "react";
import { useMutation, useQuery } from "urql";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faArrowDown,
  faArrowUp,
  faEye,
  faEyeSlash,
  faGripVertical,
  faRotate
} from "@fortawesome/free-solid-svg-icons";
import {
  LogLevel,
  LogsDocumentDocument,
  RefreshSubscriptionStashBoxesDocumentDocument,
  UpdateAutomationSettingsDocumentDocument,
  UpdateIngestSettingsDocumentDocument,
  UpdateJackettSettingsDocumentDocument,
  UpdateQBittorrentSettingsDocumentDocument,
  UpdateStashSettingsDocumentDocument,
  UpdateSubscriptionSettingsDocumentDocument,
  type DashboardDocumentQuery,
  type LogsDocumentQuery,
  type LogsDocumentQueryVariables,
  type RefreshSubscriptionStashBoxesDocumentMutation,
  type UpdateAutomationSettingsDocumentMutation,
  type UpdateAutomationSettingsDocumentMutationVariables,
  type UpdateIngestSettingsDocumentMutation,
  type UpdateIngestSettingsDocumentMutationVariables,
  type UpdateJackettSettingsDocumentMutation,
  type UpdateJackettSettingsDocumentMutationVariables,
  type UpdateQBittorrentSettingsDocumentMutation,
  type UpdateQBittorrentSettingsDocumentMutationVariables,
  type UpdateStashSettingsDocumentMutation,
  type UpdateStashSettingsDocumentMutationVariables,
  type UpdateSubscriptionSettingsDocumentMutation,
  type UpdateSubscriptionSettingsDocumentMutationVariables
} from "../../graphql/generated/graphql";
import type { SettingsTab, ToastTone } from "../../types";
import {
  EMPTY_AUTOMATION_FORM,
  EMPTY_INGEST_FORM,
  EMPTY_JACKETT_FORM,
  EMPTY_QBITTORRENT_FORM,
  EMPTY_STASH_FORM,
  EMPTY_SUBSCRIPTION_FORM,
  LOG_LEVEL_OPTIONS
} from "../../constants";
import { serviceStatus } from "../../utils";
import { describeQueryError } from "../../services/queryError";
import { formatDateTime, formatLogEntries } from "../../utils";

type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;
type RuntimeSettingsStatus = NonNullable<DashboardDocumentQuery["settingsStatus"]>;

interface SettingsPanelProps {
  settingsTab: SettingsTab;
  runtimeSettings: RuntimeSettings | null;
  runtimeStatus: RuntimeSettingsStatus | null;
  appVersion: string;
  drawer: string | null;
  renderedDrawer: string | null;
  pushToast: (tone: ToastTone, message: string) => void;
  refreshDashboard: (opts?: Record<string, unknown>) => unknown;
}

export function SettingsPanel({
  settingsTab,
  runtimeSettings,
  runtimeStatus,
  appVersion,
  drawer,
  renderedDrawer,
  pushToast,
  refreshDashboard
}: SettingsPanelProps) {
  const [logsLevel, setLogsLevel] = useState<LogLevel>(LogLevel.Info);
  const [downloadingLogFile, setDownloadingLogFile] = useState(false);
  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const [visibleSecrets, setVisibleSecrets] = useState<{ stashApiKey: boolean; jackettApiKey: boolean; jackettPassword: boolean; qbittorrentPassword: boolean }>({
    stashApiKey: false,
    jackettApiKey: false,
    jackettPassword: false,
    qbittorrentPassword: false
  });
  const [stashForm, setStashForm] = useState(EMPTY_STASH_FORM);
  const [ingestForm, setIngestForm] = useState(EMPTY_INGEST_FORM);
  const [jackettForm, setJackettForm] = useState(EMPTY_JACKETT_FORM);
  const [qbittorrentForm, setQBittorrentForm] = useState(EMPTY_QBITTORRENT_FORM);
  const [automationForm, setAutomationForm] = useState(EMPTY_AUTOMATION_FORM);
  const [subscriptionForm, setSubscriptionForm] = useState(EMPTY_SUBSCRIPTION_FORM);

  const toggleSecret = (key: "stashApiKey" | "jackettApiKey" | "jackettPassword" | "qbittorrentPassword") => {
    setVisibleSecrets((current) => ({ ...current, [key]: !current[key] }));
  };

  const [{ data: logsData, fetching: fetchingLogs, error: logsError }, refreshLogs] = useQuery<
    LogsDocumentQuery,
    LogsDocumentQueryVariables
  >({
    query: LogsDocumentDocument,
    variables: {
      minLevel: logsLevel
    },
    pause: settingsTab !== "日志" || (drawer !== "settings" && renderedDrawer !== "settings")
  });
  const logs = logsData?.logs ?? [];

  const [{ fetching: updatingStash }, updateStashSettings] = useMutation<
    UpdateStashSettingsDocumentMutation,
    UpdateStashSettingsDocumentMutationVariables
  >(UpdateStashSettingsDocumentDocument);
  const [{ fetching: updatingIngest }, updateIngestSettings] = useMutation<
    UpdateIngestSettingsDocumentMutation,
    UpdateIngestSettingsDocumentMutationVariables
  >(UpdateIngestSettingsDocumentDocument);
  const [{ fetching: updatingJackett }, updateJackettSettings] = useMutation<
    UpdateJackettSettingsDocumentMutation,
    UpdateJackettSettingsDocumentMutationVariables
  >(UpdateJackettSettingsDocumentDocument);
  const [{ fetching: updatingQBittorrent }, updateQBittorrentSettings] = useMutation<
    UpdateQBittorrentSettingsDocumentMutation,
    UpdateQBittorrentSettingsDocumentMutationVariables
  >(UpdateQBittorrentSettingsDocumentDocument);
  const [{ fetching: updatingAutomation }, updateAutomationSettings] = useMutation<
    UpdateAutomationSettingsDocumentMutation,
    UpdateAutomationSettingsDocumentMutationVariables
  >(UpdateAutomationSettingsDocumentDocument);
  const [{ fetching: updatingSubscription }, updateSubscriptionSettings] = useMutation<
    UpdateSubscriptionSettingsDocumentMutation,
    UpdateSubscriptionSettingsDocumentMutationVariables
  >(UpdateSubscriptionSettingsDocumentDocument);
  const [{ fetching: refreshingStashBoxes }, refreshStashBoxesMutation] = useMutation<
    RefreshSubscriptionStashBoxesDocumentMutation
  >(RefreshSubscriptionStashBoxesDocumentDocument);

  useEffect(() => {
    if (!runtimeSettings) return;

    setStashForm({
      url: runtimeSettings.stash.url || "",
      apiKey: runtimeSettings.stash.apiKey ?? ""
    });
    setIngestForm({
      mode: runtimeSettings.ingest.mode || "SHARED_STORAGE",
      libraryPath: runtimeSettings.ingest.libraryPath || "",
      qbittorrentPathPrefix: runtimeSettings.ingest.qbittorrentPathPrefix || "",
      stashPathPrefix: runtimeSettings.ingest.stashPathPrefix || "",
      transferAction: runtimeSettings.ingest.transferAction || "",
      transferTargetPath: runtimeSettings.ingest.transferTargetPath || ""
    });
    setJackettForm({
      url: runtimeSettings.jackett.url || "",
      apiKey: runtimeSettings.jackett.apiKey ?? "",
      password: runtimeSettings.jackett.password ?? ""
    });
    setQBittorrentForm({
      url: runtimeSettings.qbittorrent.url || "",
      username: runtimeSettings.qbittorrent.username || "",
      password: runtimeSettings.qbittorrent.password ?? "",
      defaultSavePath: runtimeSettings.qbittorrent.defaultSavePath || "",
      category: runtimeSettings.qbittorrent.category || "",
      tags: runtimeSettings.qbittorrent.tags || ""
    });
    setAutomationForm({
      taskProgressSyncIntervalSeconds: String(runtimeSettings.automation.taskProgressSyncIntervalSeconds || 60),
      subscriptionPollIntervalSeconds: String(runtimeSettings.automation.subscriptionPollIntervalSeconds || 3600)
    });
    setSubscriptionForm({
      stashBoxEndpoints: [...(runtimeSettings.subscription.stashBoxEndpoints ?? [])]
    });
  }, [runtimeSettings]);

  useEffect(() => {
    const logsTabActive = settingsTab === "日志" && (drawer === "settings" || renderedDrawer === "settings");
    if (!logsTabActive) return;

    const timer = window.setInterval(() => {
      void refreshLogs({ requestPolicy: "network-only" });
    }, 5000);

    return () => window.clearInterval(timer);
  }, [drawer, renderedDrawer, refreshLogs, settingsTab]);

  const saveStashSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = await updateStashSettings({
      input: {
        url: stashForm.url.trim(),
        apiKey: stashForm.apiKey.trim()
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "Stash 设置已保存。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveIngestSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = await updateIngestSettings({
      input: {
        mode: ingestForm.mode.trim(),
        libraryPath: ingestForm.libraryPath.trim(),
        qbittorrentPathPrefix: ingestForm.qbittorrentPathPrefix.trim(),
        stashPathPrefix: ingestForm.stashPathPrefix.trim(),
        transferAction: ingestForm.transferAction.trim(),
        transferTargetPath: ingestForm.transferTargetPath.trim()
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "入库设置已保存。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveJackettSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = await updateJackettSettings({
      input: {
        url: jackettForm.url.trim(),
        apiKey: jackettForm.apiKey.trim(),
        password: jackettForm.password.trim()
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "Jackett 设置已保存。");
    // Mirror qBittorrent's pattern: clear the password field after a
    // successful save so the plaintext doesn't linger in component state.
    setJackettForm((current) => ({ ...current, password: "" }));
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveQBittorrentSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = await updateQBittorrentSettings({
      input: {
        url: qbittorrentForm.url.trim(),
        username: qbittorrentForm.username.trim(),
        password: qbittorrentForm.password.trim(),
        defaultSavePath: qbittorrentForm.defaultSavePath.trim(),
        category: qbittorrentForm.category.trim(),
        tags: qbittorrentForm.tags.trim()
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "qBittorrent 设置已保存。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveAutomationSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const taskProgressSyncIntervalSeconds = Number.parseInt(automationForm.taskProgressSyncIntervalSeconds.trim(), 10);
    const subscriptionPollIntervalSeconds = Number.parseInt(automationForm.subscriptionPollIntervalSeconds.trim(), 10);
    const result = await updateAutomationSettings({
      input: {
        taskProgressSyncIntervalSeconds: Number.isNaN(taskProgressSyncIntervalSeconds) ? 60 : taskProgressSyncIntervalSeconds,
        subscriptionPollIntervalSeconds: Number.isNaN(subscriptionPollIntervalSeconds) ? 3600 : subscriptionPollIntervalSeconds
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "自动化设置已保存。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveSubscriptionSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = await updateSubscriptionSettings({
      input: {
        stashBoxEndpoints: subscriptionForm.stashBoxEndpoints
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "Stash-Box 优先级已保存。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const refreshSubscriptionStashBoxes = async () => {
    const result = await refreshStashBoxesMutation({});
    if (result.error) {
      pushToast("tone-danger", `刷新 Stash-Box 失败：${describeQueryError(result.error)}`);
      return;
    }
    pushToast("tone-success", "Stash-Box 列表已刷新。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const handleCopyLogs = async () => {
    await navigator.clipboard.writeText(formatLogEntries(logs));
  };

  const handleDownloadCurrentLogFile = async () => {
    setDownloadingLogFile(true);
    try {
      const response = await fetch("/api/logs/current");
      if (!response.ok) throw new Error(`下载失败：HTTP ${response.status}`);
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "moji.log";
      link.click();
      URL.revokeObjectURL(url);
    } catch (error) {
      pushToast("tone-danger", error instanceof Error ? error.message : "下载当前日志文件失败。");
    } finally {
      setDownloadingLogFile(false);
    }
  };

  if (!runtimeSettings || !runtimeStatus) {
    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>{settingsTab}</h3>
        </div>
        <dl className="settings-grid">
          <div>
            <dt>当前状态</dt>
            <dd>等待后端返回设置数据</dd>
          </div>
        </dl>
      </article>
    );
  }

  if (settingsTab === "连接") {
    const stashStatus = serviceStatus(runtimeStatus.stash.configured, runtimeStatus.stash.enabled);
    const jackettStatus = serviceStatus(runtimeStatus.jackett.configured, runtimeStatus.jackett.enabled);
    const qbittorrentStatus = serviceStatus(runtimeStatus.qbittorrent.configured, runtimeStatus.qbittorrent.enabled);

    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>连接</h3>
        </div>

        <form className="settings-form" onSubmit={(event) => void saveStashSettings(event)}>
          <div className="settings-meta">
            <span>Stash</span>
            <span className={`status-chip ${stashStatus.tone}`}>{stashStatus.label}</span>
          </div>
          <label className="settings-field">
            <span>Stash URL</span>
            <input
              value={stashForm.url}
              onChange={(event) => setStashForm((current) => ({ ...current, url: event.target.value }))}
              placeholder="http://localhost:9999"
            />
          </label>
          <label className="settings-field">
            <span>API key</span>
            <div className="secret-input">
              <input
                className="secret-input__field"
                type={visibleSecrets.stashApiKey ? "text" : "password"}
                value={stashForm.apiKey}
                onChange={(event) => setStashForm((current) => ({ ...current, apiKey: event.target.value }))}
                autoComplete="off"
                spellCheck={false}
              />
              <button type="button" className="secret-input__toggle" onClick={() => toggleSecret("stashApiKey")}>
                <FontAwesomeIcon icon={visibleSecrets.stashApiKey ? faEyeSlash : faEye} aria-hidden="true" />
              </button>
            </div>
          </label>
          <div className="settings-actions">
            <button type="submit" disabled={updatingStash}>保存 Stash 连接</button>
          </div>
        </form>

        <form className="settings-form" onSubmit={(event) => void saveJackettSettings(event)}>
          <div className="settings-meta">
            <span>Jackett</span>
            <span className={`status-chip ${jackettStatus.tone}`}>{jackettStatus.label}</span>
          </div>
          <label className="settings-field">
            <span>Jackett URL</span>
            <input
              value={jackettForm.url}
              onChange={(event) => setJackettForm((current) => ({ ...current, url: event.target.value }))}
              placeholder="http://localhost:9117"
            />
          </label>
          <label className="settings-field">
            <span>API key</span>
            <div className="secret-input">
              <input
                className="secret-input__field"
                type={visibleSecrets.jackettApiKey ? "text" : "password"}
                value={jackettForm.apiKey}
                onChange={(event) => setJackettForm((current) => ({ ...current, apiKey: event.target.value }))}
                autoComplete="off"
                spellCheck={false}
              />
              <button type="button" className="secret-input__toggle" onClick={() => toggleSecret("jackettApiKey")}>
                <FontAwesomeIcon icon={visibleSecrets.jackettApiKey ? faEyeSlash : faEye} aria-hidden="true" />
              </button>
            </div>
          </label>
          <label className="settings-field">
            <span>
              Dashboard 密码
            </span>
            <div className="secret-input">
              <input
                className="secret-input__field"
                type={visibleSecrets.jackettPassword ? "text" : "password"}
                value={jackettForm.password}
                onChange={(event) => setJackettForm((current) => ({ ...current, password: event.target.value }))}
                placeholder="Jackett 管理界面登录密码"
                autoComplete="off"
                spellCheck={false}
              />
              <button type="button" className="secret-input__toggle" onClick={() => toggleSecret("jackettPassword")}>
                <FontAwesomeIcon icon={visibleSecrets.jackettPassword ? faEyeSlash : faEye} aria-hidden="true" />
              </button>
            </div>
          </label>
          <div className="settings-actions">
            <button type="submit" disabled={updatingJackett}>保存 Jackett 连接</button>
          </div>
        </form>

        <form className="settings-form" onSubmit={(event) => void saveQBittorrentSettings(event)}>
          <div className="settings-meta">
            <span>qBittorrent</span>
            <span className={`status-chip ${qbittorrentStatus.tone}`}>{qbittorrentStatus.label}</span>
          </div>
          <label className="settings-field">
            <span>qBittorrent URL</span>
            <input
              value={qbittorrentForm.url}
              onChange={(event) => setQBittorrentForm((current) => ({ ...current, url: event.target.value }))}
              placeholder="http://localhost:8080"
            />
          </label>
          <label className="settings-field">
            <span>用户名</span>
            <input
              value={qbittorrentForm.username}
              onChange={(event) => setQBittorrentForm((current) => ({ ...current, username: event.target.value }))}
              placeholder="admin"
            />
          </label>
          <label className="settings-field">
            <span>密码</span>
            <div className="secret-input">
              <input
                className="secret-input__field"
                type={visibleSecrets.qbittorrentPassword ? "text" : "password"}
                value={qbittorrentForm.password}
                onChange={(event) => setQBittorrentForm((current) => ({ ...current, password: event.target.value }))}
                autoComplete="off"
                spellCheck={false}
              />
              <button type="button" className="secret-input__toggle" onClick={() => toggleSecret("qbittorrentPassword")}>
                <FontAwesomeIcon icon={visibleSecrets.qbittorrentPassword ? faEyeSlash : faEye} aria-hidden="true" />
              </button>
            </div>
          </label>
          <label className="settings-field">
            <span>默认保存路径</span>
            <input
              value={qbittorrentForm.defaultSavePath}
              onChange={(event) => setQBittorrentForm((current) => ({ ...current, defaultSavePath: event.target.value }))}
              placeholder="/downloads"
            />
          </label>
          <label className="settings-field">
            <span>默认分类</span>
            <input
              value={qbittorrentForm.category}
              onChange={(event) => setQBittorrentForm((current) => ({ ...current, category: event.target.value }))}
              placeholder="moji"
            />
          </label>
          <label className="settings-field">
            <span>默认标签</span>
            <input
              value={qbittorrentForm.tags}
              onChange={(event) => setQBittorrentForm((current) => ({ ...current, tags: event.target.value }))}
              placeholder="auto"
            />
          </label>
          <div className="settings-actions">
            <button type="submit" disabled={updatingQBittorrent}>保存 qBittorrent 连接</button>
          </div>
        </form>
      </article>
    );
  }

  if (settingsTab === "入库") {
    const stashModeGuide = (() => {
      if (ingestForm.mode === "SHARED_STORAGE") {
        return {
          tone: "tone-info",
          title: "共享存储 / 路径映射",
          summary: "适用于 qBittorrent 和 Stash 共用同一批文件，只是挂载路径不同。",
          caution: "要求下载路径命中 qBittorrent 路径前缀；映射失败时不会自动退回整库扫描。"
        };
      }
      if (ingestForm.mode === "FILE_TRANSFER") {
        return {
          tone: "tone-warn",
          title: "文件搬运",
          summary: "由 Moji 在本地文件系统执行复制或移动，成功后再扫描目标文件。",
          caution: "目标目录已有同名文件时会直接失败，不覆盖也不自动重命名。"
        };
      }
      return {
        tone: "tone-danger",
        title: "整库扫描",
        summary: "始终扫描整个库目录，适合先跑通接入或无法稳定定位单文件时使用。",
        caution: "无法精确锁定本次下载文件，扫描范围也最大。"
      };
    })();

    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>入库</h3>
        </div>
        <form className="settings-form" onSubmit={(event) => void saveIngestSettings(event)}>
          <label className="settings-field">
            <span>工作方式</span>
            <select
              value={ingestForm.mode}
              onChange={(event) => setIngestForm((current) => ({ ...current, mode: event.target.value }))}
            >
              <option value="SHARED_STORAGE">共享存储 / 路径映射</option>
              <option value="FILE_TRANSFER">文件搬运</option>
              <option value="LIBRARY_SCAN">整库扫描</option>
            </select>
          </label>
          <section className={`settings-mode-guide ${stashModeGuide.tone}`}>
            <div className="settings-mode-guide__head">
              <strong>{stashModeGuide.title}</strong>
              <span>{stashModeGuide.summary}</span>
            </div>
            <p>{stashModeGuide.caution}</p>
          </section>

          {ingestForm.mode === "SHARED_STORAGE" ? (
            <>
              <label className="settings-field">
                <span>qBittorrent 路径前缀</span>
                <input
                  value={ingestForm.qbittorrentPathPrefix}
                  onChange={(event) => setIngestForm((current) => ({ ...current, qbittorrentPathPrefix: event.target.value }))}
                  placeholder="/downloads"
                />
              </label>
              <label className="settings-field">
                <span>Stash 路径前缀</span>
                <input
                  value={ingestForm.stashPathPrefix}
                  onChange={(event) => setIngestForm((current) => ({ ...current, stashPathPrefix: event.target.value }))}
                  placeholder="/library"
                />
              </label>
            </>
          ) : null}

          {ingestForm.mode === "FILE_TRANSFER" ? (
            <>
              <label className="settings-field">
                <span>搬运动作</span>
                <select
                  value={ingestForm.transferAction}
                  onChange={(event) => setIngestForm((current) => ({ ...current, transferAction: event.target.value }))}
                >
                  <option value="">请选择</option>
                  <option value="COPY">复制</option>
                  <option value="MOVE">移动</option>
                </select>
              </label>
              <label className="settings-field">
                <span>搬运目标目录</span>
                <input
                  value={ingestForm.transferTargetPath}
                  onChange={(event) => setIngestForm((current) => ({ ...current, transferTargetPath: event.target.value }))}
                  placeholder="/stash-import"
                />
              </label>
            </>
          ) : null}

          {ingestForm.mode === "LIBRARY_SCAN" ? (
            <label className="settings-field">
              <span>Library Path</span>
              <input
                value={ingestForm.libraryPath}
                onChange={(event) => setIngestForm((current) => ({ ...current, libraryPath: event.target.value }))}
                placeholder="/data/library"
              />
            </label>
          ) : null}

          <div className="settings-actions">
            <button type="submit" disabled={updatingIngest}>保存入库策略</button>
          </div>
        </form>
      </article>
    );
  }

  if (settingsTab === "自动化") {
    const stashBoxes = runtimeStatus.subscription.stashBoxes ?? [];
    const loaded = runtimeStatus.subscription.stashBoxesLoaded;
    const loadError = runtimeStatus.subscription.stashBoxesLoadError;
    const order = subscriptionForm.stashBoxEndpoints;
    const byEndpoint = new Map(stashBoxes.map((box) => [box.endpoint, box]));
    const display: { endpoint: string; box: typeof stashBoxes[number] | null }[] = [];
    const used = new Set<string>();

    order.forEach((endpoint) => {
      display.push({ endpoint, box: byEndpoint.get(endpoint) ?? null });
      used.add(endpoint);
    });
    stashBoxes.forEach((box) => {
      if (used.has(box.endpoint)) return;
      display.push({ endpoint: box.endpoint, box });
    });

    const reorder = (next: string[]) => {
      setSubscriptionForm((current) => ({ ...current, stashBoxEndpoints: next }));
    };
    const move = (from: number, to: number) => {
      if (from === to || from < 0 || to < 0 || from >= display.length || to >= display.length) return;
      const next = display.map((item) => item.endpoint);
      const [moved] = next.splice(from, 1);
      next.splice(to, 0, moved);
      reorder(next);
    };

    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>自动化</h3>
        </div>

        <form className="settings-form" onSubmit={(event) => void saveAutomationSettings(event)}>
          <label className="settings-field">
            <span>任务进度同步间隔（秒）</span>
            <input
              value={automationForm.taskProgressSyncIntervalSeconds}
              onChange={(event) => setAutomationForm((current) => ({ ...current, taskProgressSyncIntervalSeconds: event.target.value }))}
              placeholder="60"
            />
          </label>
          <label className="settings-field">
            <span>订阅轮询间隔（秒）</span>
            <input
              value={automationForm.subscriptionPollIntervalSeconds}
              onChange={(event) => setAutomationForm((current) => ({ ...current, subscriptionPollIntervalSeconds: event.target.value }))}
              placeholder="3600"
            />
          </label>
          <div className="settings-meta">
            <span>任务同步: {runtimeStatus.automation.taskProgressSyncEnabled ? "已启用" : "未启用"}</span>
            <span>订阅轮询: {runtimeStatus.automation.subscriptionPollEnabled ? "已启用" : "未启用"}</span>
          </div>
          <div className="settings-actions">
            <button type="submit" disabled={updatingAutomation}>保存自动化设置</button>
          </div>
        </form>

        <form className="settings-form" onSubmit={(event) => void saveSubscriptionSettings(event)}>
          <section className="stashbox-source">
            <header className="stashbox-source__head">
              <div>
                <h4>Stash-Box 数据源优先级</h4>
                <p className="stashbox-source__sub">
                  在 Stash 中配置的 Stash-Box 会出现在这里。所有端点都会参与订阅查询，拖动卡片即可调整优先级。
                </p>
              </div>
              <div className="stashbox-source__stats">
                <button
                  type="button"
                  className="ghost-button stashbox-source__refresh"
                  disabled={refreshingStashBoxes}
                  onClick={() => void refreshSubscriptionStashBoxes()}
                >
                  <FontAwesomeIcon icon={faRotate} className={refreshingStashBoxes ? "is-spinning" : undefined} />
                  <span>{refreshingStashBoxes ? "刷新中..." : "刷新"}</span>
                </button>
                <button type="submit" disabled={updatingSubscription}>保存优先级</button>
              </div>
            </header>

            {!loaded ? (
              <div className="stashbox-source__empty stashbox-source__empty--loading">
                <div className="stashbox-source__spinner" aria-hidden="true" />
                <div>
                  <strong>正在从 Stash 拉取 Stash-Box 端点…</strong>
                  <p>这一过程由后端在启动时自动完成，请稍候。</p>
                </div>
              </div>
            ) : display.length === 0 ? (
              <div className="stashbox-source__empty stashbox-source__empty--danger">
                <div className="stashbox-source__icon" aria-hidden="true">!</div>
                <div>
                  <strong>Stash 中尚未配置任何 Stash-Box</strong>
                  <p>请先在 Stash 中添加至少一个端点，再回到这里刷新列表。</p>
                  {loadError ? <p className="stashbox-source__error">拉取失败：{loadError}</p> : null}
                </div>
              </div>
            ) : (
              <ul className="stashbox-source__list">
                {display.map((item, index) => {
                  const classes = ["stashbox-card"];
                  if (draggedIndex === index) classes.push("is-dragging");
                  if (dragOverIndex === index) {
                    classes.push(dragOverIndex < (draggedIndex ?? -1) ? "is-drop-top" : "is-drop-bottom");
                  }

                  return (
                    <li
                      key={item.endpoint}
                      className={classes.join(" ")}
                      draggable
                      onDragStart={(event) => {
                        event.dataTransfer.effectAllowed = "move";
                        event.dataTransfer.setData("text/plain", String(index));
                        setDraggedIndex(index);
                      }}
                      onDragOver={(event) => {
                        event.preventDefault();
                        event.dataTransfer.dropEffect = "move";
                        setDragOverIndex(index);
                      }}
                      onDragLeave={() => {
                        if (dragOverIndex === index) setDragOverIndex(null);
                      }}
                      onDrop={(event) => {
                        event.preventDefault();
                        const from = Number.parseInt(event.dataTransfer.getData("text/plain"), 10);
                        move(Number.isNaN(from) ? draggedIndex ?? -1 : from, index);
                        setDraggedIndex(null);
                        setDragOverIndex(null);
                      }}
                      onDragEnd={() => {
                        setDraggedIndex(null);
                        setDragOverIndex(null);
                      }}
                    >
                      <span className="stashbox-card__handle" aria-hidden="true" title="拖动以重新排序">
                        <FontAwesomeIcon icon={faGripVertical} />
                      </span>
                      <span className="stashbox-card__body">
                        <span className="stashbox-card__title-row">
                          <strong className="stashbox-card__name">{item.box?.name || item.endpoint}</strong>
                          <span
                            className={`stashbox-card__chip ${
                              item.box?.apiKeyConfigured ? "stashbox-card__chip--ok" : "stashbox-card__chip--warn"
                            }`}
                          >
                            {item.box?.apiKeyConfigured ? "API key 已配置" : "未配置 API key"}
                          </span>
                        </span>
                        <code className="stashbox-card__endpoint">{item.endpoint}</code>
                      </span>
                      <span className="stashbox-card__move">
                        <button
                          type="button"
                          className="ghost-button ghost-button--icon"
                          disabled={index === 0}
                          onClick={() => move(index, index - 1)}
                          aria-label="上移"
                        >
                          <FontAwesomeIcon icon={faArrowUp} />
                        </button>
                        <button
                          type="button"
                          className="ghost-button ghost-button--icon"
                          disabled={index === display.length - 1}
                          onClick={() => move(index, index + 1)}
                          aria-label="下移"
                        >
                          <FontAwesomeIcon icon={faArrowDown} />
                        </button>
                      </span>
                    </li>
                  );
                })}
              </ul>
            )}
          </section>
        </form>
      </article>
    );
  }

  if (settingsTab === "日志") {
    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>日志</h3>
        </div>
        <div className="toolbar-inline toolbar-inline--logs">
          <select value={logsLevel} onChange={(event) => setLogsLevel(event.target.value as LogLevel)}>
            {LOG_LEVEL_OPTIONS.map((level) => (
              <option key={level} value={level}>
                {level}
              </option>
            ))}
          </select>
          <button type="button" className="ghost-button" onClick={() => void refreshLogs({ requestPolicy: "network-only" })}>
            <FontAwesomeIcon icon={faRotate} /> {fetchingLogs ? "刷新中..." : "刷新日志"}
          </button>
          <button type="button" className="ghost-button" onClick={() => void handleCopyLogs()}>
            复制当前列表
          </button>
          <button type="button" className="ghost-button" disabled={downloadingLogFile} onClick={() => void handleDownloadCurrentLogFile()}>
            {downloadingLogFile ? "下载中..." : "下载当前日志"}
          </button>
        </div>
        <div className="settings-meta">
          <span>级别过滤: {logsLevel}</span>
          <span>已加载: {logs.length}</span>
          <span>状态: {fetchingLogs ? "同步中" : "已就绪"}</span>
          <span>来源: 当前日志文件</span>
        </div>
        {logsError ? <p className="settings-feedback tone-danger">{describeQueryError(logsError)}</p> : null}
        {!logs.length && !fetchingLogs ? (
          <article className="empty-card empty-card--wide">
            <h3>暂无日志</h3>
            <p>当前过滤条件下没有最近日志记录。</p>
          </article>
        ) : (
          <div className="log-stream" role="log" aria-live="polite">
            {logs.map((entry, index) => (
              <div
                key={`${entry.time}-${index}`}
                className={`log-line ${
                  entry.level === LogLevel.Error
                    ? "log-line--error"
                    : entry.level === LogLevel.Warning
                      ? "log-line--warn"
                      : entry.level === LogLevel.Debug
                        ? "log-line--debug"
                        : "log-line--info"
                }`}
              >
                <span className="log-line__time">{formatDateTime(entry.time)}</span>
                <span className="log-line__level">[{entry.level}]</span>
                <span className="log-line__message">{entry.message}</span>
              </div>
            ))}
          </div>
        )}
      </article>
    );
  }

  return (
    <article className="drawer-card">
      <div className="drawer-card__head">
        <h3>关于</h3>
      </div>
      <dl className="settings-grid">
        <div>
          <dt>版本</dt>
          <dd>{appVersion || "dev"}</dd>
        </div>
      </dl>
    </article>
  );
}
