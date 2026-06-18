import { FormEvent, useEffect, useState } from "react";
import { useMutation, useQuery } from "urql";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faRotate } from "@fortawesome/free-solid-svg-icons";
import {
  LogLevel,
  LogsDocumentDocument,
  RefreshSubscriptionStashBoxesDocumentDocument,
  UpdateSubscriptionSettingsDocumentDocument,
  UpdateJackettSettingsDocumentDocument,
  UpdateLoggingSettingsDocumentDocument,
  UpdateQBittorrentSettingsDocumentDocument,
  UpdateStashSettingsDocumentDocument,
  type DashboardDocumentQuery,
  type LogsDocumentQuery,
  type LogsDocumentQueryVariables,
  type RefreshSubscriptionStashBoxesDocumentMutation,
  type UpdateSubscriptionSettingsDocumentMutation,
  type UpdateSubscriptionSettingsDocumentMutationVariables,
  type UpdateJackettSettingsDocumentMutation,
  type UpdateJackettSettingsDocumentMutationVariables,
  type UpdateLoggingSettingsDocumentMutation,
  type UpdateLoggingSettingsDocumentMutationVariables,
  type UpdateQBittorrentSettingsDocumentMutation,
  type UpdateQBittorrentSettingsDocumentMutationVariables,
  type UpdateStashSettingsDocumentMutation,
  type UpdateStashSettingsDocumentMutationVariables
} from "../../graphql/generated/graphql";
import type { SettingsTab, ToastTone } from "../../types";
import {
  EMPTY_STASH_FORM, EMPTY_JACKETT_FORM, EMPTY_QBITTORRENT_FORM,
  EMPTY_SUBSCRIPTION_FORM, EMPTY_LOGGING_FORM,
  LOG_LEVEL_OPTIONS
} from "../../constants";
import { boolState, serviceStatus, taskSyncStatus } from "../../utils";
import { describeQueryError } from "../../services/queryError";
import { formatDateTime, formatLogEntries } from "../../utils";

type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;

interface SettingsPanelProps {
  settingsTab: SettingsTab;
  onSettingsTabChange: (tab: SettingsTab) => void;
  runtimeSettings: RuntimeSettings | null;
  drawer: string | null;
  renderedDrawer: string | null;
  pushToast: (tone: ToastTone, message: string) => void;
  refreshDashboard: (opts?: Record<string, unknown>) => unknown;
}

