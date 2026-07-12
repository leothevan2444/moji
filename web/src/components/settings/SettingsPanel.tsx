import { FormEvent, useEffect, useState } from "react";
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
  faRotate,
  faTrash
} from "@fortawesome/free-solid-svg-icons";
import {
  JackettIndexersDocumentDocument,
  ClearImageCacheDocumentDocument,
  LogLevel,
  LogsDocumentDocument,
  RefreshSubscriptionStashBoxesDocumentDocument,
  SubscriptionReleaseBehavior,
  SubscriptionReleaseDateRange,
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
  type SettingsPageDocumentQuery,
  type ClearImageCacheDocumentMutation,
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
import { formatBytes } from "../../utils";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";

type RuntimeSettings = NonNullable<SettingsPageDocumentQuery["settings"]>;
type RuntimeSettingsStatus = NonNullable<SettingsPageDocumentQuery["settingsStatus"]>;
type JackettIndexer = JackettIndexersDocumentQuery["jackettIndexers"][number];

const TORRENT_SELECTION_RULE_KEYS: Record<TorrentSelectionRuleType, string> = {
  [TorrentSelectionRuleType.IndexerPreference]: "indexerPreference", [TorrentSelectionRuleType.TitleMatch]: "titleMatch",
  [TorrentSelectionRuleType.PublishDate]: "publishDate", [TorrentSelectionRuleType.TitleSimilarity]: "titleSimilarity",
  [TorrentSelectionRuleType.Seeders]: "seeders", [TorrentSelectionRuleType.Size]: "size",
  [TorrentSelectionRuleType.TorrentSingleVideo]: "singleVideo", [TorrentSelectionRuleType.TorrentFileNameMatch]: "fileNameMatch"
};

const SUBSCRIPTION_RELEASE_BEHAVIOR_KEYS: Record<SubscriptionReleaseBehavior, string> = {
  [SubscriptionReleaseBehavior.Download]: "download", [SubscriptionReleaseBehavior.Review]: "review", [SubscriptionReleaseBehavior.Block]: "block",
};

const SUBSCRIPTION_RELEASE_BEHAVIOR_OPTIONS = [
  SubscriptionReleaseBehavior.Download,
  SubscriptionReleaseBehavior.Review,
  SubscriptionReleaseBehavior.Block
] as const;

const SUBSCRIPTION_RELEASE_DATE_RANGE_KEYS: Record<SubscriptionReleaseDateRange, string> = {
  [SubscriptionReleaseDateRange.All]: "all", [SubscriptionReleaseDateRange.OneYear]: "oneYear",
  [SubscriptionReleaseDateRange.TwoYears]: "twoYears", [SubscriptionReleaseDateRange.ThreeYears]: "threeYears", [SubscriptionReleaseDateRange.FiveYears]: "fiveYears",
};

const SUBSCRIPTION_RELEASE_DATE_RANGE_OPTIONS = [
  SubscriptionReleaseDateRange.OneYear,
  SubscriptionReleaseDateRange.TwoYears,
  SubscriptionReleaseDateRange.ThreeYears,
  SubscriptionReleaseDateRange.FiveYears,
  SubscriptionReleaseDateRange.All
] as const;

function isTorrentInspectionRuleType(type: TorrentSelectionRuleType): boolean {
  return type === TorrentSelectionRuleType.TorrentSingleVideo || type === TorrentSelectionRuleType.TorrentFileNameMatch;
}

function usesRuleDirection(type: TorrentSelectionRuleType): boolean {
  return type === TorrentSelectionRuleType.PublishDate || type === TorrentSelectionRuleType.Seeders || type === TorrentSelectionRuleType.Size;
}

function getRuleDirection(rule: Pick<TorrentSelectionRuleDraft, "type" | "publishDate" | "seeders" | "size">): TorrentSelectionDirection {
  switch (rule.type) {
    case TorrentSelectionRuleType.PublishDate:
      return rule.publishDate.direction;
    case TorrentSelectionRuleType.Seeders:
      return rule.seeders.direction;
    case TorrentSelectionRuleType.Size:
      return rule.size.direction;
    default:
      return TorrentSelectionDirection.Desc;
  }
}

type AutomationForm = typeof EMPTY_AUTOMATION_FORM;
type TorrentSelectionRuleDraft = AutomationForm["torrentSelection"]["fastRules"][number];
type AutomationFormShape = AutomationForm;

function serializeTorrentSelectionRule(rule: TorrentSelectionRuleDraft) {
  const payload: {
    type: TorrentSelectionRuleType;
    enabled: boolean;
    indexerPreference?: { trackerIds: string[] };
    publishDate?: { direction: TorrentSelectionDirection };
    seeders?: { direction: TorrentSelectionDirection };
    size?: { direction: TorrentSelectionDirection };
    titleMatch?: {
      clauses: Array<{
        pattern: string;
        patternMode: TitleMatchPatternMode;
        effect: TitleMatchEffect;
      }>;
    };
    torrentFileNameMatch?: {
      clauses: Array<{
        pattern: string;
        patternMode: TitleMatchPatternMode;
        effect: TorrentFileMatchEffect;
      }>;
    };
  } = {
    type: rule.type,
    enabled: rule.enabled,
  };

  switch (rule.type) {
    case TorrentSelectionRuleType.IndexerPreference:
      payload.indexerPreference = { trackerIds: [...rule.indexerPreference.trackerIds] };
      break;
    case TorrentSelectionRuleType.TitleMatch:
      payload.titleMatch = {
        clauses: rule.titleMatch.clauses.map((clause) => ({
          pattern: clause.pattern,
          patternMode: clause.patternMode,
          effect: clause.effect
        }))
      };
      break;
    case TorrentSelectionRuleType.PublishDate:
      payload.publishDate = { direction: rule.publishDate.direction || TorrentSelectionDirection.Desc };
      break;
    case TorrentSelectionRuleType.Seeders:
      payload.seeders = { direction: rule.seeders.direction || TorrentSelectionDirection.Desc };
      break;
    case TorrentSelectionRuleType.Size:
      payload.size = { direction: rule.size.direction || TorrentSelectionDirection.Desc };
      break;
    case TorrentSelectionRuleType.TorrentFileNameMatch:
      payload.torrentFileNameMatch = {
        clauses: rule.torrentFileNameMatch.clauses.map((clause) => ({
          pattern: clause.pattern,
          patternMode: clause.patternMode,
          effect: clause.effect
        }))
      };
      break;
    default:
      break;
  }

  return payload;
}

/**
 * 把 AutomationForm 表单数据编码为 GraphQL mutation 所需的 input。
 * 列表保存与抽屉保存都通过此函数构造 payload，保证前后端协议一致。
 */
function buildAutomationSettingsPayload(form: AutomationFormShape) {
  const taskProgressSyncIntervalSeconds = Number.parseInt(form.taskProgressSyncIntervalSeconds.trim(), 10);
  const subscriptionPollIntervalHours = Number.parseInt(form.subscriptionPollIntervalHours.trim(), 10);
  const inspectionCandidateLimit = Number.parseInt(form.torrentSelection.inspectionCandidateLimit.trim(), 10);
  const maxGroupPerformerCount = Number.parseInt(form.subscriptionReleasePolicy.maxGroupPerformerCount.trim(), 10);
  return {
    taskProgressSyncIntervalSeconds: Number.isNaN(taskProgressSyncIntervalSeconds) ? 60 : taskProgressSyncIntervalSeconds,
    subscriptionPollIntervalHours: Number.isNaN(subscriptionPollIntervalHours) ? 1 : subscriptionPollIntervalHours,
    stashBoxEndpoints: form.stashBoxEndpoints,
    subscriptionReleasePolicy: {
      soloBehavior: form.subscriptionReleasePolicy.soloBehavior,
      groupBehavior: form.subscriptionReleasePolicy.groupBehavior,
      compilationBehavior: form.subscriptionReleasePolicy.compilationBehavior,
      maxGroupPerformerCount: Number.isNaN(maxGroupPerformerCount) ? 3 : maxGroupPerformerCount,
      releaseDateRange: form.subscriptionReleasePolicy.releaseDateRange
    },
    torrentSelection: {
      enabled: form.torrentSelection.enabled,
      inspectionCandidateLimit: Number.isNaN(inspectionCandidateLimit) ? 5 : inspectionCandidateLimit,
      fastRules: form.torrentSelection.fastRules.map(serializeTorrentSelectionRule),
      torrentRules: form.torrentSelection.torrentRules.map(serializeTorrentSelectionRule)
    }
  };
}

function releasePolicySummary(form: AutomationFormShape["subscriptionReleasePolicy"], t: TFunction) {
  const max = Number.parseInt(form.maxGroupPerformerCount.trim(), 10);
  const safeMax = Number.isNaN(max) ? 3 : max;
  return t("automation.policy.summary", {
    max: safeMax,
    solo: t(`automation.behaviors.${SUBSCRIPTION_RELEASE_BEHAVIOR_KEYS[form.soloBehavior]}`),
    group: t(`automation.behaviors.${SUBSCRIPTION_RELEASE_BEHAVIOR_KEYS[form.groupBehavior]}`),
    compilation: t(`automation.behaviors.${SUBSCRIPTION_RELEASE_BEHAVIOR_KEYS[form.compilationBehavior]}`),
    range: t(`automation.ranges.${SUBSCRIPTION_RELEASE_DATE_RANGE_KEYS[form.releaseDateRange]}`)
  });
}

function renderSubscriptionReleaseBehaviorOptions(t: TFunction) {
  return SUBSCRIPTION_RELEASE_BEHAVIOR_OPTIONS.map((behavior) => (
    <option key={behavior} value={behavior}>
      {t(`automation.behaviors.${SUBSCRIPTION_RELEASE_BEHAVIOR_KEYS[behavior]}`)}
    </option>
  ));
}

/**
 * 单条规则的只读摘要：按 type 反映关键配置，紧凑展示给列表页。
 */
function buildRuleSummary(rule: TorrentSelectionRuleDraft, t: TFunction): string {
  if (rule.type === TorrentSelectionRuleType.IndexerPreference) {
    const order = rule.indexerPreference.trackerIds.map((id) => id.trim()).filter(Boolean);
    return order.length === 0
      ? t("automation.rules.summary.noIndexers")
      : order.join(" > ");
  }
  if (rule.type === TorrentSelectionRuleType.TitleMatch) {
    const count = rule.titleMatch.clauses.length;
    return count === 0
      ? t("automation.rules.summary.noTitleRules")
      : t("automation.rules.summary.titleRules", { count });
  }
  if (rule.type === TorrentSelectionRuleType.PublishDate) {
    return t(rule.publishDate.direction === TorrentSelectionDirection.Asc ? "automation.rules.summary.dateAsc" : "automation.rules.summary.dateDesc");
  }
  if (rule.type === TorrentSelectionRuleType.TitleSimilarity) {
    return t("automation.rules.summary.similarity");
  }
  if (rule.type === TorrentSelectionRuleType.Seeders) {
    return t(rule.seeders.direction === TorrentSelectionDirection.Asc ? "automation.rules.summary.seedersAsc" : "automation.rules.summary.seedersDesc");
  }
  if (rule.type === TorrentSelectionRuleType.Size) {
    return t(rule.size.direction === TorrentSelectionDirection.Asc ? "automation.rules.summary.sizeAsc" : "automation.rules.summary.sizeDesc");
  }
  if (rule.type === TorrentSelectionRuleType.TorrentSingleVideo) {
    return t("automation.rules.summary.singleVideo");
  }
  if (rule.type === TorrentSelectionRuleType.TorrentFileNameMatch) {
    const count = rule.torrentFileNameMatch.clauses.length;
    const hasLock = rule.torrentFileNameMatch.clauses.some((clause) => clause.effect === TorrentFileMatchEffect.Lock);
    return count === 0
      ? t("automation.rules.summary.noFileRules")
      : t("automation.rules.summary.fileRules", { count, lock: hasLock ? " · LOCK" : "" });
  }
  return "";
}

function cloneTorrentSelectionRule(rule: TorrentSelectionRuleDraft): TorrentSelectionRuleDraft {
  return {
    ...rule,
    indexerPreference: { trackerIds: [...rule.indexerPreference.trackerIds] },
    titleMatch: { clauses: rule.titleMatch.clauses.map((clause) => ({ ...clause })) },
    publishDate: { direction: rule.publishDate.direction },
    seeders: { direction: rule.seeders.direction },
    size: { direction: rule.size.direction },
    torrentFileNameMatch: { clauses: rule.torrentFileNameMatch.clauses.map((clause) => ({ ...clause })) }
  };
}

function syncIndexerPreferenceTrackerIds(trackerIds: string[], indexers: JackettIndexer[]): string[] {
  const enabledIds = indexers.map((indexer) => indexer.id);
  const kept = trackerIds.filter((id) => enabledIds.includes(id));
  const missing = enabledIds.filter((id) => !kept.includes(id));
  return [...kept, ...missing];
}

function mapRuntimeTorrentSelectionRule(rule: RuntimeSettings["automation"]["torrentSelection"]["fastRules"][number]): TorrentSelectionRuleDraft {
  return {
    type: rule.type,
    enabled: rule.enabled,
    indexerPreference: {
      trackerIds: [...rule.indexerPreference.trackerIds]
    },
    titleMatch: {
      clauses: rule.titleMatch.clauses.map((clause) => ({
        pattern: clause.pattern,
        patternMode: clause.patternMode,
        effect: clause.effect
      }))
    },
    publishDate: {
      direction: rule.publishDate.direction
    },
    seeders: {
      direction: rule.seeders.direction
    },
    size: {
      direction: rule.size.direction
    },
    torrentFileNameMatch: {
      clauses: rule.torrentFileNameMatch.clauses.map((clause) => ({
        pattern: clause.pattern,
        patternMode: clause.patternMode,
        effect: clause.effect
      }))
    }
  };
}

function torrentSelectionFromRuntime(runtimeSettings: RuntimeSettings) {
  const sourceFastRules = runtimeSettings.automation.torrentSelection.fastRules;
  const sourceTorrentRules = runtimeSettings.automation.torrentSelection.torrentRules;
  const fastRules = sourceFastRules.length > 0
    ? sourceFastRules.map(mapRuntimeTorrentSelectionRule)
    : DEFAULT_TORRENT_SELECTION_RULES.map((rule) => cloneTorrentSelectionRule(rule));
  const torrentRules = sourceTorrentRules.length > 0
    ? sourceTorrentRules.map(mapRuntimeTorrentSelectionRule)
    : DEFAULT_TORRENT_FILE_INSPECTION_RULES.map((rule) => cloneTorrentSelectionRule(rule));

  return {
    enabled: runtimeSettings.automation.torrentSelection.enabled,
    inspectionCandidateLimit: String(runtimeSettings.automation.torrentSelection.inspectionCandidateLimit || 5),
    fastRules,
    torrentRules
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
  const { t } = useTranslation();
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
  const [draggedIndexerId, setDraggedIndexerId] = useState<string | null>(null);
  const [dragOverIndexerId, setDragOverIndexerId] = useState<string | null>(null);
  const [systemForm, setSystemForm] = useState(EMPTY_SYSTEM_FORM);
  const [confirmingImageCacheClear, setConfirmingImageCacheClear] = useState(false);
  const [imageCacheClearError, setImageCacheClearError] = useState<string | null>(null);
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
    pause: settingsTab !== "logs" || (drawer !== "settings" && renderedDrawer !== "settings")
  });
  const logs = logsData?.logs ?? [];

  const [{ data: jackettIndexersData, fetching: fetchingJackettIndexers }] = useQuery<
    JackettIndexersDocumentQuery,
    JackettIndexersDocumentQueryVariables
  >({
    query: JackettIndexersDocumentDocument,
    pause: (settingsTab !== "connections" && settingsTab !== "automation") || (drawer !== "settings" && renderedDrawer !== "settings")
  });
  const jackettIndexers = (jackettIndexersData?.jackettIndexers ?? []).filter((indexer: JackettIndexer) => indexer.enabled);

  useEffect(() => {
    if (jackettIndexers.length === 0) return;
    setAutomationForm((current) => {
      let changed = false;
      const fastRules = current.torrentSelection.fastRules.map((rule) => {
        if (rule.type !== TorrentSelectionRuleType.IndexerPreference) return rule;
        const trackerIds = syncIndexerPreferenceTrackerIds(rule.indexerPreference.trackerIds, jackettIndexers);
        if (trackerIds.length === rule.indexerPreference.trackerIds.length && trackerIds.every((id, index) => id === rule.indexerPreference.trackerIds[index])) {
          return rule;
        }
        changed = true;
        return {
          ...rule,
          indexerPreference: { trackerIds }
        };
      });
      if (!changed) return current;
      return {
        ...current,
        torrentSelection: {
          ...current.torrentSelection,
          fastRules
        }
      };
    });
  }, [jackettIndexers]);

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
  const [{ fetching: clearingImageCache }, clearImageCache] = useMutation<ClearImageCacheDocumentMutation>(ClearImageCacheDocumentDocument);
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
      subscriptionReleasePolicy: {
        soloBehavior: runtimeSettings.automation.subscriptionReleasePolicy.soloBehavior,
        groupBehavior: runtimeSettings.automation.subscriptionReleasePolicy.groupBehavior,
        compilationBehavior: runtimeSettings.automation.subscriptionReleasePolicy.compilationBehavior,
        maxGroupPerformerCount: String(runtimeSettings.automation.subscriptionReleasePolicy.maxGroupPerformerCount || 3),
        releaseDateRange: runtimeSettings.automation.subscriptionReleasePolicy.releaseDateRange
      },
      torrentSelection: torrentSelectionFromRuntime(runtimeSettings)
    });
    setSystemForm({
      taskDeletePolicy: runtimeSettings.system.taskDeletePolicy || TaskDeletePolicy.KeepOnly,
      imageCacheEnabled: runtimeSettings.system.imageCache.enabled,
      imageCacheMaxSizeMb: String(runtimeSettings.system.imageCache.maxSizeMb),
      imageCacheRetentionDays: String(runtimeSettings.system.imageCache.retentionDays)
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
    pushToast("tone-success", t("settings.connections.saved", { service: "Stash" }));
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
    pushToast("tone-success", t("settings.ingest.saved"));
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
    pushToast("tone-success", t("settings.connections.saved", { service: "Jackett" }));
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
    pushToast("tone-success", t("settings.connections.saved", { service: "qBittorrent" }));
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveAutomationSettingsSection = async (successMessage: string) => {
    const payload = buildAutomationSettingsPayload(automationForm);
    const result = await updateAutomationSettings({ input: payload });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", successMessage);
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const saveAutomationSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await saveAutomationSettingsSection(t("automationUi.saved"));
  };

  const saveTorrentSelectionSettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await saveAutomationSettingsSection(t("ruleUi.saved"));
  };

  const saveStashBoxPrioritySettings = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await saveAutomationSettingsSection(t("automationUi.stashBoxes.saved"));
  };

  const moveCandidateRuleInSection = (section: "fast" | "file", from: number, to: number) => {
    setAutomationForm((current) => {
      const sourceRules = section === "fast" ? current.torrentSelection.fastRules : current.torrentSelection.torrentRules;
      if (to < 0 || to >= sourceRules.length) return current;
      const reordered = [...sourceRules];
      const [moved] = reordered.splice(from, 1);
      reordered.splice(to, 0, moved);
      return {
        ...current,
        torrentSelection: {
          ...current.torrentSelection,
          fastRules: section === "fast" ? reordered : current.torrentSelection.fastRules,
          torrentRules: section === "file" ? reordered : current.torrentSelection.torrentRules
        }
      };
    });
  };

  const updateCandidateRule = (ruleType: TorrentSelectionRuleType, updater: (rule: TorrentSelectionRuleDraft) => TorrentSelectionRuleDraft) => {
    setAutomationForm((current) => ({
      ...current,
      torrentSelection: {
        ...current.torrentSelection,
        fastRules: current.torrentSelection.fastRules.map((rule) => (rule.type === ruleType ? updater(rule) : rule)),
        torrentRules: current.torrentSelection.torrentRules.map((rule) => (rule.type === ruleType ? updater(rule) : rule))
      }
    }));
  };

  const addTitleMatchClause = (ruleType: TorrentSelectionRuleType) => {
    updateCandidateRule(ruleType, (rule) => ({
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
    ruleType: TorrentSelectionRuleType,
    clauseIndex: number,
    updater: (clause: TorrentSelectionRuleDraft["titleMatch"]["clauses"][number]) => TorrentSelectionRuleDraft["titleMatch"]["clauses"][number]
  ) => {
    updateCandidateRule(ruleType, (rule) => ({
      ...rule,
      titleMatch: {
        clauses: rule.titleMatch.clauses.map((clause, index) => (index === clauseIndex ? updater(clause) : clause))
      }
    }));
  };

  const moveTitleMatchClause = (ruleType: TorrentSelectionRuleType, from: number, to: number) => {
    updateCandidateRule(ruleType, (rule) => {
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

  const removeTitleMatchClause = (ruleType: TorrentSelectionRuleType, clauseIndex: number) => {
    updateCandidateRule(ruleType, (rule) => ({
      ...rule,
      titleMatch: {
        clauses: rule.titleMatch.clauses.filter((_, index) => index !== clauseIndex)
      }
    }));
  };

  const addTorrentFileNameMatchClause = (ruleType: TorrentSelectionRuleType) => {
    updateCandidateRule(ruleType, (rule) => ({
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
    ruleType: TorrentSelectionRuleType,
    clauseIndex: number,
    updater: (clause: TorrentSelectionRuleDraft["torrentFileNameMatch"]["clauses"][number]) => TorrentSelectionRuleDraft["torrentFileNameMatch"]["clauses"][number]
  ) => {
    updateCandidateRule(ruleType, (rule) => ({
      ...rule,
      torrentFileNameMatch: {
        clauses: rule.torrentFileNameMatch.clauses.map((clause, index) => (index === clauseIndex ? updater(clause) : clause))
      }
    }));
  };

  const removeTorrentFileNameMatchClause = (ruleType: TorrentSelectionRuleType, clauseIndex: number) => {
    updateCandidateRule(ruleType, (rule) => ({
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
        taskDeletePolicy: systemForm.taskDeletePolicy as TaskDeletePolicy,
        imageCache: { enabled: systemForm.imageCacheEnabled, maxSizeMb: Number(systemForm.imageCacheMaxSizeMb), retentionDays: Number(systemForm.imageCacheRetentionDays) }
      }
    });
    if (result.error) {
      pushToast("tone-danger", describeQueryError(result.error));
      return;
    }
    pushToast("tone-success", "系统设置已保存。");
    await refreshDashboard({ requestPolicy: "network-only" });
  };

  const handleClearImageCache = async () => {
    const releasedBytes = Number(runtimeStatus?.imageCache.usedBytes ?? 0);
    setImageCacheClearError(null);
    const result = await clearImageCache({});
    if (result.error) {
      const message = describeQueryError(result.error);
      setImageCacheClearError(message);
      pushToast("tone-danger", message);
      return;
    }
    setConfirmingImageCacheClear(false);
    pushToast("tone-success", releasedBytes > 0 ? `图片缓存已清空，释放 ${formatBytes(releasedBytes)}。` : "图片缓存已清空。");
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

  const fastRules = automationForm.torrentSelection.fastRules;
  const fileInspectionRules = automationForm.torrentSelection.torrentRules;
  const totalRuleCount = fastRules.length + fileInspectionRules.length;

  if (!runtimeSettings || !runtimeStatus) {
    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>{t(`settings.tabs.${settingsTab}`)}</h3>
        </div>
        <dl className="settings-grid">
          <div>
            <dt>{t("settings.state")}</dt>
            <dd>{t("settings.waiting")}</dd>
          </div>
        </dl>
      </article>
    );
  }

  if (settingsTab === "connections") {
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
          <h3>{t("settings.tabs.connections")}</h3>
        </div>

        <form className="settings-form" onSubmit={(event) => void saveStashSettings(event)}>
          <div className="settings-meta">
            <span>Stash</span>
            <span className={`status-chip ${stashStatus.tone}`}>{t(stashStatus.labelKey)}</span>
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
              {updatingStash ? t("settings.saving") : t("settings.connections.save", { service: "Stash" })}
            </button>
          </div>
        </form>

        <form className="settings-form" onSubmit={(event) => void saveJackettSettings(event)}>
          <div className="settings-meta">
            <span>Jackett</span>
            <span className={`status-chip ${jackettStatus.tone}`}>{t(jackettStatus.labelKey)}</span>
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
              {t("settings.connections.dashboardPassword")}
            </span>
            <div className="secret-input">
              <input
                className="secret-input__field"
                type={visibleSecrets.jackettPassword ? "text" : "password"}
                value={jackettForm.password}
                onChange={(event) => setJackettForm((current) => ({ ...current, password: event.target.value }))}
                placeholder={t("settings.connections.dashboardPasswordPlaceholder")}
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
              {updatingJackett ? t("settings.saving") : t("settings.connections.save", { service: "Jackett" })}
            </button>
          </div>
        </form>

        <form className="settings-form" onSubmit={(event) => void saveQBittorrentSettings(event)}>
          <div className="settings-meta">
            <span>qBittorrent</span>
            <span className={`status-chip ${qbittorrentStatus.tone}`}>{t(qbittorrentStatus.labelKey)}</span>
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
            <span>{t("settings.connections.username")}</span>
            <input
              value={qbittorrentForm.username}
              onChange={(event) => setQBittorrentForm((current) => ({ ...current, username: event.target.value }))}
              placeholder="admin"
            />
          </label>
          <label className="settings-field">
            <span>{t("settings.connections.password")}</span>
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
            <span>{t("settings.connections.defaultSavePath")}</span>
            <input
              value={qbittorrentForm.defaultSavePath}
              onChange={(event) => setQBittorrentForm((current) => ({ ...current, defaultSavePath: event.target.value }))}
              placeholder="/downloads"
            />
          </label>
          <label className="settings-field">
            <span>{t("settings.connections.defaultCategory")}</span>
            <input
              value={qbittorrentForm.category}
              onChange={(event) => setQBittorrentForm((current) => ({ ...current, category: event.target.value }))}
              placeholder="moji"
            />
          </label>
          <label className="settings-field">
            <span>{t("settings.connections.defaultTags")}</span>
            <input
              value={qbittorrentForm.tags}
              onChange={(event) => setQBittorrentForm((current) => ({ ...current, tags: event.target.value }))}
              placeholder="auto"
            />
          </label>
          <div className="settings-actions">
            <button type="submit" disabled={updatingQBittorrent}>
              {updatingQBittorrent ? t("settings.saving") : t("settings.connections.save", { service: "qBittorrent" })}
            </button>
          </div>
        </form>
      </article>
    );
  }

  if (settingsTab === "ingest") {
    const stashLibraries = runtimeStatus.stashLibraries ?? [];
    const stashLibrariesLoadError = runtimeStatus.stashLibrariesLoadError ?? null;
    const qbDefaultSavePath = qbittorrentForm.defaultSavePath.trim();
    const deliveryModeInfo = ingestForm.deliveryMode === "PATH_MAP"
      ? t("settings.ingest.pathMapInfo") : t("settings.ingest.transferInfo");
    const qbRootInfo = t("settings.ingest.qbRootInfo");
    const mojiDownloadsRootInfo = t("settings.ingest.mojiDownloadInfo");
    const mojiLibraryRootInfo = t("settings.ingest.mojiLibraryInfo");
    const stashLibraryInfo = t("settings.ingest.stashInfo");
    const transferActionInfo = t("settings.ingest.actionInfo");

    return (
      <article className="drawer-card">
        <div className="drawer-card__head">
          <h3>{t("settings.tabs.ingest")}</h3>
        </div>
        <form className="settings-form" onSubmit={(event) => void saveIngestSettings(event)}>
          <label className="settings-field">
            <FieldLabel text={t("settings.ingest.mode")} info={deliveryModeInfo} />
            <select
              value={ingestForm.deliveryMode}
              onChange={(event) => setIngestForm((current) => ({
                ...current,
                deliveryMode: event.target.value
              }))}
            >
              <option value="PATH_MAP">{t("settings.ingest.pathMap")}</option>
              <option value="TRANSFER">{t("settings.ingest.transfer")}</option>
            </select>
          </label>
          <label className="settings-field">
            <FieldLabel text={t("settings.ingest.qbRoot")} info={qbRootInfo} />
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
                aria-label={t("settings.ingest.useQbDefault")}
                title={qbDefaultSavePath === "" ? t("settings.ingest.noQbDefault") : t("settings.ingest.initializeWith", { path: qbDefaultSavePath })}
              >
                <FontAwesomeIcon icon={faRotate} aria-hidden="true" />
              </button>
            </div>
          </label>
          <label className="settings-field">
            <FieldLabel text={t("settings.ingest.stashRoot")} info={stashLibraryInfo} />
            <select
              value={ingestForm.library.stashRoot}
              onChange={(event) => setIngestForm((current) => ({
                ...current,
                library: { ...current.library, stashRoot: event.target.value }
              }))}
            >
              <option value="" disabled hidden>{stashLibraries.length > 0 ? t("settings.ingest.selectStashRoot") : t("settings.ingest.noStashRoot")}</option>
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
                <FieldLabel text={t("settings.ingest.action")} info={transferActionInfo} />
                <select
                  value={ingestForm.transfer.action}
                  onChange={(event) => setIngestForm((current) => ({
                    ...current,
                    transfer: { ...current.transfer, action: event.target.value }
                  }))}
                >
                  <option value="COPY">{t("settings.ingest.copy")}</option>
                  <option value="MOVE">{t("settings.ingest.move")}</option>
                  <option value="SYMLINK">{t("settings.ingest.symlink")}</option>
                </select>
              </label>
              <label className="settings-field">
                <FieldLabel text={t("settings.ingest.mojiDownloadRoot")} info={mojiDownloadsRootInfo} />
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
                <FieldLabel text={t("settings.ingest.mojiLibraryRoot")} info={mojiLibraryRootInfo} />
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
              {updatingIngest ? t("settings.saving") : t("settings.ingest.save")}
            </button>
          </div>
        </form>

        {pendingIngestQBRootInitialization ? (
          <div className="drawer-scrim drawer-scrim--modal" onClick={() => setPendingIngestQBRootInitialization(null)}>
            <aside className="drawer drawer--modal" onClick={(event) => event.stopPropagation()}>
              <div className="drawer__head">
                <div>
                  <h2>{t("settings.ingest.initTitle")}</h2>
                </div>
                <button
                  type="button"
                  className="ghost-button"
                  onClick={() => setPendingIngestQBRootInitialization(null)}
                  disabled={updatingIngest}
                >
                  {t("common.close")}
                </button>
              </div>
              <div className="drawer-body">
                <div className="drawer-stack">
                  <article className="drawer-card">
                    <div className="drawer-card__head">
                      <div>
                        <h3>{t("settings.ingest.emptyTitle")}</h3>
                        <p>{t("settings.ingest.initQuestion", { path: pendingIngestQBRootInitialization })}</p>
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
                        {updatingIngest ? t("settings.saving") : t("settings.ingest.useAndSave")}
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
                        {t("settings.ingest.keepEmpty")}
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

  if (settingsTab === "automation") {
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
          <h3>{t("settings.tabs.automation")}</h3>
        </div>

        <form className="settings-form" onSubmit={(event) => void saveAutomationSettings(event)}>
          <label className="settings-field">
            <span>{t("automationUi.syncInterval")}</span>
            <input
              value={automationForm.taskProgressSyncIntervalSeconds}
              onChange={(event) => setAutomationForm((current) => ({ ...current, taskProgressSyncIntervalSeconds: event.target.value }))}
              placeholder="60"
            />
          </label>
          <label className="settings-field">
            <span>{t("automationUi.pollInterval")}</span>
            <input
              value={automationForm.subscriptionPollIntervalHours}
              onChange={(event) => setAutomationForm((current) => ({ ...current, subscriptionPollIntervalHours: event.target.value }))}
              placeholder="1"
            />
          </label>
          <div className="settings-meta">
            <span>{t("automationUi.taskSync", { state: t(runtimeStatus.automation.taskProgressSyncEnabled ? "automationUi.enabled" : "automationUi.disabled") })}</span>
            <span>{t("automationUi.poll", { state: t(runtimeStatus.automation.subscriptionPollEnabled ? "automationUi.enabled" : "automationUi.disabled") })}</span>
          </div>
          <section className="settings-section settings-section--nested">
            <header className="settings-section__head">
              <div>
                <h4>{t("automationUi.policyTitle")}</h4>
                <p className="settings-section__hint">{releasePolicySummary(automationForm.subscriptionReleasePolicy, t)}</p>
              </div>
            </header>
            <label className="settings-field">
              <FieldLabel text={t("automationUi.solo")} info={t("automationUi.soloInfo")} />
              <select
                value={automationForm.subscriptionReleasePolicy.soloBehavior}
                onChange={(event) =>
                  setAutomationForm((current) => ({
                    ...current,
                    subscriptionReleasePolicy: {
                      ...current.subscriptionReleasePolicy,
                      soloBehavior: event.target.value as SubscriptionReleaseBehavior
                    }
                  }))
                }
              >
                {renderSubscriptionReleaseBehaviorOptions(t)}
              </select>
            </label>
            <label className="settings-field">
              <FieldLabel text={t("automationUi.group")} info={t("automationUi.groupInfo")} />
              <select
                value={automationForm.subscriptionReleasePolicy.groupBehavior}
                onChange={(event) =>
                  setAutomationForm((current) => ({
                    ...current,
                    subscriptionReleasePolicy: {
                      ...current.subscriptionReleasePolicy,
                      groupBehavior: event.target.value as SubscriptionReleaseBehavior
                    }
                  }))
                }
              >
                {renderSubscriptionReleaseBehaviorOptions(t)}
              </select>
            </label>
            <label className="settings-field">
              <FieldLabel text={t("automationUi.compilation")} info={t("automationUi.compilationInfo")} />
              <select
                value={automationForm.subscriptionReleasePolicy.compilationBehavior}
                onChange={(event) =>
                  setAutomationForm((current) => ({
                    ...current,
                    subscriptionReleasePolicy: {
                      ...current.subscriptionReleasePolicy,
                      compilationBehavior: event.target.value as SubscriptionReleaseBehavior
                    }
                  }))
                }
              >
                {renderSubscriptionReleaseBehaviorOptions(t)}
              </select>
            </label>
            <label className="settings-field">
              <FieldLabel text={t("automationUi.groupMax")} info={t("automationUi.groupMaxInfo")} />
              <input
                value={automationForm.subscriptionReleasePolicy.maxGroupPerformerCount}
                onChange={(event) =>
                  setAutomationForm((current) => ({
                    ...current,
                    subscriptionReleasePolicy: {
                      ...current.subscriptionReleasePolicy,
                      maxGroupPerformerCount: event.target.value
                    }
                  }))
                }
                placeholder="3"
              />
            </label>
            <label className="settings-field">
              <FieldLabel text={t("automationUi.dateRange")} info={t("automationUi.dateRangeInfo")} />
              <select
                value={automationForm.subscriptionReleasePolicy.releaseDateRange}
                onChange={(event) =>
                  setAutomationForm((current) => ({
                    ...current,
                    subscriptionReleasePolicy: {
                      ...current.subscriptionReleasePolicy,
                      releaseDateRange: event.target.value as SubscriptionReleaseDateRange
                    }
                  }))
                }
              >
                {SUBSCRIPTION_RELEASE_DATE_RANGE_OPTIONS.map((dateRange) => (
                  <option key={dateRange} value={dateRange}>
                    {t(`automation.ranges.${SUBSCRIPTION_RELEASE_DATE_RANGE_KEYS[dateRange]}`)}
                  </option>
                ))}
              </select>
            </label>
          </section>
          <div className="settings-actions">
            <button type="submit" disabled={updatingAutomation}>
              {updatingAutomation ? t("settings.saving") : t("automationUi.save")}
            </button>
          </div>
        </form>
<div className="settings-spacer" />
        <form className="settings-form" onSubmit={(event) => void saveStashBoxPrioritySettings(event)}>
          <section className="stashbox-source">
            <header className="stashbox-source__head">
              <div>
                <h4>{t("automationUi.stashBoxes.title")}</h4>
                <p className="stashbox-source__sub">
                  {t("automationUi.stashBoxes.detail")}
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
                  <span>{refreshingStashBoxes ? t("automationUi.stashBoxes.refreshing") : t("automationUi.stashBoxes.refresh")}</span>
                </button>
                <button type="submit" disabled={updatingAutomation}>
                  {updatingAutomation ? t("settings.saving") : t("automationUi.stashBoxes.save")}
                </button>
              </div>
            </header>

            {!loaded ? (
              <div className="stashbox-source__empty stashbox-source__empty--loading">
                <div className="stashbox-source__spinner" aria-hidden="true" />
                <div>
                  <strong>{t("automationUi.stashBoxes.loading")}</strong>
                  <p>{t("automationUi.stashBoxes.loadingDetail")}</p>
                </div>
              </div>
            ) : display.length === 0 ? (
              <div className="stashbox-source__empty stashbox-source__empty--danger">
                <div className="stashbox-source__icon" aria-hidden="true">!</div>
                <div>
                  <strong>{t("automationUi.stashBoxes.empty")}</strong>
                  <p>{t("automationUi.stashBoxes.emptyDetail")}</p>
                  {loadError ? <p className="stashbox-source__error">{t("automationUi.stashBoxes.loadFailed", { error: loadError })}</p> : null}
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
                      <span className="stashbox-card__handle" aria-hidden="true" title={t("automationUi.stashBoxes.drag")}>
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
                            {t(item.box?.apiKeyConfigured ? "automationUi.stashBoxes.keyReady" : "automationUi.stashBoxes.keyMissing")}
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
                          aria-label={t("automationUi.stashBoxes.up")}
                        >
                          <FontAwesomeIcon icon={faArrowUp} />
                        </button>
                        <button
                          type="button"
                          className="ghost-button ghost-button--icon"
                          disabled={index === display.length - 1}
                          onClick={() => move(index, index + 1)}
                          aria-label={t("automationUi.stashBoxes.down")}
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
<div className="settings-spacer" />

        <form className="settings-form" onSubmit={(event) => void saveTorrentSelectionSettings(event)}>
          <section className="torrent-rules">
            <header className="torrent-rules__head">
              <div>
                <h4>{t("ruleUi.title")}</h4>
                <p className="torrent-rules__sub">
                  {t("ruleUi.detail")}
                </p>
              </div>
              <div className="torrent-rules__save">
                <button type="submit" disabled={updatingAutomation}>
                  {updatingAutomation ? t("settings.saving") : t("ruleUi.save")}
                </button>
              </div>
            </header>

            <div className="torrent-rules__toolbar">
              <label className="switch-row">
                <span className="switch-row__label">{t("ruleUi.enableChain")}</span>
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
                <span>{t(rulesExpanded ? "common.collapse" : "common.expand")}</span>
              </button>
            </div>

            {!rulesExpanded && totalRuleCount > 0 ? (
              <p className="torrent-rules__hint">
                {t("ruleUi.configured", { count: totalRuleCount })}
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
                    key={rule.type}
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
                      moveCandidateRuleInSection("fast", Number.isNaN(from) ? draggedIndex ?? ruleIndex : from, ruleIndex);
                      setDraggedIndex(null);
                      setDragOverIndex(null);
                    }}
                    onDragEnd={() => {
                      setDraggedIndex(null);
                      setDragOverIndex(null);
                    }}
                  >
                    <header className="torrent-rule__head">
                    <span className="torrent-rule__handle" aria-hidden="true" title={t("ruleUi.drag")}>
                      <FontAwesomeIcon icon={faGripVertical} />
                    </span>
                    <span className="torrent-rule__order">{ruleIndex + 1}</span>
                    <h3 className="torrent-rule__name">{t(`automation.rules.names.${TORRENT_SELECTION_RULE_KEYS[rule.type]}`)}</h3>
                    <div className="torrent-rule__inline-readonly" aria-hidden="true">
                      <span className="torrent-rule__badge torrent-rule__badge--type">
                        {t(`automation.rules.names.${TORRENT_SELECTION_RULE_KEYS[rule.type]}`)}
                      </span>
                      {usesRuleDirection(rule.type) ? (
                        <span className="torrent-rule__badge torrent-rule__badge--dir">
                          {getRuleDirection(rule) === TorrentSelectionDirection.Asc ? "ASC" : "DESC"}
                        </span>
                      ) : null}
                    </div>
                    <div className="torrent-rule__actions">
                      <label className="switch switch--sm" role="switch" aria-checked={rule.enabled} aria-label={t("automation.rules.enableLabel", { rule: t(`automation.rules.names.${TORRENT_SELECTION_RULE_KEYS[rule.type]}`) })}>
                        <input
                          type="checkbox"
                          checked={rule.enabled}
                          onChange={(event) =>
                            updateCandidateRule(rule.type, (current) => ({
                              ...current,
                              enabled: event.target.checked
                            }))
                          }
                        />
                        <span className="switch__track" aria-hidden="true" />
                        <span className="switch__thumb" aria-hidden="true" />
                      </label>
                      <button type="button" className="ghost-button ghost-button--icon" onClick={() => moveCandidateRuleInSection("fast", ruleIndex, ruleIndex - 1)} disabled={ruleIndex === 0} aria-label={t("common.moveUp")}>
                        <FontAwesomeIcon icon={faArrowUp} />
                      </button>
                      <button
                        type="button"
                        className="ghost-button ghost-button--icon"
                        onClick={() => moveCandidateRuleInSection("fast", ruleIndex, ruleIndex + 1)}
                        disabled={ruleIndex === fastRules.length - 1}
                        aria-label={t("common.moveDown")}
                      >
                        <FontAwesomeIcon icon={faArrowDown} />
                      </button>
                    </div>
                    </header>

                    {rule.enabled ? (
                      <>
                        <p className="torrent-rule__summary">{buildRuleSummary(rule, t)}</p>
                        <div className="torrent-rule__body">
                          {usesRuleDirection(rule.type) ? (
                            <label className="torrent-rule__inline-field">
                              <span className="torrent-rule__inline-label">{t("ruleUi.direction")}</span>
                              <select
                                value={getRuleDirection(rule)}
                                onChange={(event) =>
                                  updateCandidateRule(rule.type, (current) => {
                                    const nextDirection = event.target.value as TorrentSelectionDirection;
                                    if (current.type === TorrentSelectionRuleType.PublishDate) {
                                      return { ...current, publishDate: { direction: nextDirection } };
                                    }
                                    if (current.type === TorrentSelectionRuleType.Seeders) {
                                      return { ...current, seeders: { direction: nextDirection } };
                                    }
                                    if (current.type === TorrentSelectionRuleType.Size) {
                                      return { ...current, size: { direction: nextDirection } };
                                    }
                                    return current;
                                  })
                                }
                              >
                                <option value={TorrentSelectionDirection.Desc}>DESC</option>
                                <option value={TorrentSelectionDirection.Asc}>ASC</option>
                              </select>
                            </label>
                          ) : null}

                          {rule.type === TorrentSelectionRuleType.TitleSimilarity ? (
                            <p className="torrent-rule__hint">{t("ruleUi.similarityHint")}</p>
                          ) : null}

                          {rule.type === TorrentSelectionRuleType.IndexerPreference ? (
                            <>
                              {fetchingJackettIndexers ? <span className="torrent-rule__hint">{t("ruleUi.loadingIndexers")}</span> : null}
                              {!fetchingJackettIndexers && jackettIndexers.length === 0 ? (
                                <span className="torrent-rule__hint">{t("ruleUi.noIndexers")}</span>
                              ) : null}
                              {!fetchingJackettIndexers && jackettIndexers.length > 0 ? (
                                <ol className="torrent-rule__indexer-list">
                                  {rule.indexerPreference.trackerIds.map((trackerId, selectedIndex) => {
                                    const indexer = jackettIndexers.find((item: JackettIndexer) => item.id === trackerId);
                                    if (!indexer) return null;
                                    const classes = ["torrent-rule__indexer-card"];
                                    if (draggedIndexerId === indexer.id) classes.push("is-dragging");
                                    if (dragOverIndexerId === indexer.id && draggedIndexerId !== indexer.id) classes.push("is-drop-target");
                                    return (
                                      <li
                                        key={indexer.id}
                                        className={classes.join(" ")}
                                        draggable
                                        onDragStart={(event) => {
                                          event.dataTransfer.effectAllowed = "move";
                                          event.dataTransfer.setData("text/plain", indexer.id);
                                          setDraggedIndexerId(indexer.id);
                                        }}
                                        onDragOver={(event) => {
                                          if (!draggedIndexerId || draggedIndexerId === indexer.id) return;
                                          event.preventDefault();
                                          event.dataTransfer.dropEffect = "move";
                                          setDragOverIndexerId(indexer.id);
                                        }}
                                        onDragLeave={() => {
                                          if (dragOverIndexerId === indexer.id) setDragOverIndexerId(null);
                                        }}
                                        onDrop={(event) => {
                                          event.preventDefault();
                                          const fromId = event.dataTransfer.getData("text/plain") || draggedIndexerId;
                                          if (!fromId || fromId === indexer.id) return;
                                          updateCandidateRule(rule.type, (current) => {
                                            const trackerIds = [...current.indexerPreference.trackerIds];
                                            const from = trackerIds.indexOf(fromId);
                                            const to = trackerIds.indexOf(indexer.id);
                                            if (from < 0 || to < 0) return current;
                                            const [moved] = trackerIds.splice(from, 1);
                                            trackerIds.splice(to, 0, moved);
                                            return {
                                              ...current,
                                              indexerPreference: { trackerIds }
                                            };
                                          });
                                          setDraggedIndexerId(null);
                                          setDragOverIndexerId(null);
                                        }}
                                        onDragEnd={() => {
                                          setDraggedIndexerId(null);
                                          setDragOverIndexerId(null);
                                        }}
                                      >
                                        <div className="torrent-rule__indexer-main">
                                          <span className="torrent-rule__indexer-handle" aria-hidden="true" title={t("ruleUi.dragPriority")}>
                                            <FontAwesomeIcon icon={faGripVertical} />
                                          </span>
                                          <div className="torrent-rule__indexer-copy">
                                            <strong title={indexer.name}>{indexer.name}</strong>
                                          </div>
                                        </div>
                                        <div className="torrent-rule__indexer-meta">
                                          <span className="torrent-rule__indexer-rank">#{selectedIndex + 1}</span>
                                        </div>
                                      </li>
                                    );
                                  })}
                                </ol>
                              ) : null}
                            </>
                          ) : null}

                          {rule.type === TorrentSelectionRuleType.TitleMatch ? (
                            <>
                              <div className="torrent-rule__section-head">
                                <div>
                                  <p className="torrent-rule__hint">{t("ruleUi.titleHint")}</p>
                                </div>
                                <button
                                  type="button"
                                  className="ghost-button"
                                  onClick={() => addTitleMatchClause(rule.type)}
                                >
                                  {t("common.add")}
                                </button>
                              </div>
                              {rule.titleMatch.clauses.length === 0 ? (
                                <p className="torrent-rule__hint">{t("ruleUi.noTitleRules")}</p>
                              ) : (
                                <div className="torrent-rule__clauses">
                                  {rule.titleMatch.clauses.map((clause, clauseIndex) => (
                                    <div key={`${rule.type}-clause-${clauseIndex}`} className="torrent-rule__clause">
                                      <input
                                        className="torrent-rule__clause-pattern"
                                        value={clause.pattern}
                                        onChange={(event) =>
                                          updateTitleMatchClause(rule.type, clauseIndex, (current) => ({
                                            ...current,
                                            pattern: event.target.value
                                          }))
                                        }
                                        placeholder="uncensored or /\\b4k\\b/i"
                                        aria-label={t("ruleUi.titlePattern")}
                                      />
                                      <select
                                        className="torrent-rule__clause-mode"
                                        value={clause.patternMode}
                                        onChange={(event) =>
                                          updateTitleMatchClause(rule.type, clauseIndex, (current) => ({
                                            ...current,
                                            patternMode: event.target.value as TitleMatchPatternMode
                                          }))
                                        }
                                        aria-label={t("ruleUi.patternMode")}
                                      >
                                        <option value={TitleMatchPatternMode.Plain}>PLAIN</option>
                                        <option value={TitleMatchPatternMode.Regex}>REGEX</option>
                                      </select>
                                      <select
                                        className="torrent-rule__clause-effect"
                                        value={clause.effect}
                                        onChange={(event) =>
                                          updateTitleMatchClause(rule.type, clauseIndex, (current) => ({
                                            ...current,
                                            effect: event.target.value as TitleMatchEffect
                                          }))
                                        }
                                        aria-label={t("ruleUi.effect")}
                                      >
                                        <option value={TitleMatchEffect.Prefer}>PREFER</option>
                                        <option value={TitleMatchEffect.Avoid}>AVOID</option>
                                      </select>
                                      <div className="torrent-rule__clause-actions">
                                        <button
                                          type="button"
                                          className="ghost-button ghost-button--icon"
                                          onClick={() => moveTitleMatchClause(rule.type, clauseIndex, clauseIndex - 1)}
                                          disabled={clauseIndex === 0}
                                          aria-label={t("common.moveUp")}
                                        >
                                          <FontAwesomeIcon icon={faArrowUp} />
                                        </button>
                                        <button
                                          type="button"
                                          className="ghost-button ghost-button--icon"
                                          onClick={() => moveTitleMatchClause(rule.type, clauseIndex, clauseIndex + 1)}
                                          disabled={clauseIndex === rule.titleMatch.clauses.length - 1}
                                          aria-label={t("common.moveDown")}
                                        >
                                          <FontAwesomeIcon icon={faArrowDown} />
                                        </button>
                                        <button
                                          type="button"
                                          className="ghost-button ghost-button--icon"
                                          onClick={() => removeTitleMatchClause(rule.type, clauseIndex)}
                                          aria-label={t("common.delete")}
                                        >
                                          <FontAwesomeIcon icon={faTrash} />
                                        </button>
                                      </div>
                                    </div>
                                  ))}
                                </div>
                              )}
                            </>
                          ) : null}
                        </div>
                      </>
                    ) : null}
                </article>
                );
              })}

              <div className="torrent-rules__divider" aria-hidden="true" />
              <div className="torrent-rules__inspection-row">
                <p className="torrent-rules__note">
                  {t("ruleUi.inspectionIntro", { count: automationForm.torrentSelection.inspectionCandidateLimit || "5" })}
                </p>
                <label className="torrent-rules__limit-inline">
                  <FieldLabel text={t("ruleUi.inspectionScope")} info={t("ruleUi.inspectionInfo")} />
                  <input
                    type="number"
                    min="1"
                    step="1"
                    value={automationForm.torrentSelection.inspectionCandidateLimit}
                    onChange={(event) =>
                      setAutomationForm((current) => ({
                        ...current,
                        torrentSelection: {
                          ...current.torrentSelection,
                          inspectionCandidateLimit: event.target.value
                        }
                      }))
                    }
                    placeholder="5"
                  />
                </label>
              </div>

              {fileInspectionRules.map((rule: TorrentSelectionRuleDraft, ruleIndex: number) => {
                const displayIndex = fastRules.length + ruleIndex + 1;
                return (
                  <article key={rule.type} className={`torrent-rule${rule.enabled ? "" : " is-disabled"}`}>
                    <header className="torrent-rule__head">
                      <span className="torrent-rule__order">{displayIndex}</span>
                      <h3 className="torrent-rule__name">{t(`automation.rules.names.${TORRENT_SELECTION_RULE_KEYS[rule.type]}`)}</h3>
                      <div className="torrent-rule__inline-readonly" aria-hidden="true">
                        <span className="torrent-rule__badge torrent-rule__badge--type">
                          {t(`automation.rules.names.${TORRENT_SELECTION_RULE_KEYS[rule.type]}`)}
                        </span>
                        {usesRuleDirection(rule.type) ? (
                          <span className="torrent-rule__badge torrent-rule__badge--dir">
                            {getRuleDirection(rule) === TorrentSelectionDirection.Asc ? "ASC" : "DESC"}
                          </span>
                        ) : null}
                      </div>
                      <div className="torrent-rule__actions">
                        <label className="switch switch--sm" role="switch" aria-checked={rule.enabled} aria-label={t("automation.rules.enableLabel", { rule: t(`automation.rules.names.${TORRENT_SELECTION_RULE_KEYS[rule.type]}`) })}>
                          <input
                            type="checkbox"
                            checked={rule.enabled}
                            onChange={(event) =>
                              updateCandidateRule(rule.type, (current) => ({
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

                    {rule.enabled ? (
                      <>
                        <p className="torrent-rule__summary">{buildRuleSummary(rule, t)}</p>
                        <div className="settings-form">
                          {rule.type === TorrentSelectionRuleType.TorrentSingleVideo ? (
                            <p className="torrent-rule__hint">
                              {t("ruleUi.singleVideoHint", { count: automationForm.torrentSelection.inspectionCandidateLimit || "5" })}
                            </p>
                          ) : null}

                          {rule.type === TorrentSelectionRuleType.TorrentFileNameMatch ? (
                            <>
                              <div className="drawer-card__head">
                                <div>
                                  <p className="torrent-rule__hint">{t("ruleUi.fileHint")}</p>
                                  <p className="torrent-rule__hint">{t("ruleUi.fileModeHint")}</p>
                                </div>
                                <button
                                  type="button"
                                  className="ghost-button"
                                  onClick={() => addTorrentFileNameMatchClause(rule.type)}
                                >
                                  {t("common.add")}
                                </button>
                              </div>
                              {rule.torrentFileNameMatch.clauses.length === 0 ? (
                                <p className="torrent-rule__hint">{t("ruleUi.noFileRules")}</p>
                              ) : (
                                <div className="torrent-rule__clauses">
                                  {rule.torrentFileNameMatch.clauses.map((clause, clauseIndex) => (
                                    <div key={`${rule.type}-inline-torrent-file-clause-${clauseIndex}`} className="torrent-rule__clause">
                                      <input
                                        className="torrent-rule__clause-pattern"
                                        value={clause.pattern}
                                        onChange={(event) =>
                                          updateTorrentFileNameMatchClause(rule.type, clauseIndex, (current) => ({
                                            ...current,
                                            pattern: event.target.value
                                          }))
                                        }
                                        placeholder="hhd800.com or /sample/i"
                                        aria-label={t("ruleUi.filePattern")}
                                      />
                                      <select
                                        className="torrent-rule__clause-mode"
                                        value={clause.patternMode}
                                        onChange={(event) =>
                                          updateTorrentFileNameMatchClause(rule.type, clauseIndex, (current) => ({
                                            ...current,
                                            patternMode: event.target.value as TitleMatchPatternMode
                                          }))
                                        }
                                        aria-label={t("ruleUi.patternMode")}
                                      >
                                        <option value={TitleMatchPatternMode.Plain}>PLAIN</option>
                                        <option value={TitleMatchPatternMode.Regex}>REGEX</option>
                                      </select>
                                      <select
                                        className="torrent-rule__clause-effect"
                                        value={clause.effect}
                                        onChange={(event) =>
                                          updateTorrentFileNameMatchClause(rule.type, clauseIndex, (current) => ({
                                            ...current,
                                            effect: event.target.value as TorrentFileMatchEffect
                                          }))
                                        }
                                        aria-label={t("ruleUi.effect")}
                                      >
                                        <option value={TorrentFileMatchEffect.Prefer}>PREFER</option>
                                        <option value={TorrentFileMatchEffect.Avoid}>AVOID</option>
                                        <option value={TorrentFileMatchEffect.Lock}>LOCK</option>
                                      </select>
                                      <div className="torrent-rule__clause-actions">
                                        <button
                                          type="button"
                                          className="ghost-button"
                                          onClick={() => removeTorrentFileNameMatchClause(rule.type, clauseIndex)}
                                        >
                                          {t("common.delete")}
                                        </button>
                                      </div>
                                    </div>
                                  ))}
                                </div>
                              )}
                            </>
                          ) : null}
                        </div>
                      </>
                    ) : null}
                  </article>
                );
              })}
            </div>
          </section>

        </form>
      </article>
    );
  }

  if (settingsTab === "logs") {
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

  if (settingsTab === "system") {
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
              onChange={(event) => setSystemForm((current) => ({ ...current, taskDeletePolicy: event.target.value }))}
            >
              <option value={TaskDeletePolicy.KeepOnly}>仅删除 Moji 任务记录</option>
              <option value={TaskDeletePolicy.RemoveTorrent}>同时删除 qBittorrent 下载任务</option>
              <option value={TaskDeletePolicy.RemoveTorrentAndFiles}>同时删除 qBittorrent 下载任务和文件</option>
            </select>
          </label>
          <div className="settings-field">
            <FieldLabel text="图片缓存" info="图片始终由 Moji 代理。关闭后仍会读取已有缓存，但新下载的图片不再写入磁盘。" />
            <label className="switch-row image-cache-switch">
              <span className="switch-row__label">启用图片缓存</span>
              <span className="switch" role="switch" aria-checked={systemForm.imageCacheEnabled}>
                <input
                  type="checkbox"
                  checked={systemForm.imageCacheEnabled}
                  aria-label="启用图片缓存"
                  onChange={(event) => setSystemForm((current) => ({ ...current, imageCacheEnabled: event.target.checked }))}
                />
                <span className="switch__track" aria-hidden="true" />
                <span className="switch__thumb" aria-hidden="true" />
              </span>
            </label>
          </div>
          <div className={`image-cache-config${systemForm.imageCacheEnabled ? "" : " is-disabled"}`} aria-disabled={!systemForm.imageCacheEnabled}>
            <label className="settings-field">
              <FieldLabel text="缓存容量上限（MB）" info="允许 64–20480 MB；超过上限后按最近访问时间淘汰至上限的 90%。" />
              <input
                type="number"
                min="64"
                max="20480"
                disabled={!systemForm.imageCacheEnabled}
                value={systemForm.imageCacheMaxSizeMb}
                onChange={(event) => setSystemForm((current) => ({ ...current, imageCacheMaxSizeMb: event.target.value }))}
              />
            </label>
            <label className="settings-field">
              <FieldLabel text="缓存保留天数" info="允许 1–365 天；长期未访问的图片会被清理。" />
              <input
                type="number"
                min="1"
                max="365"
                disabled={!systemForm.imageCacheEnabled}
                value={systemForm.imageCacheRetentionDays}
                onChange={(event) => setSystemForm((current) => ({ ...current, imageCacheRetentionDays: event.target.value }))}
              />
            </label>
            {!systemForm.imageCacheEnabled ? <p className="image-cache-config__note">磁盘持久化已关闭，以上配置暂不生效。</p> : null}
          </div>
          <div className="image-cache-management">
            <div>
              <div className="settings-meta"><span>当前占用：{formatBytes(Number(runtimeStatus.imageCache.usedBytes ?? 0))}</span><span>图片：{runtimeStatus.imageCache.entryCount ?? 0} 张</span><span>最近清理：{formatDateTime(runtimeStatus.imageCache.lastCleanupAt)}</span></div>
              <p className="image-cache-management__hint">清空后保留图片来源登记，图片将在下次访问时自动重新下载。</p>
            </div>
            <button
              type="button"
              className="image-cache-management__clear"
              disabled={clearingImageCache || Number(runtimeStatus.imageCache.entryCount ?? 0) === 0}
              onClick={() => {
                setImageCacheClearError(null);
                setConfirmingImageCacheClear(true);
              }}
            >
              {clearingImageCache ? "清理中..." : Number(runtimeStatus.imageCache.entryCount ?? 0) === 0 ? "暂无缓存" : "清空图片缓存"}
            </button>
          </div>
          {confirmingImageCacheClear ? (
            <div className="image-cache-confirm" role="alertdialog" aria-labelledby="image-cache-confirm-title" aria-describedby="image-cache-confirm-description">
              <div>
                <strong id="image-cache-confirm-title">清空图片缓存？</strong>
                <p id="image-cache-confirm-description">
                  将删除 {runtimeStatus.imageCache.entryCount ?? 0} 张本地图片并释放 {formatBytes(Number(runtimeStatus.imageCache.usedBytes ?? 0))}。来源登记会保留。
                </p>
                {imageCacheClearError ? <p className="image-cache-confirm__error">{imageCacheClearError}</p> : null}
              </div>
              <div className="image-cache-confirm__actions">
                <button type="button" className="ghost-button" disabled={clearingImageCache} onClick={() => setConfirmingImageCacheClear(false)}>取消</button>
                <button type="button" className="image-cache-confirm__submit" disabled={clearingImageCache} onClick={() => void handleClearImageCache()}>{clearingImageCache ? "清理中..." : "确认清空"}</button>
              </div>
            </div>
          ) : null}
          {runtimeStatus.imageCache.lastError ? <p className="settings-feedback tone-danger">{runtimeStatus.imageCache.lastError}</p> : null}
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
