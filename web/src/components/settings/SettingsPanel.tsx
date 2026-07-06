import { FormEvent, useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { useMutation, useQuery } from "urql";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faArrowDown,
  faArrowUp,
  faChevronDown,
  faChevronUp,
  faCircleInfo,
  faEye,
  faEyeSlash,
  faGripVertical,
  faPenToSquare,
  faPlus,
  faRotate,
  faTimes,
  faTrash
} from "@fortawesome/free-solid-svg-icons";
import {
  JackettIndexersDocumentDocument,
  LogLevel,
  LogsDocumentDocument,
  RefreshSubscriptionStashBoxesDocumentDocument,
  TorrentFileMatchEffect,
  TitleMatchEffect,
  TitleMatchPatternMode,
  TorrentSelectionDirection,
  TorrentSelectionRuleType,
  UpdateAutomationSettingsDocumentDocument,
  UpdateIngestSettingsDocumentDocument,
  UpdateJackettSettingsDocumentDocument,
  UpdateQBittorrentSettingsDocumentDocument,
  UpdateStashSettingsDocumentDocument,
  UpdateSystemSettingsDocumentDocument,
  type DashboardDocumentQuery,
  type JackettIndexersDocumentQuery,
  type JackettIndexersDocumentQueryVariables,
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
  type UpdateSystemSettingsDocumentMutation,
  type UpdateSystemSettingsDocumentMutationVariables,
  TaskDeletePolicy
} from "../../graphql/generated/graphql";
import type { SettingsTab, ToastTone } from "../../types";
import {
  DEFAULT_TORRENT_FILE_INSPECTION_RULES,
  DEFAULT_TORRENT_SELECTION_RULES,
  EMPTY_AUTOMATION_FORM,
  EMPTY_INGEST_FORM,
  EMPTY_JACKETT_FORM,
  EMPTY_QBITTORRENT_FORM,
  EMPTY_STASH_FORM,
  EMPTY_SYSTEM_FORM,
  LOG_LEVEL_OPTIONS
} from "../../constants";
import { serviceStatus } from "../../utils";
import { describeQueryError } from "../../services/queryError";
import { formatDateTime, formatLogEntries } from "../../utils";
import { Menu } from "../common/Menu";

type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;
type RuntimeSettingsStatus = NonNullable<DashboardDocumentQuery["settingsStatus"]>;
type JackettIndexer = JackettIndexersDocumentQuery["jackettIndexers"][number];

const TORRENT_SELECTION_RULE_TYPE_OPTIONS: TorrentSelectionRuleType[] = [
  TorrentSelectionRuleType.IndexerPreference,
  TorrentSelectionRuleType.TitleMatch,
  TorrentSelectionRuleType.PublishDate,
  TorrentSelectionRuleType.TitleSimilarity,
  TorrentSelectionRuleType.Seeders,
  TorrentSelectionRuleType.Size,
  TorrentSelectionRuleType.TorrentSingleVideo,
  TorrentSelectionRuleType.TorrentFileNameMatch
];

const FAST_TORRENT_SELECTION_RULE_TYPE_OPTIONS = TORRENT_SELECTION_RULE_TYPE_OPTIONS.filter(
  (type) => type !== TorrentSelectionRuleType.TorrentSingleVideo && type !== TorrentSelectionRuleType.TorrentFileNameMatch
);

const TORRENT_SELECTION_RULE_LABELS: Record<TorrentSelectionRuleType, string> = {
  [TorrentSelectionRuleType.IndexerPreference]: "索引器偏好",
  [TorrentSelectionRuleType.TitleMatch]: "标题匹配",
  [TorrentSelectionRuleType.PublishDate]: "发布时间",
  [TorrentSelectionRuleType.TitleSimilarity]: "标题相似度",
  [TorrentSelectionRuleType.Seeders]: "Seeders",
  [TorrentSelectionRuleType.Size]: "Size",
  [TorrentSelectionRuleType.TorrentSingleVideo]: "Torrent 单视频优先",
  [TorrentSelectionRuleType.TorrentFileNameMatch]: "Torrent 文件名匹配"
};

const TORRENT_SELECTION_RULE_HINTS: Record<TorrentSelectionRuleType, string> = {
  [TorrentSelectionRuleType.IndexerPreference]: "按索引器优先级挑种",
  [TorrentSelectionRuleType.TitleMatch]: "匹配/排除关键字或正则",
  [TorrentSelectionRuleType.PublishDate]: "按发布新旧比较",
  [TorrentSelectionRuleType.TitleSimilarity]: "与订阅标题相似度",
  [TorrentSelectionRuleType.Seeders]: "按做种人数比较",
  [TorrentSelectionRuleType.Size]: "按文件体积比较",
  [TorrentSelectionRuleType.TorrentSingleVideo]: "优先包含单个视频文件的种子",
  [TorrentSelectionRuleType.TorrentFileNameMatch]: "匹配 torrent 内部文件路径或文件名"
};

