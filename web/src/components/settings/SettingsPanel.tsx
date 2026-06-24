import { FormEvent, useEffect, useState } from "react";
import { useMutation, useQuery } from "urql";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faArrowDown,
  faArrowUp,
  faCircleInfo,
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

interface FieldLabelProps {
  text: string;
  info?: string;
}

function FieldLabel({ text, info }: FieldLabelProps) {
  return (
    <span className="settings-field__label">
      <span>{text}</span>
      {info ? (
        <span className="settings-info" tabIndex={0} aria-label={info}>
          <FontAwesomeIcon icon={faCircleInfo} aria-hidden="true" />
          <span className="settings-info__tooltip" role="tooltip">
            {info}
          </span>
        </span>
      ) : null}
    </span>
  );
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
      deliveryMode: runtimeSettings.ingest.deliveryMode || "PATH_MAP",
      stashLibraryPath: runtimeSettings.ingest.stashLibraryPath || "",
      transfer: {
        action: runtimeSettings.ingest.transfer.action || "",
        mojiSourceRoot: runtimeSettings.ingest.transfer.mojiSourceRoot || "",
        mojiTargetRoot: runtimeSettings.ingest.transfer.mojiTargetRoot || ""
      }
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
        deliveryMode: ingestForm.deliveryMode.trim(),
        stashLibraryPath: ingestForm.stashLibraryPath.trim(),
        transfer: {
          action: ingestForm.transfer.action.trim(),
          mojiSourceRoot: ingestForm.transfer.mojiSourceRoot.trim(),
          mojiTargetRoot: ingestForm.transfer.mojiTargetRoot.trim()
        }
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
    const stashStatus = serviceStatus(
      runtimeStatus.stash.configured,
      runtimeStatus.stash.ready,
      runtimeStatus.stashStats?.lastError ?? null,
      runtimeStatus.stashStats?.okAt ?? null
    );
    const jackettStatus = serviceStatus(
      runtimeStatus.jackett.configured,
      runtimeStatus.jackett.ready,
      runtimeStatus.jackettStats?.lastError ?? null,
      runtimeStatus.jackettStats?.okAt ?? null
    );
    const qbittorrentStatus = serviceStatus(
      runtimeStatus.qbittorrent.configured,
      runtimeStatus.qbittorrent.ready,
      runtimeStatus.qbittorrentStats?.lastError ?? null,
      runtimeStatus.qbittorrentStats?.okAt ?? null
    );

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
    const stashLibraries = runtimeStatus.stashLibraries ?? [];
    const stashLibrariesLoadError = runtimeStatus.stashLibrariesLoadError ?? null;
    const deliveryModeInfo = ingestForm.deliveryMode === "PATH_MAP"
      ? "适用于 qBittorrent 和 Stash 共享同一份底层存储，但容器挂载路径可以不同。Moji 不搬运文件，只会根据任务保存目录和内容路径自动换算出 Stash 侧扫描路径。"
      : "适用于下载区和媒体库分离的场景。Moji 需要同时访问下载区和媒体库目录，先执行文件交付，再把相对路径换算到所选 Stash 媒体库下触发扫描。";
    const stashLibraryInfo = "这里选择的是 Stash 已配置的媒体库根路径。无论使用 PATH_MAP 还是 TRANSFER，Moji 最终都会把相对路径拼到这个库路径下，并通知 Stash 扫描。";
    const transferActionInfo = "COPY 会保留下载区原文件，MOVE 会把文件迁移进媒体库，SYMLINK 会在媒体库里创建指向源文件的符号链接。目标已存在同名文件或链接时会直接失败。";
    const mojiSourceRootInfo = "填写 Moji 自己能访问到的下载区根目录。源文件必须位于这个目录之下，Moji 会以它为基准保留相对目录层级。";
    const mojiTargetRootInfo = "填写 Moji 自己能写入的媒体库根目录。Moji 会把源文件相对下载区的层级结构复制、移动或链接到这里，再映射到上面选择的 Stash 媒体库路径。";

    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>入库</h3>
        </div>
        <form className="settings-form" onSubmit={(event) => void saveIngestSettings(event)}>
          <label className="settings-field">
            <FieldLabel text="入库方式" info={deliveryModeInfo} />
            <select
              value={ingestForm.deliveryMode}
              onChange={(event) => setIngestForm((current) => ({
                ...current,
                deliveryMode: event.target.value
              }))}
            >
              <option value="PATH_MAP">路径映射</option>
              <option value="TRANSFER">文件交付</option>
            </select>
          </label>
          <label className="settings-field">
            <FieldLabel text="目标媒体库" info={stashLibraryInfo} />
            <select
              value={ingestForm.stashLibraryPath}
              onChange={(event) => setIngestForm((current) => ({
                ...current,
                stashLibraryPath: event.target.value
              }))}
            >
              <option value="">请选择 Stash 媒体库</option>
              {stashLibraries.map((library) => (
                <option key={library.path} value={library.path}>
                  {library.path}
                </option>
              ))}
            </select>
          </label>
          {stashLibrariesLoadError ? (
            <p className="service-card__error" role="alert">
              {stashLibrariesLoadError}
            </p>
          ) : null}

          {ingestForm.deliveryMode === "TRANSFER" ? (
            <>
              <label className="settings-field">
                <FieldLabel text="交付动作" info={transferActionInfo} />
                <select
                  value={ingestForm.transfer.action}
                  onChange={(event) => setIngestForm((current) => ({
                    ...current,
                    transfer: { ...current.transfer, action: event.target.value }
                  }))}
                >
                  <option value="COPY">复制</option>
                  <option value="MOVE">移动</option>
                  <option value="SYMLINK">符号链接</option>
                </select>
              </label>
              <label className="settings-field">
                <FieldLabel text="Moji 下载区目录" info={mojiSourceRootInfo} />
                <input
                  value={ingestForm.transfer.mojiSourceRoot}
                  onChange={(event) => setIngestForm((current) => ({
                    ...current,
                    transfer: { ...current.transfer, mojiSourceRoot: event.target.value }
                  }))}
                  placeholder="/downloads"
                />
              </label>
              <label className="settings-field">
                <FieldLabel text="Moji 媒体库目录" info={mojiTargetRootInfo} />
                <input
                  value={ingestForm.transfer.mojiTargetRoot}
                  onChange={(event) => setIngestForm((current) => ({
                    ...current,
                    transfer: { ...current.transfer, mojiTargetRoot: event.target.value }
                  }))}
                  placeholder="/mnt/media-library"
                />
              </label>
            </>
          ) : null}

          <div className="settings-actions">
            <button type="submit" disabled={updatingIngest}>保存入库设置</button>
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