export function SettingsPanel({
  settingsTab,
  onSettingsTabChange,
  runtimeSettings,
  drawer,
  renderedDrawer,
  pushToast,
  refreshDashboard
}: SettingsPanelProps) {
  const [logsLevel, setLogsLevel] = useState<LogLevel>(LogLevel.Info);
  const [downloadingLogFile, setDownloadingLogFile] = useState(false);

  const [stashForm, setStashForm] = useState(EMPTY_STASH_FORM);
  const [jackettForm, setJackettForm] = useState(EMPTY_JACKETT_FORM);
  const [qbittorrentForm, setQBittorrentForm] = useState(EMPTY_QBITTORRENT_FORM);
  const [subscriptionForm, setSubscriptionForm] = useState(EMPTY_SUBSCRIPTION_FORM);
  const [loggingForm, setLoggingForm] = useState(EMPTY_LOGGING_FORM);

  // ── Queries ──────────────────────────────────────────────────────

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

  // ── Mutations ────────────────────────────────────────────────────

  const [{ fetching: updatingStash }, updateStashSettings] = useMutation<
    UpdateStashSettingsDocumentMutation,
    UpdateStashSettingsDocumentMutationVariables
  >(UpdateStashSettingsDocumentDocument);

  const [{ fetching: updatingJackett }, updateJackettSettings] = useMutation<
    UpdateJackettSettingsDocumentMutation,
    UpdateJackettSettingsDocumentMutationVariables
  >(UpdateJackettSettingsDocumentDocument);

  const [{ fetching: updatingQBittorrent }, updateQBittorrentSettings] = useMutation<
    UpdateQBittorrentSettingsDocumentMutation,
    UpdateQBittorrentSettingsDocumentMutationVariables
  >(UpdateQBittorrentSettingsDocumentDocument);

  const [{ fetching: updatingSubscription }, updateSubscriptionSettings] = useMutation<
    UpdateSubscriptionSettingsDocumentMutation,
    UpdateSubscriptionSettingsDocumentMutationVariables
  >(UpdateSubscriptionSettingsDocumentDocument);

  const [{ fetching: refreshingStashBoxes }, refreshStashBoxesMutation] = useMutation<
    RefreshSubscriptionStashBoxesDocumentMutation
  >(RefreshSubscriptionStashBoxesDocumentDocument);

  const [{ fetching: updatingLogging }, updateLoggingSettings] = useMutation<
    UpdateLoggingSettingsDocumentMutation,
    UpdateLoggingSettingsDocumentMutationVariables
  >(UpdateLoggingSettingsDocumentDocument);

  // ── Sync forms from runtime settings ─────────────────────────────

  useEffect(() => {
    if (!runtimeSettings) return;

    setStashForm({
      url: runtimeSettings.stash.url || "",
      apiKey: "",
      libraryPath: runtimeSettings.stash.libraryPath || ""
    });
    setJackettForm({
      url: runtimeSettings.jackett.url || "",
      apiKey: ""
    });
    setQBittorrentForm({
      url: runtimeSettings.qbittorrent.url || "",
      username: runtimeSettings.qbittorrent.username || "",
      password: "",
      defaultSavePath: runtimeSettings.qbittorrent.defaultSavePath || "",
      category: runtimeSettings.qbittorrent.category || "",
      tags: runtimeSettings.qbittorrent.tags || ""
    });
    setSubscriptionForm({
      store: runtimeSettings.subscription.store || "sqlite",
      dbPath: runtimeSettings.subscription.dbPath || "",
      pollIntervalSeconds: String(runtimeSettings.subscription.pollIntervalSeconds || 3600),
      selectedStashBoxEndpoints: [...(runtimeSettings.subscription.selectedStashBoxEndpoints ?? [])]
    });
    setLoggingForm({
      level: runtimeSettings.logging.level || "info",
      filePath: runtimeSettings.logging.filePath || "",
      maxEntries: String(runtimeSettings.logging.maxEntries || 500),
      maxFileSizeBytes: String(runtimeSettings.logging.maxFileSizeBytes || 10 * 1024 * 1024),
      maxFileBackups: String(runtimeSettings.logging.maxFileBackups || 5)
    });
  }, [runtimeSettings]);

  // ── Settings status ──────────────────────────────────────────────

  const settingsStatus = (() => {
    if (!runtimeSettings) return { label: "加载中", tone: "tone-neutral" as const };
    if (settingsTab === "Stash") {
      return serviceStatus(runtimeSettings.stash.configured, runtimeSettings.stash.enabled);
    }
    if (settingsTab === "索引器") {
      return serviceStatus(runtimeSettings.jackett.configured, runtimeSettings.jackett.enabled);
    }
    if (settingsTab === "下载器") {
      return serviceStatus(runtimeSettings.qbittorrent.configured, runtimeSettings.qbittorrent.enabled);
    }
    if (settingsTab === "任务") {
      return {
        label: taskSyncStatus(runtimeSettings.tasks),
        tone: runtimeSettings.tasks.progressSyncEnabled ? "tone-success" as const : "tone-neutral" as const
      };
    }
    if (settingsTab === "订阅") {
      return {
        label: runtimeSettings.subscription.pollEnabled ? "已启用" : "未启用",
        tone: runtimeSettings.subscription.pollEnabled ? "tone-success" as const : "tone-neutral" as const
      };
    }
    if (settingsTab === "系统") {
      return { label: "已接线", tone: "tone-info" as const };
    }
    return { label: "规划中", tone: "tone-neutral" as const };
  })();

  // ── Auto-refresh logs ────────────────────────────────────────────

  useEffect(() => {
    const logsTabActive = settingsTab === "日志" && (drawer === "settings" || renderedDrawer === "settings");
    if (!logsTabActive) {
      return;
    }

    const timer = window.setInterval(() => {
      void refreshLogs({ requestPolicy: "network-only" });
    }, 5000);

    return () => window.clearInterval(timer);
  }, [drawer, renderedDrawer, refreshLogs, settingsTab]);

  // ── Save handlers ────────────────────────────────────────────────

  const saveStashSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = await updateStashSettings({
      input: {
        url: stashForm.url.trim(),
        apiKey: stashForm.apiKey.trim() || null,
        libraryPath: stashForm.libraryPath.trim()
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    setStashForm((current) => ({ ...current, apiKey: "" }));
    pushToast("tone-success", "Stash 设置已保存，配置文件与运行时快照已刷新。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveJackettSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = await updateJackettSettings({
      input: {
        url: jackettForm.url.trim(),
        apiKey: jackettForm.apiKey.trim() || null
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    setJackettForm((current) => ({ ...current, apiKey: "" }));
    pushToast("tone-success", "索引器设置已保存，后端配置已同步。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveQBittorrentSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = await updateQBittorrentSettings({
      input: {
        url: qbittorrentForm.url.trim(),
        username: qbittorrentForm.username.trim(),
        password: qbittorrentForm.password.trim() || null,
        defaultSavePath: qbittorrentForm.defaultSavePath.trim(),
        category: qbittorrentForm.category.trim(),
        tags: qbittorrentForm.tags.trim()
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    setQBittorrentForm((current) => ({ ...current, password: "" }));
    pushToast("tone-success", "下载器设置已保存，新的默认值已同步到后端。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveSubscriptionSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const pollIntervalSeconds = Number.parseInt(subscriptionForm.pollIntervalSeconds.trim(), 10);
    const normalizedPollIntervalSeconds = Number.isNaN(pollIntervalSeconds) ? 0 : pollIntervalSeconds;
    const result = await updateSubscriptionSettings({
      input: {
        store: subscriptionForm.store.trim() || "sqlite",
        dbPath: subscriptionForm.dbPath.trim(),
        pollIntervalSeconds: normalizedPollIntervalSeconds,
        selectedStashBoxEndpoints: subscriptionForm.selectedStashBoxEndpoints
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "订阅设置已保存，轮询与 Stash-Box 选择已同步到后端。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const refreshSubscriptionStashBoxes = async () => {
    const result = await refreshStashBoxesMutation({});
    if (result.error) {
      pushToast("tone-danger", `刷新 Stash-Box 失败：${describeQueryError(result.error)}`);
      return;
    }
    const subscription = result.data?.refreshSubscriptionStashBoxes?.subscription;
    if (subscription) {
      const count = subscription.stashBoxes?.length ?? 0;
      pushToast(
        subscription.stashBoxesLoaded
          ? "tone-success"
          : "tone-danger",
        subscription.stashBoxesLoaded
          ? `Stash-Box 已刷新，共 ${count} 个端点。`
          : `刷新失败：${subscription.stashBoxesLoadError ?? "未知错误"}`
      );
    } else {
      pushToast("tone-success", "Stash-Box 已刷新。");
    }
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveLoggingSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const maxEntries = Number.parseInt(loggingForm.maxEntries.trim(), 10);
    const maxFileSizeBytes = Number.parseInt(loggingForm.maxFileSizeBytes.trim(), 10);
    const maxFileBackups = Number.parseInt(loggingForm.maxFileBackups.trim(), 10);
    const result = await updateLoggingSettings({
      input: {
        level: loggingForm.level.trim() || "info",
        filePath: loggingForm.filePath.trim(),
        maxEntries: Number.isNaN(maxEntries) ? 500 : maxEntries,
        maxFileSizeBytes: Number.isNaN(maxFileSizeBytes) ? 10 * 1024 * 1024 : maxFileSizeBytes,
        maxFileBackups: Number.isNaN(maxFileBackups) ? 5 : maxFileBackups
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "日志设置已保存，并已即时重载到当前进程。");
    await refreshDashboard({ requestPolicy: "network-only" });
    await refreshLogs({ requestPolicy: "network-only" });
  };

  // ── Log actions ──────────────────────────────────────────────────

  const handleCopyLogs = async () => {
    await navigator.clipboard.writeText(formatLogEntries(logs));
  };

  const handleDownloadCurrentLogFile = async () => {
    if (!runtimeSettings) return;
    setDownloadingLogFile(true);
    try {
      const filePath = runtimeSettings.logging.filePath;
      const response = await fetch(`/api/logs/file?path=${encodeURIComponent(filePath)}`);
      if (!response.ok) throw new Error(`下载失败：HTTP ${response.status}`);
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = filePath.split("/").pop() || "moji.log";
      link.click();
      URL.revokeObjectURL(url);
    } catch (error) {
      pushToast("tone-danger", error instanceof Error ? error.message : "下载当前日志文件失败。");
    } finally {
      setDownloadingLogFile(false);
    }
  };

  // ── Render ───────────────────────────────────────────────────────

  if (!runtimeSettings) {
    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>{settingsTab}</h3>
          <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
        </div>
        <dl className="settings-grid">
          <div>
            <dt>当前状态</dt>
            <dd>等待后端返回配置状态</dd>
          </div>
          <div>
            <dt>说明</dt>
            <dd>设置面板会在 dashboard 查询完成后显示实时状态。</dd>
          </div>
        </dl>
      </article>
    );
  }

  if (settingsTab === "Stash") {
    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>{settingsTab}</h3>
          <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
        </div>
        <form className="settings-form" onSubmit={(event) => void saveStashSettings(event)}>
          <label className="settings-field">
            <span>Stash URL</span>
            <input
              value={stashForm.url}
              onChange={(event) => setStashForm((current) => ({ ...current, url: event.target.value }))}
              placeholder="http://localhost:9999"
            />
          </label>
          <label className="settings-field">
            <span>Library path</span>
            <input
              value={stashForm.libraryPath}
              onChange={(event) => setStashForm((current) => ({ ...current, libraryPath: event.target.value }))}
              placeholder="/data/library"
            />
          </label>
          <label className="settings-field">
            <span>API key</span>
            <input
              type="password"
              value={stashForm.apiKey}
              onChange={(event) => setStashForm((current) => ({ ...current, apiKey: event.target.value }))}
              placeholder={runtimeSettings.stash.apiKeyConfigured ? "留空则保留现有 API key" : "输入新的 API key"}
            />
          </label>
          <div className="settings-meta">
            <span>当前 API key: {boolState(runtimeSettings.stash.apiKeyConfigured)}</span>
          </div>
          <div className="settings-actions">
            <button type="submit" disabled={updatingStash}>保存 Stash 设置</button>
          </div>
        </form>
      </article>
    );
  }

  if (settingsTab === "索引器") {
    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>{settingsTab}</h3>
          <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
        </div>
        <form className="settings-form" onSubmit={(event) => void saveJackettSettings(event)}>
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
            <input
              type="password"
              value={jackettForm.apiKey}
              onChange={(event) => setJackettForm((current) => ({ ...current, apiKey: event.target.value }))}
              placeholder={runtimeSettings.jackett.apiKeyConfigured ? "留空则保留现有 API key" : "输入新的 API key"}
            />
          </label>
          <div className="settings-meta">
            <span>当前 API key: {boolState(runtimeSettings.jackett.apiKeyConfigured)}</span>
            <span>后续可继续扩展 tracker 分组与默认搜索策略。</span>
          </div>
          <div className="settings-actions">
            <button type="submit" disabled={updatingJackett}>保存索引器设置</button>
          </div>
        </form>
      </article>
    );
  }

  if (settingsTab === "下载器") {
    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>{settingsTab}</h3>
          <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
        </div>
        <form className="settings-form" onSubmit={(event) => void saveQBittorrentSettings(event)}>
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
            <input
              type="password"
              value={qbittorrentForm.password}
              onChange={(event) => setQBittorrentForm((current) => ({ ...current, password: event.target.value }))}
              placeholder={runtimeSettings.qbittorrent.passwordConfigured ? "留空则保留现有密码" : "输入新的登录密码"}
            />
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
          <div className="settings-meta">
            <span>当前密码: {boolState(runtimeSettings.qbittorrent.passwordConfigured)}</span>
            <span>用户名会直接回显，密码仍只支持覆盖更新。</span>
          </div>
          <div className="settings-actions">
            <button type="submit" disabled={updatingQBittorrent}>保存下载器设置</button>
          </div>
        </form>
      </article>
    );
  }

  if (settingsTab === "任务") {
    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>{settingsTab}</h3>
          <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
        </div>
        <dl className="settings-grid">
          <div>
            <dt>存储类型</dt>
            <dd>{runtimeSettings.tasks.store || "sqlite"}</dd>
          </div>
          <div>
            <dt>数据库路径</dt>
            <dd>{runtimeSettings.tasks.dbPath || "moji.db"}</dd>
          </div>
          <div>
            <dt>同步间隔</dt>
            <dd>{runtimeSettings.tasks.progressSyncIntervalSeconds} 秒</dd>
          </div>
          <div>
            <dt>进度同步</dt>
            <dd>{taskSyncStatus(runtimeSettings.tasks)}</dd>
          </div>
          <div>
            <dt>说明</dt>
            <dd>当前同步开关由任务配置和下载链路是否启用共同决定。</dd>
          </div>
        </dl>
      </article>
    );
  }

  if (settingsTab === "订阅") {
    const stashBoxes = runtimeSettings.subscription.stashBoxes ?? [];
    const selected = subscriptionForm.selectedStashBoxEndpoints;
    const loaded = runtimeSettings.subscription.stashBoxesLoaded;
    const loadError = runtimeSettings.subscription.stashBoxesLoadError;

    const toggleStashBox = (endpoint: string, checked: boolean) => {
      setSubscriptionForm((current) => {
        const set = new Set(current.selectedStashBoxEndpoints);
        if (checked) {
          set.add(endpoint);
        } else {
          set.delete(endpoint);
        }
        return { ...current, selectedStashBoxEndpoints: Array.from(set) };
      });
    };
    const setSelection = (endpoints: string[]) => {
      setSubscriptionForm((current) => ({ ...current, selectedStashBoxEndpoints: endpoints }));
    };
    const allEndpoints = stashBoxes.map((box) => box.endpoint);
    const allSelected = allEndpoints.length > 0 && selected.length === allEndpoints.length;
    const toggleAll = () => {
      setSelection(allSelected ? [] : allEndpoints);
    };

    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>{settingsTab}</h3>
          <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
        </div>
        <form className="settings-form" onSubmit={(event) => void saveSubscriptionSettings(event)}>
          <label className="settings-field">
            <span>存储类型</span>
            <input
              value={subscriptionForm.store}
              onChange={(event) => setSubscriptionForm((current) => ({ ...current, store: event.target.value }))}
              placeholder="sqlite"
            />
          </label>
          <label className="settings-field">
            <span>数据库路径</span>
            <input
              value={subscriptionForm.dbPath}
              onChange={(event) => setSubscriptionForm((current) => ({ ...current, dbPath: event.target.value }))}
              placeholder="moji.db"
            />
          </label>
          <label className="settings-field">
            <span>轮询间隔（秒）</span>
            <input
              value={subscriptionForm.pollIntervalSeconds}
              onChange={(event) => setSubscriptionForm((current) => ({ ...current, pollIntervalSeconds: event.target.value }))}
              placeholder="3600"
              inputMode="numeric"
            />
          </label>

          <section className="stashbox-source">
            <header className="stashbox-source__head">
              <div>
                <h4>Stash-Box 数据源</h4>
                <p className="stashbox-source__sub">
                  在 Stash → 设置 → 元数据提供者 → Stash-box 中配置后会出现在这里。
                </p>
              </div>
              <div className="stashbox-source__stats">
                {loaded && stashBoxes.length > 0 ? (
                  <>
                    <span className="stashbox-source__count">{selected.length}<small> / {stashBoxes.length}</small></span>
                    <button
                      type="button"
                      className="ghost-button"
                      onClick={toggleAll}
                      disabled={allEndpoints.length === 0}
                    >
                      {allSelected ? "全部取消" : "全部选择"}
                    </button>
                  </>
                ) : null}
                <button
                  type="button"
                  className="ghost-button stashbox-source__refresh"
                  onClick={() => void refreshSubscriptionStashBoxes()}
                  disabled={refreshingStashBoxes}
                  title="从 Stash 重新拉取 Stash-Box 配置"
                  aria-label="刷新 Stash-Box 列表"
                >
                  <FontAwesomeIcon
                    icon={faRotate}
                    className={refreshingStashBoxes ? "is-spinning" : undefined}
                    aria-hidden="true"
                  />
                  <span>{refreshingStashBoxes ? "刷新中..." : "刷新"}</span>
                </button>
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
            ) : stashBoxes.length === 0 ? (
              <div className="stashbox-source__empty stashbox-source__empty--danger">
                <div className="stashbox-source__icon" aria-hidden="true">!</div>
                <div>
                  <strong>Stash 中尚未配置任何 Stash-Box</strong>
                  <p>请在 Stash → Settings → Stash-Boxes 中至少添加一个端点后，刷新本页。</p>
                  {loadError ? <p className="stashbox-source__error">拉取失败：{loadError}</p> : null}
                </div>
              </div>
            ) : (
              <ul className="stashbox-source__list">
                {stashBoxes.map((box) => {
                  const checked = selected.includes(box.endpoint);
                  return (
                    <li key={box.endpoint}>
                      <label className={`stashbox-card${checked ? " is-selected" : ""}`}>
                        <input
                          type="checkbox"
                          className="stashbox-card__checkbox"
                          checked={checked}
                          onChange={(event) => toggleStashBox(box.endpoint, event.target.checked)}
                        />
                        <span className="stashbox-card__body">
                          <span className="stashbox-card__title-row">
                            <strong className="stashbox-card__name">{box.name || "(未命名)"}</strong>
                            <span
                              className={`stashbox-card__chip ${
                                box.apiKeyConfigured ? "stashbox-card__chip--ok" : "stashbox-card__chip--warn"
                              }`}
                            >
                              {box.apiKeyConfigured ? "API key 已配置" : "未配置 API key"}
                            </span>
                          </span>
                          <code className="stashbox-card__endpoint">{box.endpoint}</code>
                        </span>
                      </label>
                    </li>
                  );
                })}
              </ul>
            )}

            {loaded && stashBoxes.length > 0 ? (
              <footer className="stashbox-source__foot">
                {selected.length === 0
                  ? "未勾选任何端点，订阅将使用全部可用 Stash-Box。"
                  : `已选 ${selected.length} / ${stashBoxes.length} 个端点。`}
              </footer>
            ) : null}
          </section>

          <div className="settings-meta">
            <span>当前存储: {runtimeSettings.subscription.store || "sqlite"}</span>
            <span>轮询状态: {runtimeSettings.subscription.pollEnabled ? "已启用" : "未启用"}</span>
          </div>
          <div className="settings-actions">
            <button
              type="submit"
              disabled={updatingSubscription || !loaded}
            >
              保存订阅设置
            </button>
          </div>
        </form>
      </article>
    );
  }

  if (settingsTab === "系统") {
    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>{settingsTab}</h3>
          <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
        </div>
        <form className="settings-form" onSubmit={(event) => void saveLoggingSettings(event)}>
          <label className="settings-field">
            <span>日志级别</span>
            <input
              value={loggingForm.level}
              onChange={(event) => setLoggingForm((current) => ({ ...current, level: event.target.value }))}
              placeholder="info"
            />
          </label>
          <label className="settings-field">
            <span>日志文件路径</span>
            <input
              value={loggingForm.filePath}
              onChange={(event) => setLoggingForm((current) => ({ ...current, filePath: event.target.value }))}
              placeholder="moji.log"
            />
          </label>
          <label className="settings-field">
            <span>内存保留条数</span>
            <input
              value={loggingForm.maxEntries}
              onChange={(event) => setLoggingForm((current) => ({ ...current, maxEntries: event.target.value }))}
              inputMode="numeric"
              placeholder="500"
            />
          </label>
          <label className="settings-field">
            <span>单文件大小上限（字节）</span>
            <input
              value={loggingForm.maxFileSizeBytes}
              onChange={(event) => setLoggingForm((current) => ({ ...current, maxFileSizeBytes: event.target.value }))}
              inputMode="numeric"
              placeholder={String(10 * 1024 * 1024)}
            />
          </label>
          <label className="settings-field">
            <span>滚动备份份数</span>
            <input
              value={loggingForm.maxFileBackups}
              onChange={(event) => setLoggingForm((current) => ({ ...current, maxFileBackups: event.target.value }))}
              inputMode="numeric"
              placeholder="5"
            />
          </label>
          <div className="settings-meta">
            <span>版本: {runtimeSettings.system.appVersion || "dev"}</span>
            <span>当前日志文件: {runtimeSettings.logging.filePath}</span>
            <span>当前缓存: {runtimeSettings.logging.maxEntries} 条</span>
          </div>
          <div className="settings-actions">
            <button type="submit" disabled={updatingLogging}>保存系统设置</button>
          </div>
        </form>
      </article>
    );
  }

  if (settingsTab === "日志") {
    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>{settingsTab}</h3>
          <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
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
            {fetchingLogs ? "刷新中..." : "刷新日志"}
          </button>
          <button type="button" className="ghost-button" onClick={() => void handleCopyLogs()}>
            复制当前列表
          </button>
          <button type="button" className="ghost-button" onClick={() => void handleDownloadCurrentLogFile()} disabled={downloadingLogFile}>
            {downloadingLogFile ? "下载中..." : "下载当前日志"}
          </button>
        </div>
        <div className="settings-meta">
          <span>级别过滤: {logsLevel}</span>
          <span>已加载: {logs.length}</span>
          <span>状态: {fetchingLogs ? "同步中" : "已就绪"}</span>
          <span>文件: {runtimeSettings.logging.filePath}</span>
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
        <h3>{settingsTab}</h3>
        <span className={`status-chip ${settingsStatus.tone}`}>{settingsStatus.label}</span>
      </div>
      <dl className="settings-grid">
        <div>
          <dt>当前状态</dt>
          <dd>该分区尚未接入后端契约</dd>
        </div>
        <div>
          <dt>敏感值</dt>
          <dd>前端不展示明文</dd>
        </div>
        <div>
          <dt>接入方式</dt>
          <dd>后续会扩展为真实查询或操作面板</dd>
        </div>
        <div>
          <dt>说明</dt>
          <dd>
            {settingsTab === "安全性"
              ? "这里会放访问控制、CORS 和未来登录策略。"
              : settingsTab === "工具"
                  ? "这里会放重新同步、重新探测和修复动作。"
                  : settingsTab === "更新历史"
                    ? "这里会放版本记录和升级提示。"
                    : "这里会放项目定位、许可证和作者信息。"}
          </dd>
        </div>
      </dl>
    </article>
  );
}