function makeRuleId(prefix: string) {
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

function isTorrentInspectionRuleType(type: TorrentSelectionRuleType): boolean {
  return type === TorrentSelectionRuleType.TorrentSingleVideo || type === TorrentSelectionRuleType.TorrentFileNameMatch;
}

function makeTorrentSelectionRule(type: TorrentSelectionRuleType) {
  return {
    id: makeRuleId(type.toLowerCase()),
    name: TORRENT_SELECTION_RULE_LABELS[type],
    type,
    enabled: true,
    direction: TorrentSelectionDirection.Desc,
    indexerPreference: { trackerIds: [] as string[] },
    titleMatch: { clauses: [] as Array<{ pattern: string; patternMode: TitleMatchPatternMode; effect: TitleMatchEffect }> },
    torrentFileNameMatch: { clauses: [] as Array<{ pattern: string; patternMode: TitleMatchPatternMode; effect: TorrentFileMatchEffect }> }
  };
}

type AutomationForm = typeof EMPTY_AUTOMATION_FORM;
type TorrentSelectionRuleDraft = ReturnType<typeof makeTorrentSelectionRule>;
type AutomationFormShape = AutomationForm;

/**
 * 把 AutomationForm 表单数据编码为 GraphQL mutation 所需的 input。
 * 列表保存与抽屉保存都通过此函数构造 payload，保证前后端协议一致。
 */
function buildAutomationSettingsPayload(form: AutomationFormShape) {
  const taskProgressSyncIntervalSeconds = Number.parseInt(form.taskProgressSyncIntervalSeconds.trim(), 10);
  const subscriptionPollIntervalHours = Number.parseInt(form.subscriptionPollIntervalHours.trim(), 10);
  const { rules } = ensureTorrentFileInspectionRules(form.torrentSelection.rules);
  return {
    taskProgressSyncIntervalSeconds: Number.isNaN(taskProgressSyncIntervalSeconds) ? 60 : taskProgressSyncIntervalSeconds,
    subscriptionPollIntervalHours: Number.isNaN(subscriptionPollIntervalHours) ? 1 : subscriptionPollIntervalHours,
    stashBoxEndpoints: form.stashBoxEndpoints,
    torrentSelection: {
      enabled: form.torrentSelection.enabled,
      rules: rules.map((rule: TorrentSelectionRuleDraft) => ({
        id: rule.id,
        name: rule.name.trim(),
        type: rule.type,
        enabled: rule.enabled,
        direction: rule.direction,
        indexerPreference: { trackerIds: rule.indexerPreference.trackerIds },
        titleMatch: {
          clauses: rule.titleMatch.clauses.map((clause: TorrentSelectionRuleDraft["titleMatch"]["clauses"][number]) => ({
            pattern: clause.pattern,
            patternMode: clause.patternMode,
            effect: clause.effect
          }))
        },
        torrentFileNameMatch: {
          clauses: rule.torrentFileNameMatch.clauses.map((clause: TorrentSelectionRuleDraft["torrentFileNameMatch"]["clauses"][number]) => ({
            pattern: clause.pattern,
            patternMode: clause.patternMode,
            effect: clause.effect
          }))
        }
      }))
    }
  };
}

/**
 * 单条规则的只读摘要：按 type 反映关键配置，紧凑展示给列表页。
 */
function buildRuleSummary(rule: TorrentSelectionRuleDraft): string {
  const dirLabel = rule.direction === TorrentSelectionDirection.Asc ? "ASC" : "DESC";
  if (rule.type === TorrentSelectionRuleType.IndexerPreference) {
    const count = rule.indexerPreference.trackerIds.length;
    return count === 0
      ? `${dirLabel} · 未配置索引器`
      : `${dirLabel} · 选中 ${count} 个索引器`;
  }
  if (rule.type === TorrentSelectionRuleType.TitleMatch) {
    const count = rule.titleMatch.clauses.length;
    return count === 0
      ? `${dirLabel} · 未配置标题规则`
      : `${dirLabel} · ${count} 条标题匹配规则`;
  }
  if (rule.type === TorrentSelectionRuleType.TorrentSingleVideo) {
    return `${dirLabel} · 命中单视频结构时优先`;
  }
  if (rule.type === TorrentSelectionRuleType.TorrentFileNameMatch) {
    const count = rule.torrentFileNameMatch.clauses.length;
    const hasLock = rule.torrentFileNameMatch.clauses.some((clause) => clause.effect === TorrentFileMatchEffect.Lock);
    return count === 0
      ? `${dirLabel} · 未配置文件名规则`
      : `${dirLabel} · ${count} 条文件名规则${hasLock ? " · 含 LOCK" : ""}`;
  }
  return dirLabel;
}

function displayRuleName(rule: Pick<TorrentSelectionRuleDraft, "name" | "type">): string {
  const name = rule.name.trim();
  return name === "" ? TORRENT_SELECTION_RULE_LABELS[rule.type] : name;
}

function partitionTorrentSelectionRules(rules: TorrentSelectionRuleDraft[]) {
  const fastRules: TorrentSelectionRuleDraft[] = [];
  const fileInspectionRules: TorrentSelectionRuleDraft[] = [];
  for (const rule of rules) {
    if (isTorrentInspectionRuleType(rule.type)) {
      fileInspectionRules.push(rule);
    } else {
      fastRules.push(rule);
    }
  }
  return { fastRules, fileInspectionRules };
}

function cloneTorrentSelectionRule(rule: TorrentSelectionRuleDraft): TorrentSelectionRuleDraft {
  return {
    ...rule,
    indexerPreference: { trackerIds: [...rule.indexerPreference.trackerIds] },
    titleMatch: { clauses: rule.titleMatch.clauses.map((clause) => ({ ...clause })) },
    torrentFileNameMatch: { clauses: rule.torrentFileNameMatch.clauses.map((clause) => ({ ...clause })) }
  };
}

function ensureTorrentFileInspectionRules(rules: TorrentSelectionRuleDraft[]) {
  const { fastRules, fileInspectionRules } = partitionTorrentSelectionRules(rules);
  const byType = new Map(fileInspectionRules.map((rule) => [rule.type, rule]));
  const ensuredFileInspectionRules = DEFAULT_TORRENT_FILE_INSPECTION_RULES.map((rule: TorrentSelectionRuleDraft) => {
    const existing = byType.get(rule.type);
    return cloneTorrentSelectionRule(existing ?? rule);
  });
  return {
    fastRules,
    fileInspectionRules: ensuredFileInspectionRules,
    rules: [...fastRules, ...ensuredFileInspectionRules]
  };
}

/** 给定当前 draft 与新 type，重置类型专属字段，避免跨类型残留旧数据。 */
function withTypeReset(draft: TorrentSelectionRuleDraft, nextType: TorrentSelectionRuleType): TorrentSelectionRuleDraft {
  if (draft.type === nextType) return draft;
  return {
    ...draft,
    type: nextType,
    indexerPreference: nextType === TorrentSelectionRuleType.IndexerPreference ? draft.indexerPreference : { trackerIds: [] },
    titleMatch: nextType === TorrentSelectionRuleType.TitleMatch ? draft.titleMatch : { clauses: [] },
    torrentFileNameMatch: nextType === TorrentSelectionRuleType.TorrentFileNameMatch ? draft.torrentFileNameMatch : { clauses: [] }
  };
}

function torrentSelectionFromRuntime(runtimeSettings: RuntimeSettings) {
  const rawRules = runtimeSettings.automation.torrentSelection.rules.length > 0
    ? runtimeSettings.automation.torrentSelection.rules.map((rule: RuntimeSettings["automation"]["torrentSelection"]["rules"][number]) => ({
        id: rule.id,
        name: rule.name,
        type: rule.type,
        enabled: rule.enabled,
        direction: rule.direction,
        indexerPreference: {
          trackerIds: [...rule.indexerPreference.trackerIds]
        },
        titleMatch: {
          clauses: rule.titleMatch.clauses.map((clause: RuntimeSettings["automation"]["torrentSelection"]["rules"][number]["titleMatch"]["clauses"][number]) => ({
            pattern: clause.pattern,
            patternMode: clause.patternMode,
            effect: clause.effect
          }))
        },
        torrentFileNameMatch: {
          clauses: rule.torrentFileNameMatch.clauses.map((clause: RuntimeSettings["automation"]["torrentSelection"]["rules"][number]["torrentFileNameMatch"]["clauses"][number]) => ({
            pattern: clause.pattern,
            patternMode: clause.patternMode,
            effect: clause.effect
          }))
        }
      }))
    : [...DEFAULT_TORRENT_SELECTION_RULES, ...DEFAULT_TORRENT_FILE_INSPECTION_RULES].map((rule) => cloneTorrentSelectionRule(rule));
  const { rules } = ensureTorrentFileInspectionRules(rawRules);

  return {
    enabled: runtimeSettings.automation.torrentSelection.enabled,
    rules
  };
}

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
  // 规则列表的展开/收起是 UI 偏好：默认按「启用规则链」决定首屏可见性，
  // 之后用户的展开/收起操作与开关状态相互独立，不被开关重置。
  const [rulesExpanded, setRulesExpanded] = useState(() => automationForm.torrentSelection.enabled);

  // 规则编辑抽屉：editingRuleDraft 为 null 时不渲染抽屉。
  const [editingRuleDraft, setEditingRuleDraft] = useState<TorrentSelectionRuleDraft | null>(null);
  const [ruleEditorClosing, setRuleEditorClosing] = useState(false);
  const [ruleEditorSaving, setRuleEditorSaving] = useState(false);

  const openRuleEditor = (rule: TorrentSelectionRuleDraft) => {
    // 深拷贝：避免在抽屉里直接写入 automationForm 的引用
    setEditingRuleDraft(structuredClone(rule));
    setRuleEditorClosing(false);
  };

  const closeRuleEditor = () => {
    if (!editingRuleDraft) return;
    setRuleEditorClosing(true);
    window.setTimeout(() => {
      setEditingRuleDraft(null);
      setRuleEditorClosing(false);
    }, 240);
  };
  const [systemForm, setSystemForm] = useState(EMPTY_SYSTEM_FORM);
  const [pendingIngestQBRootInitialization, setPendingIngestQBRootInitialization] = useState<string | null>(null);

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

  const [{ data: jackettIndexersData, fetching: fetchingJackettIndexers }] = useQuery<
    JackettIndexersDocumentQuery,
    JackettIndexersDocumentQueryVariables
  >({
    query: JackettIndexersDocumentDocument,
    pause: (settingsTab !== "连接" && settingsTab !== "自动化") || (drawer !== "settings" && renderedDrawer !== "settings")
  });
  const jackettIndexers = jackettIndexersData?.jackettIndexers ?? [];

  // 规则编辑抽屉专用的 Jackett 索引器请求：仅在抽屉打开时拉取，避免与连接/自动化设置共用。
  const [{ data: ruleEditorJackettData, fetching: fetchingRuleEditorJackettIndexers }] = useQuery<
    JackettIndexersDocumentQuery,
    JackettIndexersDocumentQueryVariables
  >({
    query: JackettIndexersDocumentDocument,
    pause: !editingRuleDraft
  });
  const ruleEditorJackettIndexers = ruleEditorJackettData?.jackettIndexers ?? [];

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
  const [{ fetching: updatingSystem }, updateSystemSettings] = useMutation<
    UpdateSystemSettingsDocumentMutation,
    UpdateSystemSettingsDocumentMutationVariables
  >(UpdateSystemSettingsDocumentDocument);
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
      downloads: {
        qbRoot: runtimeSettings.ingest.downloads.qbRoot || "",
        mojiRoot: runtimeSettings.ingest.downloads.mojiRoot || ""
      },
      library: {
        mojiRoot: runtimeSettings.ingest.library.mojiRoot || "",
        stashRoot: runtimeSettings.ingest.library.stashRoot || ""
      },
      transfer: {
        action: runtimeSettings.ingest.transfer.action || ""
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
      subscriptionPollIntervalHours: String(runtimeSettings.automation.subscriptionPollIntervalHours || 1),
      stashBoxEndpoints: [...(runtimeSettings.automation.stashBoxEndpoints ?? [])],
      torrentSelection: torrentSelectionFromRuntime(runtimeSettings)
    });
    setSystemForm({
      taskDeletePolicy: runtimeSettings.system.taskDeletePolicy || TaskDeletePolicy.KeepOnly
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

  const submitIngestSettings = async (qbRoot: string) => {
    const result = await updateIngestSettings({
      input: {
        deliveryMode: ingestForm.deliveryMode.trim(),
        downloads: {
          qbRoot,
          mojiRoot: ingestForm.downloads.mojiRoot.trim()
        },
        library: {
          mojiRoot: ingestForm.library.mojiRoot.trim(),
          stashRoot: ingestForm.library.stashRoot.trim()
        },
        transfer: {
          action: ingestForm.transfer.action.trim()
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

  const saveIngestSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const qbRoot = ingestForm.downloads.qbRoot.trim();
    const defaultSavePath = qbittorrentForm.defaultSavePath.trim();
    if (qbRoot == "" && defaultSavePath !== "") {
      setPendingIngestQBRootInitialization(defaultSavePath);
      return;
    }

    await submitIngestSettings(qbRoot);
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
    const payload = buildAutomationSettingsPayload(automationForm);
    const result = await updateAutomationSettings({
      input: payload
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "自动化设置已保存。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveTorrentSelectionSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const payload = buildAutomationSettingsPayload(automationForm);
    const result = await updateAutomationSettings({ input: payload });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "自动选种规则已保存。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveStashBoxPrioritySettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const payload = buildAutomationSettingsPayload(automationForm);
    const result = await updateAutomationSettings({ input: payload });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "Stash-Box 优先级已保存。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  /**
   * 规则编辑抽屉保存：把已编辑的规则写回 automationForm，再走与列表保存同一 mutation 路径。
   * 先基于当前 automationForm 快照计算 nextRules，再同时更新本地状态与 mutation payload，
   * 避免依赖 React state setter 的调度时机而提交到旧规则。
   */
  const saveRuleEditor = async () => {
    if (!editingRuleDraft) return;
    const nextRules = automationForm.torrentSelection.rules.map((rule) =>
      rule.id === editingRuleDraft.id ? editingRuleDraft : rule
    );
    setAutomationForm((current) => ({
      ...current,
      torrentSelection: {
        ...current.torrentSelection,
        rules: nextRules
      }
    }));
    const formForPayload: AutomationForm = {
      ...automationForm,
      torrentSelection: {
        ...automationForm.torrentSelection,
        rules: nextRules
      }
    };
    const payload = buildAutomationSettingsPayload(formForPayload);
    setRuleEditorSaving(true);
    try {
      const result = await updateAutomationSettings({ input: payload });
      if (result.error) {
        pushToast("tone-danger", describeQueryError(result.error));
        return;
      }
      pushToast("tone-success", "规则已保存。");
      await refreshDashboard({ requestPolicy: "network-only" });
      closeRuleEditor();
    } finally {
      setRuleEditorSaving(false);
    }
  };

  const moveCandidateRuleInSection = (from: number, to: number) => {
    setAutomationForm((current) => {
      const { fastRules: currentFastRules, fileInspectionRules: currentFileInspectionRules } = ensureTorrentFileInspectionRules(current.torrentSelection.rules);
      if (to < 0 || to >= currentFastRules.length) return current;
      const nextFastRules = [...currentFastRules];
      const [moved] = nextFastRules.splice(from, 1);
      nextFastRules.splice(to, 0, moved);
      const rules = [...nextFastRules, ...currentFileInspectionRules];
      return {
        ...current,
        torrentSelection: {
          ...current.torrentSelection,
          rules
        }
      };
    });
  };

  const addCandidateRule = (type: TorrentSelectionRuleType) => {
    setAutomationForm((current) => {
      const { fastRules: currentFastRules, fileInspectionRules: currentFileInspectionRules } = ensureTorrentFileInspectionRules(current.torrentSelection.rules);
      const nextRule = makeTorrentSelectionRule(type);
      const rules = [...currentFastRules, nextRule, ...currentFileInspectionRules];
      return {
        ...current,
        torrentSelection: {
          ...current.torrentSelection,
          rules
        }
      };
    });
  };

  const updateCandidateRule = (ruleId: string, updater: (rule: typeof automationForm.torrentSelection.rules[number]) => typeof automationForm.torrentSelection.rules[number]) => {
    setAutomationForm((current) => ({
      ...current,
      torrentSelection: {
        ...current.torrentSelection,
        rules: current.torrentSelection.rules.map((rule) => (rule.id === ruleId ? updater(rule) : rule))
      }
    }));
  };

  const removeCandidateRule = (ruleId: string) => {
    setAutomationForm((current) => ({
      ...current,
      torrentSelection: {
        ...current.torrentSelection,
        rules: current.torrentSelection.rules.filter((rule) => rule.id !== ruleId || isTorrentInspectionRuleType(rule.type))
      }
    }));
  };

  const toggleIndexerPreference = (ruleId: string, trackerId: string) => {
    updateCandidateRule(ruleId, (rule) => {
      const exists = rule.indexerPreference.trackerIds.includes(trackerId);
      return {
        ...rule,
        indexerPreference: {
          trackerIds: exists
            ? rule.indexerPreference.trackerIds.filter((id) => id !== trackerId)
            : [...rule.indexerPreference.trackerIds, trackerId]
        }
      };
    });
  };

  const moveIndexerPreference = (ruleId: string, from: number, to: number) => {
    updateCandidateRule(ruleId, (rule) => {
      if (to < 0 || to >= rule.indexerPreference.trackerIds.length) return rule;
      const trackerIds = [...rule.indexerPreference.trackerIds];
      const [moved] = trackerIds.splice(from, 1);
      trackerIds.splice(to, 0, moved);
      return {
        ...rule,
        indexerPreference: { trackerIds }
      };
    });
  };

  const addTitleMatchClause = (ruleId: string) => {
    updateCandidateRule(ruleId, (rule) => ({
      ...rule,
      titleMatch: {
        clauses: [
          ...rule.titleMatch.clauses,
          {
            pattern: "",
            patternMode: TitleMatchPatternMode.Plain,
            effect: TitleMatchEffect.Prefer
          }
        ]
      }
    }));
  };

  const updateTitleMatchClause = (
    ruleId: string,
    clauseIndex: number,
    updater: (clause: typeof automationForm.torrentSelection.rules[number]["titleMatch"]["clauses"][number]) => typeof automationForm.torrentSelection.rules[number]["titleMatch"]["clauses"][number]
  ) => {
    updateCandidateRule(ruleId, (rule) => ({
      ...rule,
      titleMatch: {
        clauses: rule.titleMatch.clauses.map((clause, index) => (index === clauseIndex ? updater(clause) : clause))
      }
    }));
  };

  const moveTitleMatchClause = (ruleId: string, from: number, to: number) => {
    updateCandidateRule(ruleId, (rule) => {
      if (to < 0 || to >= rule.titleMatch.clauses.length) return rule;
      const clauses = [...rule.titleMatch.clauses];
      const [moved] = clauses.splice(from, 1);
      clauses.splice(to, 0, moved);
      return {
        ...rule,
        titleMatch: { clauses }
      };
    });
  };

  const removeTitleMatchClause = (ruleId: string, clauseIndex: number) => {
    updateCandidateRule(ruleId, (rule) => ({
      ...rule,
      titleMatch: {
        clauses: rule.titleMatch.clauses.filter((_, index) => index !== clauseIndex)
      }
    }));
  };

  const addTorrentFileNameMatchClause = (ruleId: string) => {
    updateCandidateRule(ruleId, (rule) => ({
      ...rule,
      torrentFileNameMatch: {
        clauses: [
          ...rule.torrentFileNameMatch.clauses,
          {
            pattern: "",
            patternMode: TitleMatchPatternMode.Plain,
            effect: TorrentFileMatchEffect.Prefer
          }
        ]
      }
    }));
  };

  const updateTorrentFileNameMatchClause = (
    ruleId: string,
    clauseIndex: number,
    updater: (clause: typeof automationForm.torrentSelection.rules[number]["torrentFileNameMatch"]["clauses"][number]) => typeof automationForm.torrentSelection.rules[number]["torrentFileNameMatch"]["clauses"][number]
  ) => {
    updateCandidateRule(ruleId, (rule) => ({
      ...rule,
      torrentFileNameMatch: {
        clauses: rule.torrentFileNameMatch.clauses.map((clause, index) => (index === clauseIndex ? updater(clause) : clause))
      }
    }));
  };

  const removeTorrentFileNameMatchClause = (ruleId: string, clauseIndex: number) => {
    updateCandidateRule(ruleId, (rule) => ({
      ...rule,
      torrentFileNameMatch: {
        clauses: rule.torrentFileNameMatch.clauses.filter((_, index) => index !== clauseIndex)
      }
    }));
  };

  const saveSystemSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = await updateSystemSettings({
      input: {
        taskDeletePolicy: systemForm.taskDeletePolicy as TaskDeletePolicy
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "系统设置已保存。");
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

  const { fastRules, fileInspectionRules } = ensureTorrentFileInspectionRules(automationForm.torrentSelection.rules);

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
            <button type="submit" disabled={updatingStash}>
              {updatingStash ? "保存中..." : "保存 Stash 连接"}
            </button>
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
            <button type="submit" disabled={updatingJackett}>
              {updatingJackett ? "保存中..." : "保存 Jackett 连接"}
            </button>
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
            <button type="submit" disabled={updatingQBittorrent}>
              {updatingQBittorrent ? "保存中..." : "保存 qBittorrent 连接"}
            </button>
          </div>
        </form>
      </article>
    );
  }

  if (settingsTab === "入库") {
    const stashLibraries = runtimeStatus.stashLibraries ?? [];
    const stashLibrariesLoadError = runtimeStatus.stashLibrariesLoadError ?? null;
    const qbDefaultSavePath = qbittorrentForm.defaultSavePath.trim();
    const deliveryModeInfo = ingestForm.deliveryMode === "PATH_MAP"
      ? "Moji 只把任务里的 qB 下载路径翻译成 Stash 扫描路径，不直接搬运文件。"
      : "Moji 先把 qB 下载路径翻译成自己的可操作源路径，再交付到媒体库，并把同一相对路径翻译成 Stash 扫描路径。";
    const qbRootInfo = "填写 qBittorrent 视角下的下载根目录。任务里的 ContentPath / SavePath 会先基于这个根路径计算相对路径。";
    const mojiDownloadsRootInfo = "填写 Moji 视角下的下载根目录。TRANSFER 模式会把上一步得到的相对路径拼到这里，得到 Moji 实际读取的源路径。";
    const mojiLibraryRootInfo = "填写 Moji 视角下的媒体库根目录。TRANSFER 模式会把相对路径拼到这里，得到 Moji 实际写入的交付目标。";
    const stashLibraryInfo = "填写 Stash 视角下的媒体库根目录。无论使用 PATH_MAP 还是 TRANSFER，Moji 最终都会把相对路径拼到这里并通知 Stash 扫描。";
    const transferActionInfo = "COPY 会保留下载区原文件，MOVE 会把文件迁移进媒体库，SYMLINK 会在媒体库里创建指向源文件的符号链接。目标已存在同名文件或链接时会直接失败。";

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
            <FieldLabel text="qB 下载根目录" info={qbRootInfo} />
            <div className="secret-input">
              <input
                className="secret-input__field"
                value={ingestForm.downloads.qbRoot}
                onChange={(event) => setIngestForm((current) => ({
                  ...current,
                  downloads: { ...current.downloads, qbRoot: event.target.value }
                }))}
                placeholder="/downloads"
              />
              <button
                type="button"
                className="secret-input__toggle"
                disabled={qbDefaultSavePath === ""}
                onClick={() => setIngestForm((current) => ({
                  ...current,
                  downloads: { ...current.downloads, qbRoot: qbDefaultSavePath }
                }))}
                aria-label="使用 qB 默认下载目录"
                title={qbDefaultSavePath === "" ? "当前未配置 qB 默认下载目录" : `使用 ${qbDefaultSavePath} 初始化 qB 下载根目录`}
              >
                <FontAwesomeIcon icon={faRotate} aria-hidden="true" />
              </button>
            </div>
          </label>
          <label className="settings-field">
            <FieldLabel text="Stash 媒体库根目录" info={stashLibraryInfo} />
            <select
              value={ingestForm.library.stashRoot}
              onChange={(event) => setIngestForm((current) => ({
                ...current,
                library: { ...current.library, stashRoot: event.target.value }
              }))}
            >
              <option value="" disabled hidden>{stashLibraries.length > 0 ? "请选择 Stash 媒体库根目录" : "暂无可用媒体库路径"}</option>
              {ingestForm.library.stashRoot !== "" && !stashLibraries.some((library) => library.path === ingestForm.library.stashRoot) ? (
                <option value={ingestForm.library.stashRoot}>{ingestForm.library.stashRoot}</option>
              ) : null}
              {stashLibraries.map((library) => (
                <option key={library.path} value={library.path}>{library.path}</option>
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
                <FieldLabel text="Moji 下载根目录" info={mojiDownloadsRootInfo} />
                <input
                  value={ingestForm.downloads.mojiRoot}
                  onChange={(event) => setIngestForm((current) => ({
                    ...current,
                    downloads: { ...current.downloads, mojiRoot: event.target.value }
                  }))}
                  placeholder="/srv/downloads"
                />
              </label>
              <label className="settings-field">
                <FieldLabel text="Moji 媒体库根目录" info={mojiLibraryRootInfo} />
                <input
                  value={ingestForm.library.mojiRoot}
                  onChange={(event) => setIngestForm((current) => ({
                    ...current,
                    library: { ...current.library, mojiRoot: event.target.value }
                  }))}
                  placeholder="/srv/media-library"
                />
              </label>
            </>
          ) : null}

          <div className="settings-actions">
            <button type="submit" disabled={updatingIngest}>
              {updatingIngest ? "保存中..." : "保存入库设置"}
            </button>
          </div>
        </form>

        {pendingIngestQBRootInitialization ? (
          <div className="drawer-scrim drawer-scrim--modal" onClick={() => setPendingIngestQBRootInitialization(null)}>
            <aside className="drawer drawer--modal" onClick={(event) => event.stopPropagation()}>
              <div className="drawer__head">
                <div>
                  <h2>初始化 qB 下载根目录</h2>
                </div>
                <button
                  type="button"
                  className="ghost-button"
                  onClick={() => setPendingIngestQBRootInitialization(null)}
                  disabled={updatingIngest}
                >
                  关闭
                </button>
              </div>
              <div className="drawer-body">
                <div className="drawer-stack">
                  <article className="drawer-card">
                    <div className="drawer-card__head">
                      <div>
                        <h3>qB 下载根目录当前为空</h3>
                        <p>是否使用 qB 默认下载目录 {pendingIngestQBRootInitialization} 初始化？</p>
                      </div>
                    </div>
                    <div className="settings-actions">
                      <button
                        type="button"
                        onClick={() => {
                          setIngestForm((current) => ({
                            ...current,
                            downloads: { ...current.downloads, qbRoot: pendingIngestQBRootInitialization }
                          }));
                          void (async () => {
                            await submitIngestSettings(pendingIngestQBRootInitialization);
                            setPendingIngestQBRootInitialization(null);
                          })();
                        }}
                        disabled={updatingIngest}
                      >
                        {updatingIngest ? "保存中..." : "使用默认目录并保存"}
                      </button>
                      <button
                        type="button"
                        className="ghost-button"
                        onClick={() => {
                          void (async () => {
                            await submitIngestSettings("");
                            setPendingIngestQBRootInitialization(null);
                          })();
                        }}
                        disabled={updatingIngest}
                      >
                        保持为空并保存
                      </button>
                    </div>
                  </article>
                </div>
              </div>
            </aside>
          </div>
        ) : null}
      </article>
    );
  }

  if (settingsTab === "自动化") {
    const stashBoxes = runtimeStatus.subscription.stashBoxes ?? [];
    const loaded = runtimeStatus.subscription.stashBoxesLoaded;
    const loadError = runtimeStatus.subscription.stashBoxesLoadError;
    const order = automationForm.stashBoxEndpoints;
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
      setAutomationForm((current) => ({ ...current, stashBoxEndpoints: next }));
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
            <span>订阅轮询间隔（小时）</span>
            <input
              value={automationForm.subscriptionPollIntervalHours}
              onChange={(event) => setAutomationForm((current) => ({ ...current, subscriptionPollIntervalHours: event.target.value }))}
              placeholder="1"
            />
          </label>
          <div className="settings-meta">
            <span>任务同步: {runtimeStatus.automation.taskProgressSyncEnabled ? "已启用" : "未启用"}</span>
            <span>订阅轮询: {runtimeStatus.automation.subscriptionPollEnabled ? "已启用" : "未启用"}</span>
          </div>
          <div className="settings-actions">
            <button type="submit" disabled={updatingAutomation}>
              {updatingAutomation ? "保存中..." : "保存自动化设置"}
            </button>
          </div>
        </form>

        <form className="settings-form" onSubmit={(event) => void saveStashBoxPrioritySettings(event)}>
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
                <button type="submit" disabled={updatingAutomation}>
                  {updatingAutomation ? "保存中..." : "保存优先级"}
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

        <form className="settings-form" onSubmit={(event) => void saveTorrentSelectionSettings(event)}>
          <section className="torrent-rules">
            <header className="torrent-rules__head">
              <div>
                <h4>自动选种规则</h4>
                <p className="torrent-rules__sub">
                  仅影响后端自动挑选下载候选，不影响 Jackett 搜索结果列表展示。规则按从上到下顺序依次比较。
                </p>
              </div>
              <div className="torrent-rules__save">
                <button type="submit" disabled={updatingAutomation}>
                  {updatingAutomation ? "保存中..." : "保存自动选种规则"}
                </button>
              </div>
            </header>

            <div className="torrent-rules__toolbar">
              <label className="switch-row">
                <span className="switch-row__label">启用规则链</span>
                <span className="switch" role="switch" aria-checked={automationForm.torrentSelection.enabled}>
                  <input
                    type="checkbox"
                    checked={automationForm.torrentSelection.enabled}
                    onChange={(event) => {
                      const next = event.target.checked;
                      setAutomationForm((current) => ({
                        ...current,
                        torrentSelection: {
                          ...current.torrentSelection,
                          enabled: next
                        }
                      }));
                      // 打开规则链时自动展开列表，让用户能立刻看到并编辑；
                      // 关闭时不动展开偏好，保留用户的 UI 选择。
                      if (next) setRulesExpanded(true);
                    }}
                  />
                  <span className="switch__track" aria-hidden="true" />
                  <span className="switch__thumb" aria-hidden="true" />
                </span>
              </label>
              <button
                type="button"
                className="ghost-button torrent-rules__expand"
                aria-expanded={rulesExpanded}
                aria-controls="torrent-rules-list"
                onClick={() => setRulesExpanded((current) => !current)}
              >
                <FontAwesomeIcon icon={rulesExpanded ? faChevronUp : faChevronDown} />
                <span>{rulesExpanded ? "收起" : "展开"}</span>
              </button>
            </div>

            {!rulesExpanded && automationForm.torrentSelection.rules.length > 0 ? (
              <p className="torrent-rules__hint">
                当前已配置 {automationForm.torrentSelection.rules.length} 条规则，启用后才会生效。
              </p>
            ) : null}

            <div
              id="torrent-rules-list"
              className="torrent-rules__list"
              hidden={!rulesExpanded}
            >
              {fastRules.map((rule, ruleIndex) => {
                const ruleClasses = ["torrent-rule"];
                if (!rule.enabled) ruleClasses.push("is-disabled");
                if (draggedIndex === ruleIndex) ruleClasses.push("is-dragging");
                if (dragOverIndex === ruleIndex) {
                  ruleClasses.push(dragOverIndex < (draggedIndex ?? -1) ? "is-drop-top" : "is-drop-bottom");
                }

                return (
                <article
                  key={rule.id}
                  className={ruleClasses.join(" ")}
                  draggable
                  onDragStart={(event) => {
                    event.dataTransfer.effectAllowed = "move";
                    event.dataTransfer.setData("text/plain", String(ruleIndex));
                    setDraggedIndex(ruleIndex);
                  }}
                  onDragOver={(event) => {
                    event.preventDefault();
                    event.dataTransfer.dropEffect = "move";
                    setDragOverIndex(ruleIndex);
                  }}
                  onDragLeave={() => {
                    if (dragOverIndex === ruleIndex) setDragOverIndex(null);
                  }}
                  onDrop={(event) => {
                    event.preventDefault();
                    const from = Number.parseInt(event.dataTransfer.getData("text/plain"), 10);
                    moveCandidateRuleInSection(Number.isNaN(from) ? draggedIndex ?? ruleIndex : from, ruleIndex);
                    setDraggedIndex(null);
                    setDragOverIndex(null);
                  }}
                  onDragEnd={() => {
                    setDraggedIndex(null);
                    setDragOverIndex(null);
                  }}
                >
                  <header className="torrent-rule__head">
                    <span className="torrent-rule__handle" aria-hidden="true" title="拖动以重新排序">
                      <FontAwesomeIcon icon={faGripVertical} />
                    </span>
                    <span className="torrent-rule__order">{ruleIndex + 1}</span>
                    <h3 className="torrent-rule__name">{displayRuleName(rule)}</h3>
                    <div className="torrent-rule__inline-readonly" aria-hidden="true">
                      <span className="torrent-rule__badge torrent-rule__badge--type">
                        {TORRENT_SELECTION_RULE_LABELS[rule.type]}
                      </span>
                      <span className="torrent-rule__badge torrent-rule__badge--dir">
                        {rule.direction === TorrentSelectionDirection.Asc ? "ASC" : "DESC"}
                      </span>
                      <span className={`torrent-rule__badge torrent-rule__badge--status${rule.enabled ? " is-on" : " is-off"}`}>
                        {rule.enabled ? "启用" : "未启用"}
                      </span>
                    </div>
                    <div className="torrent-rule__actions">
                      <button
                        type="button"
                        className="ghost-button"
                        onClick={() => openRuleEditor(rule)}
                      >
                        <FontAwesomeIcon icon={faPenToSquare} />
                        <span>编辑</span>
                      </button>
                      <button type="button" className="ghost-button ghost-button--icon" onClick={() => moveCandidateRuleInSection(ruleIndex, ruleIndex - 1)} disabled={ruleIndex === 0} aria-label="上移">
                        <FontAwesomeIcon icon={faArrowUp} />
                      </button>
                      <button
                        type="button"
                        className="ghost-button ghost-button--icon"
                        onClick={() => moveCandidateRuleInSection(ruleIndex, ruleIndex + 1)}
                        disabled={ruleIndex === fastRules.length - 1}
                        aria-label="下移"
                      >
                        <FontAwesomeIcon icon={faArrowDown} />
                      </button>
                      <button type="button" className="ghost-button" onClick={() => removeCandidateRule(rule.id)}>
                        删除
                      </button>
                    </div>
                  </header>

                  <p className="torrent-rule__summary">{buildRuleSummary(rule)}</p>
                </article>
                );
              })}

              <div className="torrent-rules__add-bar">
                <div className="torrent-rules__add-copy">
                  <strong>{fastRules.length === 0 ? "尚未添加任何快速规则" : "添加快速规则"}</strong>
                  <span>这些规则会在所有候选上先执行，按从上到下顺序依次比较。</span>
                </div>
                <Menu
                  className="torrent-rules__menu"
                  ariaLabel="选择快速规则类型"
                  triggerAriaLabel="添加快速规则"
                  align="end"
                  triggerLabel={
                    <>
                      <FontAwesomeIcon icon={faPlus} />
                      <span>添加快速规则</span>
                      <FontAwesomeIcon icon={faChevronDown} className="menu__trigger-caret" />
                    </>
                  }
                  items={FAST_TORRENT_SELECTION_RULE_TYPE_OPTIONS.map((type) => ({
                    key: type,
                    label: TORRENT_SELECTION_RULE_LABELS[type],
                    hint: TORRENT_SELECTION_RULE_HINTS[type],
                    onSelect: () => addCandidateRule(type)
                  }))}
                />
              </div>

              <div className="torrent-rules__divider" aria-hidden="true" />
              <p className="torrent-rules__note">
                Torrent 文件结构精排固定在快速规则之后执行，只检查首轮排序后的前 5 个且带 `.torrent` 链接的候选；下方两条规则为内建策略，可启用、禁用和编辑内容，但不支持新增、删除或改类型。
              </p>

              {fileInspectionRules.map((rule: TorrentSelectionRuleDraft, ruleIndex: number) => {
                const displayIndex = fastRules.length + ruleIndex + 1;
                return (
                  <article key={rule.id} className={`torrent-rule${rule.enabled ? "" : " is-disabled"}`}>
                    <header className="torrent-rule__head">
                      <span className="torrent-rule__order">{displayIndex}</span>
                      <h3 className="torrent-rule__name">{TORRENT_SELECTION_RULE_LABELS[rule.type]}</h3>
                      <div className="torrent-rule__inline-readonly" aria-hidden="true">
                        <span className="torrent-rule__badge torrent-rule__badge--type">
                          {TORRENT_SELECTION_RULE_LABELS[rule.type]}
                        </span>
                        <span className="torrent-rule__badge torrent-rule__badge--dir">
                          {rule.direction === TorrentSelectionDirection.Asc ? "ASC" : "DESC"}
                        </span>
                      </div>
                      <div className="torrent-rule__actions">
                        <label className="switch switch--sm" role="switch" aria-checked={rule.enabled} aria-label={`${TORRENT_SELECTION_RULE_LABELS[rule.type]}启用开关`}>
                          <input
                            type="checkbox"
                            checked={rule.enabled}
                            onChange={(event) =>
                              updateCandidateRule(rule.id, (current) => ({
                                ...current,
                                enabled: event.target.checked
                              }))
                            }
                          />
                          <span className="switch__track" aria-hidden="true" />
                          <span className="switch__thumb" aria-hidden="true" />
                        </label>
                      </div>
                    </header>

                    <p className="torrent-rule__summary">{buildRuleSummary(rule)}</p>
                    <div className="settings-form">
                      {rule.type === TorrentSelectionRuleType.TorrentSingleVideo ? (
                        <p className="torrent-rule__hint">
                          只检查首轮排序后的前 5 个且带 `.torrent` 链接的候选；命中“单个视频文件”结构时优先。`magnet` 不参与文件结构检查。
                        </p>
                      ) : null}

                      {rule.type === TorrentSelectionRuleType.TorrentFileNameMatch ? (
                        <>
                          <div className="drawer-card__head">
                            <div>
                              <p className="torrent-rule__hint">按顺序匹配 torrent 内部文件路径或文件名</p>
                              <p className="torrent-rule__hint">PLAIN 为纯文本，REGEX 为正则，LOCK 命中后直接选中。</p>
                            </div>
                            <button
                              type="button"
                              className="ghost-button"
                              onClick={() => addTorrentFileNameMatchClause(rule.id)}
                            >
                              添加规则
                            </button>
                          </div>
                          {rule.torrentFileNameMatch.clauses.length === 0 ? (
                            <p className="torrent-rule__hint">尚未添加文件名匹配规则。</p>
                          ) : (
                            <div className="torrent-rule__clauses">
                              {rule.torrentFileNameMatch.clauses.map((clause, clauseIndex) => (
                                <div key={`${rule.id}-inline-torrent-file-clause-${clauseIndex}`} className="torrent-rule__clause">
                                  <input
                                    className="torrent-rule__clause-pattern"
                                    value={clause.pattern}
                                    onChange={(event) =>
                                      updateTorrentFileNameMatchClause(rule.id, clauseIndex, (current) => ({
                                        ...current,
                                        pattern: event.target.value
                                      }))
                                    }
                                    placeholder="Pattern：hhd800.com 或 /sample/i"
                                    aria-label="Torrent 文件名 Pattern"
                                  />
                                  <select
                                    className="torrent-rule__clause-mode"
                                    value={clause.patternMode}
                                    onChange={(event) =>
                                      updateTorrentFileNameMatchClause(rule.id, clauseIndex, (current) => ({
                                        ...current,
                                        patternMode: event.target.value as TitleMatchPatternMode
                                      }))
                                    }
                                    aria-label="匹配模式"
                                  >
                                    <option value={TitleMatchPatternMode.Plain}>PLAIN</option>
                                    <option value={TitleMatchPatternMode.Regex}>REGEX</option>
                                  </select>
                                  <select
                                    className="torrent-rule__clause-effect"
                                    value={clause.effect}
                                    onChange={(event) =>
                                      updateTorrentFileNameMatchClause(rule.id, clauseIndex, (current) => ({
                                        ...current,
                                        effect: event.target.value as TorrentFileMatchEffect
                                      }))
                                    }
                                    aria-label="效果"
                                  >
                                    <option value={TorrentFileMatchEffect.Prefer}>PREFER</option>
                                    <option value={TorrentFileMatchEffect.Avoid}>AVOID</option>
                                    <option value={TorrentFileMatchEffect.Lock}>LOCK</option>
                                  </select>
                                  <div className="torrent-rule__clause-actions">
                                    <button
                                      type="button"
                                      className="ghost-button"
                                      onClick={() => removeTorrentFileNameMatchClause(rule.id, clauseIndex)}
                                    >
                                      删除
                                    </button>
                                  </div>
                                </div>
                              ))}
                            </div>
                          )}
                        </>
                      ) : null}
                    </div>
                  </article>
                );
              })}
            </div>
          </section>

          {editingRuleDraft ? renderRuleEditor() : null}
        </form>
      </article>
    );
  }

  /**
   * 规则编辑抽屉的内联实现：复用既有 .drawer-scrim/.drawer 样式，与
   * pendingIngestQBRootInitialization 走相同的 inline-modal 路径，不影响 App / DrawerKey。
   * 编辑期间对 draft 的所有改动都不会写回 automationForm，直到点保存。
   */
  function renderRuleEditor() {
    if (!editingRuleDraft) return null;
    const draft = editingRuleDraft;

    const setDraft = (updater: (current: TorrentSelectionRuleDraft) => TorrentSelectionRuleDraft) => {
      setEditingRuleDraft((current) => (current ? updater(current) : current));
    };

    const setDraftType = (nextType: TorrentSelectionRuleType) => {
      setEditingRuleDraft((current) => (current ? withTypeReset(current, nextType) : current));
    };

    const close = () => {
      if (ruleEditorSaving) return;
      closeRuleEditor();
    };

    const scrimClass = `drawer-scrim drawer-scrim--modal${ruleEditorClosing ? " is-closing" : ""}`;
    // 用 --popover 修饰：作为「设置页之上的小一圈弹窗」呈现，
    // 不与全幅 --modal（配置与系统、删除确认）同级，视觉上明显是二级弹窗。
    const drawerClass = `drawer drawer--popover${ruleEditorClosing ? " is-closing" : ""}`;

    // 必须用 portal 渲染到 document.body：
    // `.drawer-card` 祖先链上有 `backdrop-filter`（来自 `%card-base` 的 `@mixin card-base`），
    // 会为 position:fixed 的后代新建 containing block，导致 drawer 跟随 settings 的滚动条一起移动，
    // 而不是悬浮于 viewport。portal 到 body 之后，画法与真正的全局 .drawer 一致。
    return createPortal(
      <div className={scrimClass} onClick={close}>
        <aside className={drawerClass} onClick={(event) => event.stopPropagation()}>
          <div className="drawer__head">
            <div>
              <h2>编辑规则 · {displayRuleName(draft)}</h2>
            </div>
            <button type="button" className="ghost-button" onClick={close} disabled={ruleEditorSaving}>
              关闭
            </button>
          </div>
          <div className="drawer-body">
            <div className="drawer-stack">
              <article className="drawer-card">
                <div className="drawer-card__head">
                  <div>
                    <h3>基础</h3>
                  </div>
                </div>
                <form
                  className="settings-form"
                  onSubmit={(event) => {
                    event.preventDefault();
                    void saveRuleEditor();
                  }}
                >
                  <label className="settings-field">
                    <FieldLabel text="规则名称" info="仅用于展示和识别这条规则，不影响排序逻辑。" />
                    <input
                      value={draft.name}
                      onChange={(event) =>
                        setDraft((current) => ({
                          ...current,
                          name: event.target.value
                        }))
                      }
                      placeholder={TORRENT_SELECTION_RULE_LABELS[draft.type]}
                    />
                  </label>
                  <label className="settings-field">
                    <FieldLabel text="规则类型" />
                    <select
                      value={draft.type}
                      onChange={(event) => setDraftType(event.target.value as TorrentSelectionRuleType)}
                    >
                      {FAST_TORRENT_SELECTION_RULE_TYPE_OPTIONS.map((type) => (
                        <option key={type} value={type}>
                          {TORRENT_SELECTION_RULE_LABELS[type]}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="settings-field">
                    <FieldLabel text="方向" />
                    <select
                      value={draft.direction}
                      onChange={(event) =>
                        setDraft((current) => ({
                          ...current,
                          direction: event.target.value as TorrentSelectionDirection
                        }))
                      }
                    >
                      <option value={TorrentSelectionDirection.Desc}>DESC</option>
                      <option value={TorrentSelectionDirection.Asc}>ASC</option>
                    </select>
                  </label>
                  <label className="settings-field">
                    <FieldLabel text="启用" />
                    <span className="switch switch--sm" role="switch" aria-checked={draft.enabled}>
                      <input
                        type="checkbox"
                        checked={draft.enabled}
                        onChange={(event) =>
                          setDraft((current) => ({ ...current, enabled: event.target.checked }))
                        }
                      />
                      <span className="switch__track" aria-hidden="true" />
                      <span className="switch__thumb" aria-hidden="true" />
                    </span>
                  </label>
                </form>
              </article>

              {draft.type === TorrentSelectionRuleType.IndexerPreference ? (
                <article className="drawer-card">
                  <div className="drawer-card__head">
                    <div>
                      <h3>索引器偏好顺序</h3>
                      <p>勾选右侧索引器加入「偏好顺序」，用 ↑ ↓ 调整优先级。</p>
                    </div>
                  </div>
                  <div className="settings-form">
                    <div>
                      <strong className="torrent-rule__subhead">偏好顺序</strong>
                      {draft.indexerPreference.trackerIds.length === 0 ? (
                        <p className="torrent-rule__hint">未选择索引器，未命中时保持默认稳定顺序。</p>
                      ) : (
                        <ol className="torrent-rule__chips">
                          {draft.indexerPreference.trackerIds.map((trackerId, index) => (
                            <li key={`${draft.id}-selected-${trackerId}`} className="torrent-rule__chip">
                              <button
                                type="button"
                                className="torrent-rule__chip-handle"
                                onClick={() =>
                                  setDraft((current) => {
                                    if (!current) return current;
                                    const ids = [...current.indexerPreference.trackerIds];
                                    if (index <= 0) return current;
                                    [ids[index - 1], ids[index]] = [ids[index], ids[index - 1]];
                                    return {
                                      ...current,
                                      indexerPreference: { trackerIds: ids }
                                    };
                                  })
                                }
                                disabled={index === 0}
                                aria-label="上移"
                              >
                                <FontAwesomeIcon icon={faArrowUp} />
                              </button>
                              <span className="torrent-rule__chip-label" title={trackerId}>
                                {trackerId}
                              </span>
                              <button
                                type="button"
                                className="torrent-rule__chip-remove"
                                onClick={() =>
                                  setDraft((current) =>
                                    current
                                      ? {
                                          ...current,
                                          indexerPreference: {
                                            trackerIds: current.indexerPreference.trackerIds.filter((id) => id !== trackerId)
                                          }
                                        }
                                      : current
                                  )
                                }
                                aria-label="移除"
                              >
                                <FontAwesomeIcon icon={faTimes} />
                              </button>
                            </li>
                          ))}
                        </ol>
                      )}
                    </div>
                    <div>
                      <strong className="torrent-rule__subhead">可选索引器</strong>
                      <div className="torrent-rule__candidates">
                        {fetchingRuleEditorJackettIndexers ? (
                          <span className="torrent-rule__hint">加载中…</span>
                        ) : null}
                        {!fetchingRuleEditorJackettIndexers && ruleEditorJackettIndexers.length === 0 ? (
                          <span className="torrent-rule__hint">当前没有可用的 Jackett 索引器。</span>
                        ) : null}
                        {ruleEditorJackettIndexers.map((indexer: JackettIndexer) => {
                          const selected = draft.indexerPreference.trackerIds.includes(indexer.id);
                          return (
                            <button
                              key={indexer.id}
                              type="button"
                              className="ghost-button"
                              disabled={selected}
                              onClick={() =>
                                setDraft((current) =>
                                  current
                                    ? {
                                        ...current,
                                        indexerPreference: {
                                          trackerIds: current.indexerPreference.trackerIds.includes(indexer.id)
                                            ? current.indexerPreference.trackerIds
                                            : [...current.indexerPreference.trackerIds, indexer.id]
                                        }
                                      }
                                    : current
                                )
                              }
                            >
                              {indexer.name}
                            </button>
                          );
                        })}
                      </div>
                    </div>
                  </div>
                </article>
              ) : null}

              {draft.type === TorrentSelectionRuleType.TitleMatch ? (
                <article className="drawer-card">
                  <div className="drawer-card__head">
                    <div>
                      <h3>标题匹配规则</h3>
                      <p>添加关键字或正则，按顺序匹配；PLAIN 表示纯文本，REGEX 表示正则表达式。</p>
                    </div>
                    <button
                      type="button"
                      className="ghost-button"
                      onClick={() =>
                        setDraft((current) =>
                          current
                            ? {
                                ...current,
                                titleMatch: {
                                  clauses: [
                                    ...current.titleMatch.clauses,
                                    {
                                      pattern: "",
                                      patternMode: TitleMatchPatternMode.Plain,
                                      effect: TitleMatchEffect.Prefer
                                    }
                                  ]
                                }
                              }
                            : current
                        )
                      }
                    >
                      添加标题规则
                    </button>
                  </div>
                  {draft.titleMatch.clauses.length === 0 ? (
                    <p className="torrent-rule__hint">尚未添加标题匹配规则。</p>
                  ) : (
                    <div className="torrent-rule__clauses">
                      {draft.titleMatch.clauses.map((clause, clauseIndex) => (
                        <div key={`${draft.id}-clause-${clauseIndex}`} className="torrent-rule__clause">
                          <input
                            className="torrent-rule__clause-pattern"
                            value={clause.pattern}
                            onChange={(event) =>
                              setDraft((current) => {
                                if (!current) return current;
                                const clauses = [...current.titleMatch.clauses];
                                clauses[clauseIndex] = { ...clauses[clauseIndex], pattern: event.target.value };
                                return { ...current, titleMatch: { clauses } };
                              })
                            }
                            placeholder="Pattern：uncensored 或 /\\b4k\\b/i"
                            aria-label="Pattern"
                          />
                          <select
                            className="torrent-rule__clause-mode"
                            value={clause.patternMode}
                            onChange={(event) =>
                              setDraft((current) => {
                                if (!current) return current;
                                const clauses = [...current.titleMatch.clauses];
                                clauses[clauseIndex] = {
                                  ...clauses[clauseIndex],
                                  patternMode: event.target.value as TitleMatchPatternMode
                                };
                                return { ...current, titleMatch: { clauses } };
                              })
                            }
                            aria-label="匹配模式"
                          >
                            <option value={TitleMatchPatternMode.Plain}>PLAIN</option>
                            <option value={TitleMatchPatternMode.Regex}>REGEX</option>
                          </select>
                          <select
                            className="torrent-rule__clause-effect"
                            value={clause.effect}
                            onChange={(event) =>
                              setDraft((current) => {
                                if (!current) return current;
                                const clauses = [...current.titleMatch.clauses];
                                clauses[clauseIndex] = {
                                  ...clauses[clauseIndex],
                                  effect: event.target.value as TitleMatchEffect
                                };
                                return { ...current, titleMatch: { clauses } };
                              })
                            }
                            aria-label="效果"
                          >
                            <option value={TitleMatchEffect.Prefer}>PREFER</option>
                            <option value={TitleMatchEffect.Avoid}>AVOID</option>
                          </select>
                          <div className="torrent-rule__clause-actions">
                            <button
                              type="button"
                              className="ghost-button ghost-button--icon"
                              onClick={() =>
                                setDraft((current) => {
                                  if (!current || clauseIndex === 0) return current;
                                  const clauses = [...current.titleMatch.clauses];
                                  [clauses[clauseIndex - 1], clauses[clauseIndex]] = [
                                    clauses[clauseIndex],
                                    clauses[clauseIndex - 1]
                                  ];
                                  return { ...current, titleMatch: { clauses } };
                                })
                              }
                              disabled={clauseIndex === 0}
                              aria-label="上移"
                            >
                              <FontAwesomeIcon icon={faArrowUp} />
                            </button>
                            <button
                              type="button"
                              className="ghost-button ghost-button--icon"
                              onClick={() =>
                                setDraft((current) => {
                                  if (!current || clauseIndex === current.titleMatch.clauses.length - 1) return current;
                                  const clauses = [...current.titleMatch.clauses];
                                  [clauses[clauseIndex], clauses[clauseIndex + 1]] = [
                                    clauses[clauseIndex + 1],
                                    clauses[clauseIndex]
                                  ];
                                  return { ...current, titleMatch: { clauses } };
                                })
                              }
                              disabled={clauseIndex === draft.titleMatch.clauses.length - 1}
                              aria-label="下移"
                            >
                              <FontAwesomeIcon icon={faArrowDown} />
                            </button>
                            <button
                              type="button"
                              className="ghost-button ghost-button--icon"
                              onClick={() =>
                                setDraft((current) =>
                                  current
                                    ? {
                                        ...current,
                                        titleMatch: {
                                          clauses: current.titleMatch.clauses.filter((_, idx) => idx !== clauseIndex)
                                        }
                                      }
                                    : current
                                )
                              }
                              aria-label="删除"
                            >
                              <FontAwesomeIcon icon={faTrash} />
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </article>
              ) : null}

              <div className="settings-actions">
                <button type="button" className="ghost-button" onClick={close} disabled={ruleEditorSaving}>
                  取消
                </button>
                <button
                  type="button"
                  className="primary"
                  onClick={() => void saveRuleEditor()}
                  disabled={ruleEditorSaving}
                >
                  {ruleEditorSaving ? "保存中…" : "保存"}
                </button>
              </div>
            </div>
          </div>
        </aside>
      </div>,
      document.body
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

  if (settingsTab === "系统") {
    const deletePolicyInfo = "控制删除 Moji 任务时，是否联动删除 qBittorrent 里的对应下载项，以及是否同时删除下载文件。";

    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>系统</h3>
        </div>
        <form className="settings-form" onSubmit={(event) => void saveSystemSettings(event)}>
          <label className="settings-field">
            <FieldLabel text="删除任务策略" info={deletePolicyInfo} />
            <select
              value={systemForm.taskDeletePolicy}
              onChange={(event) => setSystemForm({ taskDeletePolicy: event.target.value })}
            >
              <option value={TaskDeletePolicy.KeepOnly}>仅删除 Moji 任务记录</option>
              <option value={TaskDeletePolicy.RemoveTorrent}>同时删除 qBittorrent 下载任务</option>
              <option value={TaskDeletePolicy.RemoveTorrentAndFiles}>同时删除 qBittorrent 下载任务和文件</option>
            </select>
          </label>
          <div className="settings-actions">
            <button type="submit" disabled={updatingSystem}>
              {updatingSystem ? "保存中..." : "保存系统设置"}
            </button>
          </div>
        </form>
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
