import IndexerOrderList from "./IndexerOrderList";
import { SettingsSubpanel, useSettingsPanel } from "./SettingsPanels";
import { useMemo, useState } from "react";
import { useMutation, useQuery } from "urql";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowDown } from "@fortawesome/free-solid-svg-icons/faArrowDown";
import { faArrowUp } from "@fortawesome/free-solid-svg-icons/faArrowUp";
import { faChevronDown } from "@fortawesome/free-solid-svg-icons/faChevronDown";
import { faChevronUp } from "@fortawesome/free-solid-svg-icons/faChevronUp";
import { faCircleInfo } from "@fortawesome/free-solid-svg-icons/faCircleInfo";
import { faGripVertical } from "@fortawesome/free-solid-svg-icons/faGripVertical";
import { faRotate } from "@fortawesome/free-solid-svg-icons/faRotate";
import { faTrash } from "@fortawesome/free-solid-svg-icons/faTrash";
import {
  JackettIndexersDocumentDocument,
  RefreshStashBoxesDocument,
  SubscriptionReleaseBehavior,
  SubscriptionReleaseDateRange,
  TorrentFileMatchEffect,
  TitleMatchEffect,
  TitleMatchPatternMode,
  TorrentSelectionDirection,
  TorrentSelectionRuleType,
  type SettingsPolicySectionQuery,
  type JackettIndexersDocumentQuery,
  type JackettIndexersDocumentQueryVariables,
  type RefreshStashBoxesMutation,
} from "../../graphql/generated/graphql";
import type { ToastTone } from "../../types";
import {
  DEFAULT_TORRENT_FILE_INSPECTION_RULES,
  DEFAULT_TORRENT_SELECTION_RULES,
  EMPTY_POLICY_SETTINGS_FORM,
} from "../../constants";
import { describeQueryError } from "../../services/queryError";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { SettingsSaveStatus, SettingsLoading } from "./SettingsTabCommon";

export const POLICY_SECTION_TITLE_KEYS = { tasks: "taskSettings.title", resources: "resourceRules.title", metadata: "metadataSources.title", indexers: "indexerSettings.title" } as const;
export type SettingsPolicyArea = keyof typeof POLICY_SECTION_TITLE_KEYS;

type RuntimeSettings = SettingsPolicySectionQuery["settings"];
type RuntimeSettingsStatus = SettingsPolicySectionQuery["settingsStatus"];
type JackettIndexer = JackettIndexersDocumentQuery["jackettIndexers"][number];

const TORRENT_SELECTION_RULE_KEYS: Partial<Record<TorrentSelectionRuleType, string>> = {
  [TorrentSelectionRuleType.TitleMatch]: "titleMatch",
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

type PolicySettingsForm = typeof EMPTY_POLICY_SETTINGS_FORM;
type TorrentSelectionRuleDraft = PolicySettingsForm["torrentSelection"]["fastRules"][number];
type PolicySettingsFormShape = PolicySettingsForm;

const numeric = (value: string) => value.trim() ? Number(value) : NaN;
function encodePolicy(form: PolicySettingsForm, area: SettingsPolicyArea) {
  if (area === "metadata") return {stash:{metadata_priority:form.stashBoxEndpoints}};
  const rules = form.torrentSelection.fastRules;
  if (area === "indexers") return {indexers:{query_order:rules.find(rule=>rule.type===TorrentSelectionRuleType.IndexerPreference)?.indexerPreference.trackerIds ?? []}};
  if (area === "tasks") return {tasks:{
    task_processing_enabled:form.taskProcessingEnabled,
    task_progress_sync_interval_seconds:numeric(form.taskProgressSyncIntervalSeconds),
    subscription_poll_interval_hours:numeric(form.subscriptionPollIntervalHours),
    sourcing_retry_interval_hours:numeric(form.sourcingRetryIntervalHours),
    subscription_release_policy:{solo_behavior:form.subscriptionReleasePolicy.soloBehavior,group_behavior:form.subscriptionReleasePolicy.groupBehavior,compilation_behavior:form.subscriptionReleasePolicy.compilationBehavior,vr_behavior:form.subscriptionReleasePolicy.vrBehavior,max_group_performer_count:numeric(form.subscriptionReleasePolicy.maxGroupPerformerCount),release_date_range:form.subscriptionReleasePolicy.releaseDateRange}
  }};
  const clause = (item: {pattern:string;patternMode:TitleMatchPatternMode;effect:TitleMatchEffect | TorrentFileMatchEffect})=>({pattern:item.pattern,pattern_mode:item.patternMode,effect:item.effect});
  const names: Partial<Record<TorrentSelectionRuleType,string>> = {
    [TorrentSelectionRuleType.TitleMatch]:"title_match",[TorrentSelectionRuleType.PublishDate]:"publish_date",[TorrentSelectionRuleType.TitleSimilarity]:"title_similarity",[TorrentSelectionRuleType.Seeders]:"seeders",[TorrentSelectionRuleType.Size]:"size",[TorrentSelectionRuleType.TorrentSingleVideo]:"torrent_single_video",[TorrentSelectionRuleType.TorrentFileNameMatch]:"torrent_file_name_match"
  };
  const encodeRules = (items:TorrentSelectionRuleDraft[]) => Object.fromEntries(items.filter(rule=>rule.type!==TorrentSelectionRuleType.IndexerPreference).map(rule=>[names[rule.type]!,{
    enabled:rule.enabled,
    ...(usesRuleDirection(rule.type)?{direction:getRuleDirection(rule)}:{}),
    ...(rule.type===TorrentSelectionRuleType.TitleMatch?{clauses:rule.titleMatch.clauses.map(clause)}:{}),
    ...(rule.type===TorrentSelectionRuleType.TorrentFileNameMatch?{clauses:rule.torrentFileNameMatch.clauses.map(clause)}:{})
  }]));
  return {tasks:{torrent_selection:{enabled:form.torrentSelection.enabled,inspection_candidate_limit:numeric(form.torrentSelection.inspectionCandidateLimit),fast_rule_order:rules.filter(rule=>rule.type!==TorrentSelectionRuleType.IndexerPreference).map(rule=>rule.type),fast_rules:encodeRules(rules),torrent_rules:encodeRules(form.torrentSelection.torrentRules)}}};
}

function renderSubscriptionReleaseBehaviorOptions(t: TFunction) {
  return SUBSCRIPTION_RELEASE_BEHAVIOR_OPTIONS.map((behavior) => (
    <option key={behavior} value={behavior}>
      {t(`releasePolicy.behaviors.${SUBSCRIPTION_RELEASE_BEHAVIOR_KEYS[behavior]}`)}
    </option>
  ));
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
  const kept = [...new Set(trackerIds)];
  const missing = enabledIds.filter((id) => !kept.includes(id));
  return [...kept, ...missing];
}

function mapRuntimeTorrentSelectionRule(rule: RuntimeSettings["policies"]["torrentSelection"]["fastRules"][number]): TorrentSelectionRuleDraft {
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
  const sourceFastRules = runtimeSettings.policies.torrentSelection.fastRules;
  const sourceTorrentRules = runtimeSettings.policies.torrentSelection.torrentRules;
  const fastRules = sourceFastRules.length > 0
    ? sourceFastRules.map(mapRuntimeTorrentSelectionRule)
    : DEFAULT_TORRENT_SELECTION_RULES.map((rule) => cloneTorrentSelectionRule(rule));
  const torrentRules = sourceTorrentRules.length > 0
    ? sourceTorrentRules.map(mapRuntimeTorrentSelectionRule)
    : DEFAULT_TORRENT_FILE_INSPECTION_RULES.map((rule) => cloneTorrentSelectionRule(rule));

  return {
    enabled: runtimeSettings.policies.torrentSelection.enabled,
    inspectionCandidateLimit: String(runtimeSettings.policies.torrentSelection.inspectionCandidateLimit || 5),
    fastRules,
    torrentRules
  };
}

interface PolicySettingsEditorProps {
  runtimeSettings: RuntimeSettings | null;
  runtimeStatus: RuntimeSettingsStatus | null;
  pushToast: (tone: ToastTone, message: string) => void;
  refreshSection: (opts?: Record<string, unknown>) => unknown;
  area?: SettingsPolicyArea;
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

export function PolicySettingsEditor({
  runtimeSettings,
  runtimeStatus,
  pushToast,
  refreshSection,
  area = "tasks"
}: PolicySettingsEditorProps) {
  const { t } = useTranslation();
  const panel = useSettingsPanel();
  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const initial = useMemo<PolicySettingsForm>(() => runtimeSettings ? ({
      taskProcessingEnabled: runtimeSettings.policies.taskProcessingEnabled ?? true,
      taskProgressSyncIntervalSeconds: String([30,60,120,300].includes(runtimeSettings.policies.taskProgressSyncIntervalSeconds) ? runtimeSettings.policies.taskProgressSyncIntervalSeconds : 60),
      sourcingRetryIntervalHours: String(runtimeSettings.policies.sourcingRetryIntervalHours || 6),
      subscriptionPollIntervalHours: String(runtimeSettings.policies.subscriptionPollIntervalHours || 1),
      stashBoxEndpoints:[...(runtimeSettings.policies.stashBoxEndpoints ?? [])],
      subscriptionReleasePolicy:{...runtimeSettings.policies.subscriptionReleasePolicy,vrBehavior:runtimeSettings.policies.subscriptionReleasePolicy.vrBehavior ?? SubscriptionReleaseBehavior.Review,maxGroupPerformerCount:String(runtimeSettings.policies.subscriptionReleasePolicy.maxGroupPerformerCount || 3)},
      torrentSelection:torrentSelectionFromRuntime(runtimeSettings)
  }) : EMPTY_POLICY_SETTINGS_FORM,[runtimeSettings]);
  const autosave = useSettingsAutosave(`${area}:policies`,initial,value=>encodePolicy(value,area));
  const {value:policyForm,setValue:setPolicySettingsForm}=autosave;
  // 规则列表的展开/收起是 UI 偏好：默认按「启用规则链」决定首屏可见性，
  // 之后用户的展开/收起操作与开关状态相互独立，不被开关重置。
  const [rulesExpanded, setRulesExpanded] = useState(() => policyForm.torrentSelection.enabled);
  const [{ data: jackettIndexersData, fetching: fetchingJackettIndexers }] = useQuery<
    JackettIndexersDocumentQuery,
    JackettIndexersDocumentQueryVariables
  >({
    query: JackettIndexersDocumentDocument,
    pause: area !== "indexers"
  });
  const jackettIndexers = (jackettIndexersData?.jackettIndexers ?? []).filter((indexer: JackettIndexer) => indexer.enabled);

  const [{ fetching: refreshingStashBoxes }, refreshStashBoxesMutation] = useMutation<
    RefreshStashBoxesMutation
  >(RefreshStashBoxesDocument);

  const moveCandidateRuleInSection = (section: "fast" | "file", from: number, to: number) => {
    setPolicySettingsForm((current) => {
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
    setPolicySettingsForm((current) => ({
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

  const refreshStashBoxes = async () => {
    const result = await refreshStashBoxesMutation({});
    if (result.error) {
      pushToast("tone-danger", t("metadataSources.loadFailed", { error: describeQueryError(result.error) }));
      return;
    }
    pushToast("tone-success", t("metadataSources.refresh"));
    await refreshSection({ requestPolicy: "network-only" });
  };

  const fastRules = policyForm.torrentSelection.fastRules;
  const fileInspectionRules = policyForm.torrentSelection.torrentRules;
  const visibleFastRules = fastRules.filter(rule => rule.type !== TorrentSelectionRuleType.IndexerPreference);
  const totalRuleCount = visibleFastRules.length + fileInspectionRules.length;

  if (!runtimeSettings || !runtimeStatus) {
    return <SettingsLoading title={t(POLICY_SECTION_TITLE_KEYS[area])} />;
  }

  if (area === "indexers") {
    const orderRule = fastRules.find(rule => rule.type === TorrentSelectionRuleType.IndexerPreference);
    const order = syncIndexerPreferenceTrackerIds(orderRule?.indexerPreference.trackerIds ?? [], jackettIndexers);
    const moveSite = (from: number, to: number) => {
      if (to < 0 || to >= order.length) return;
      const next = [...order]; const [id] = next.splice(from, 1); next.splice(to, 0, id);
      setPolicySettingsForm(current => ({ ...current, torrentSelection: { ...current.torrentSelection, fastRules: current.torrentSelection.fastRules.map(rule => rule.type === TorrentSelectionRuleType.IndexerPreference ? { ...rule, indexerPreference: { trackerIds: next } } : rule) } }));
    };
    return <article className="drawer-card"><div className="drawer-card__head"><h3>{t("indexerSettings.title")}</h3></div>
      <p className="settings-note">{t("indexerSettings.detail")}</p>
      {fetchingJackettIndexers ? <p role="status">{t("common.loading")}</p> : null}
      <form className="settings-form" onSubmit={event => event.preventDefault()}>
          <SettingsSaveStatus {...autosave} />
        <IndexerOrderList order={order} indexers={jackettIndexers} onMove={moveSite} />
        {!order.length ? <p className="settings-note">{t("indexerSettings.empty")}</p> : null}

      </form></article>;
  }

  {
    const stashBoxes = runtimeStatus.metadataSources.stashBoxes ?? [];
    const loaded = runtimeStatus.metadataSources.stashBoxesLoaded;
    const loadError = runtimeStatus.metadataSources.stashBoxesLoadError;
    const order = policyForm.stashBoxEndpoints;
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
      setPolicySettingsForm((current) => ({ ...current, stashBoxEndpoints: next }));
    };
    const move = (from: number, to: number) => {
      if (from === to || from < 0 || to < 0 || from >= display.length || to >= display.length) return;
      const next = display.map((item) => item.endpoint);
      const [moved] = next.splice(from, 1);
      next.splice(to, 0, moved);
      reorder(next);
    };

    return (
      <article className={area === "resources" ? "settings-resource-editor" : "drawer-card"}>
        {area !== "resources" ? <div className="drawer-card__head">
          <h3>{t(area === "tasks" && panel ? `settingsLayout.panels.${panel}` : POLICY_SECTION_TITLE_KEYS[area])}</h3>
        </div> : null}


        {area === "tasks" ? <>
        <form className="settings-form" onSubmit={event => event.preventDefault()}>
          <SettingsSaveStatus {...autosave} />
          <SettingsSubpanel id="schedule">
          <section className="task-processing-settings">
            <label className="switch-row">
              <span className="switch-row__label">{t("taskSettings.processingEnabled")}</span>
              <span className="switch" role="switch" aria-checked={policyForm.taskProcessingEnabled}>
                <input type="checkbox" aria-label={t("taskSettings.processingEnabled")} checked={policyForm.taskProcessingEnabled}
                  onChange={event => setPolicySettingsForm(current => ({ ...current, taskProcessingEnabled: event.target.checked }))} />
                <span className="switch__track" aria-hidden="true" />
                <span className="switch__thumb" aria-hidden="true" />
              </span>
            </label>
            <p className="settings-section__hint">{t("taskSettings.processingHelp")}</p>
            <label className="settings-field">
              <span>{t("taskSettings.syncInterval")}</span>
              <select value={policyForm.taskProgressSyncIntervalSeconds} disabled={!policyForm.taskProcessingEnabled}
                onChange={event => setPolicySettingsForm(current => ({ ...current, taskProgressSyncIntervalSeconds: event.target.value }))}>
                {[30, 60, 120, 300].map(seconds => <option value={seconds} key={seconds}>{t(`taskSettings.intervalOptions.${seconds}`)}</option>)}
              </select>
            </label>
          </section>
          <label className="settings-field">
            <span>{t("resourceTracking.interval")}</span>
            <input type="number" min="1" max="168" required
              value={policyForm.sourcingRetryIntervalHours}
              onChange={event => setPolicySettingsForm(current => ({ ...current, sourcingRetryIntervalHours: event.target.value }))} />
            <small>{t("resourceTracking.intervalHelp")}</small>
          </label>
          <label className="settings-field">
            <span>{t("taskSettings.pollInterval")}</span>
            <input
              value={policyForm.subscriptionPollIntervalHours}
              onChange={(event) => setPolicySettingsForm((current) => ({ ...current, subscriptionPollIntervalHours: event.target.value }))}
              placeholder="1"
            />
            <small>{t("taskSettings.pollIntervalHelp")}</small>
          </label>

          </SettingsSubpanel>
          <SettingsSubpanel id="releases">
          <section className="settings-section settings-section--nested">
            <header className="settings-section__head">
              <div>
                <h4>{t("taskSettings.policyTitle")}</h4>
                <p className="settings-section__hint">{t("taskSettings.policyScope")}</p>
              </div>
            </header>
            <label className="settings-field">
              <FieldLabel text={t("taskSettings.solo")}  />
              <select
                value={policyForm.subscriptionReleasePolicy.soloBehavior}
                onChange={(event) =>
                  setPolicySettingsForm((current) => ({
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
              <FieldLabel text={t("taskSettings.group")}  />
              <select
                value={policyForm.subscriptionReleasePolicy.groupBehavior}
                onChange={(event) =>
                  setPolicySettingsForm((current) => ({
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
              <FieldLabel text={t("taskSettings.compilation")} info={t("taskSettings.compilationInfo")} />
              <select
                value={policyForm.subscriptionReleasePolicy.compilationBehavior}
                onChange={(event) =>
                  setPolicySettingsForm((current) => ({
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
              <FieldLabel text={t("taskSettings.vr")} info={t("taskSettings.vrInfo")} />
              <select
                value={policyForm.subscriptionReleasePolicy.vrBehavior}
                onChange={(event) =>
                  setPolicySettingsForm((current) => ({
                    ...current,
                    subscriptionReleasePolicy: {
                      ...current.subscriptionReleasePolicy,
                      vrBehavior: event.target.value as SubscriptionReleaseBehavior
                    }
                  }))
                }
              >
                {renderSubscriptionReleaseBehaviorOptions(t)}
              </select>
            </label>
            <label className="settings-field">
              <FieldLabel text={t("taskSettings.dateRange")} info={t("taskSettings.dateRangeInfo")} />
              <select
                value={policyForm.subscriptionReleasePolicy.releaseDateRange}
                onChange={(event) =>
                  setPolicySettingsForm((current) => ({
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
                    {t(`releasePolicy.ranges.${SUBSCRIPTION_RELEASE_DATE_RANGE_KEYS[dateRange]}`)}
                  </option>
                ))}
              </select>
            </label>
            <p className="settings-section__hint">{t("taskSettings.behaviorHelp")}</p>
            <p className="settings-section__hint">{t("taskSettings.priorityHelp")}</p>
            <p className="settings-section__hint">{t("taskSettings.reviewHelp")}</p>
          </section>
          </SettingsSubpanel>
        </form>

        </> : null}
        {area === "metadata" ? <>
        <form className="settings-form" onSubmit={event => event.preventDefault()}>
          <SettingsSaveStatus {...autosave} />
          <section className="stashbox-source">
            <header className="stashbox-source__head">
              <div>
                <p className="stashbox-source__sub">
                  {t("metadataSources.detail")}
                </p>
              </div>
              <div className="stashbox-source__stats">
                <button
                  type="button"
                  className="ghost-button stashbox-source__refresh"
                  disabled={refreshingStashBoxes}
                  onClick={() => void refreshStashBoxes()}
                >
                  <FontAwesomeIcon icon={faRotate} className={refreshingStashBoxes ? "is-spinning" : undefined} />
                  <span>{refreshingStashBoxes ? t("metadataSources.refreshing") : t("metadataSources.refresh")}</span>
                </button>
              </div>
            </header>

            {!loaded ? (
              <div className="stashbox-source__empty stashbox-source__empty--loading">
                <div className="stashbox-source__spinner" aria-hidden="true" />
                <div>
                  <strong>{t("metadataSources.loading")}</strong>

                </div>
              </div>
            ) : display.length === 0 ? (
              <div className="stashbox-source__empty stashbox-source__empty--danger">
                <div className="stashbox-source__icon" aria-hidden="true">!</div>
                <div>
                  <strong>{t("metadataSources.empty")}</strong>
                  <p>{t("metadataSources.emptyDetail")}</p>
                  {loadError ? <p className="stashbox-source__error">{t("metadataSources.loadFailed", { error: loadError })}</p> : null}
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
                      <span className="stashbox-card__handle" aria-hidden="true" title={t("metadataSources.drag")}>
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
                            {t(item.box?.apiKeyConfigured ? "metadataSources.keyReady" : "metadataSources.keyMissing")}
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
                          aria-label={t("settings.moveUp")}
                        >
                          <FontAwesomeIcon icon={faArrowUp} />
                        </button>
                        <button
                          type="button"
                          className="ghost-button ghost-button--icon"
                          disabled={index === display.length - 1}
                          onClick={() => move(index, index + 1)}
                          aria-label={t("settings.moveDown")}
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

        </> : null}
        {area === "resources" ? <>
        <form className="settings-form" onSubmit={event => event.preventDefault()}>
          <SettingsSaveStatus {...autosave} />
          <section className="torrent-rules">
            <div className="settings-rule-engine drawer-card">
            <header className="torrent-rules__head">
              <div>
                <h4>{t("settingsLayout.groups.selectionEngine")}</h4>
                <p className="torrent-rules__sub">
                  {t("resourceRules.detail")}
                </p>
                <p className="torrent-rules__sub">{t("resourceRules.clauseHelp")}</p>
              </div>
            </header>

            <div className="torrent-rules__toolbar">
              <label className="switch-row">
                <span className="switch-row__label">{t("resourceRules.enableChain")}</span>
                <span className="switch" role="switch" aria-checked={policyForm.torrentSelection.enabled}>
                  <input
                    type="checkbox"
                    checked={policyForm.torrentSelection.enabled}
                    onChange={(event) => {
                      const next = event.target.checked;
                      setPolicySettingsForm((current) => ({
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
                {t("resourceRules.configured", { count: totalRuleCount })}
              </p>
            ) : null}

            </div>
            <div
              id="torrent-rules-list"
              className="torrent-rules__list"
              hidden={!rulesExpanded}
            >
              {fastRules.map((rule, ruleIndex) => {
                if (rule.type === TorrentSelectionRuleType.IndexerPreference) return null;
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
                    <span className="torrent-rule__handle" aria-hidden="true" title={t("resourceRules.drag")}>
                      <FontAwesomeIcon icon={faGripVertical} />
                    </span>
                    <span className="torrent-rule__order">{visibleFastRules.indexOf(rule) + 1}</span>
                    <h3 className="torrent-rule__name">{t(`resourceRules.names.${TORRENT_SELECTION_RULE_KEYS[rule.type]}`)}</h3>
                    <div className="torrent-rule__inline-readonly" aria-hidden="true">
                      <span className="torrent-rule__badge torrent-rule__badge--type">
                        {t(`resourceRules.names.${TORRENT_SELECTION_RULE_KEYS[rule.type]}`)}
                      </span>
                      {usesRuleDirection(rule.type) ? (
                        <span className="torrent-rule__badge torrent-rule__badge--dir">
                          {getRuleDirection(rule) === TorrentSelectionDirection.Asc ? "ASC" : "DESC"}
                        </span>
                      ) : null}
                    </div>
                    <div className="torrent-rule__actions">
                      <label className="switch switch--sm" role="switch" aria-checked={rule.enabled} aria-label={t("resourceRules.enableLabel", { rule: t(`resourceRules.names.${TORRENT_SELECTION_RULE_KEYS[rule.type]}`) })}>
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
                      <button type="button" className="ghost-button ghost-button--icon" onClick={() => moveCandidateRuleInSection("fast", ruleIndex, ruleIndex - 1)} disabled={visibleFastRules.indexOf(rule) === 0} aria-label={t("common.moveUp")}>
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
                        <div className="torrent-rule__body">
                          {usesRuleDirection(rule.type) ? (
                            <label className="torrent-rule__inline-field">
                              <span className="torrent-rule__inline-label">{t("resourceRules.direction")}</span>
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
                                <option value={TorrentSelectionDirection.Desc}>{t(`resourceRules.directions.${TORRENT_SELECTION_RULE_KEYS[rule.type]}.desc`)}</option>
                                <option value={TorrentSelectionDirection.Asc}>{t(`resourceRules.directions.${TORRENT_SELECTION_RULE_KEYS[rule.type]}.asc`)}</option>
                              </select>
                            </label>
                          ) : null}

                          {rule.type === TorrentSelectionRuleType.TitleMatch ? (
                            <>
                              <div className="torrent-rule__section-head">
                                <div>
                                  <p className="torrent-rule__hint">{t("resourceRules.titleHint")}</p>
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
                                <p className="torrent-rule__hint">{t("resourceRules.noTitleRules")}</p>
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
                                        aria-label={t("resourceRules.titlePattern")}
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
                                        aria-label={t("resourceRules.patternMode")}
                                      >
                                        <option value={TitleMatchPatternMode.Plain}>{t("resourceRules.modes.PLAIN")}</option>
                                        <option value={TitleMatchPatternMode.Regex}>{t("resourceRules.modes.REGEX")}</option>
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
                                        aria-label={t("resourceRules.effect")}
                                      >
                                        {[TitleMatchEffect.Require, TitleMatchEffect.Exclude, TitleMatchEffect.Prefer, TitleMatchEffect.Demote].map(effect => <option key={effect} value={effect}>{t(`resourceRules.effects.${effect}`)}</option>)}
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
                <label className="torrent-rules__limit-inline">
                  <FieldLabel text={t("resourceRules.inspectionScope")} info={t("resourceRules.inspectionInfo")} />
                  <input
                    type="number"
                    min="1"
                    step="1"
                    value={policyForm.torrentSelection.inspectionCandidateLimit}
                    onChange={(event) =>
                      setPolicySettingsForm((current) => ({
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
                const displayIndex = visibleFastRules.length + ruleIndex + 1;
                return (
                  <article key={rule.type} className={`torrent-rule${rule.enabled ? "" : " is-disabled"}`}>
                    <header className="torrent-rule__head">
                      <span className="torrent-rule__order">{displayIndex}</span>
                      <h3 className="torrent-rule__name">{t(`resourceRules.names.${TORRENT_SELECTION_RULE_KEYS[rule.type]}`)}</h3>
                      <div className="torrent-rule__inline-readonly" aria-hidden="true">
                        <span className="torrent-rule__badge torrent-rule__badge--type">
                          {t(`resourceRules.names.${TORRENT_SELECTION_RULE_KEYS[rule.type]}`)}
                        </span>
                        {usesRuleDirection(rule.type) ? (
                          <span className="torrent-rule__badge torrent-rule__badge--dir">
                            {getRuleDirection(rule) === TorrentSelectionDirection.Asc ? "ASC" : "DESC"}
                          </span>
                        ) : null}
                      </div>
                      <div className="torrent-rule__actions">
                        <label className="switch switch--sm" role="switch" aria-checked={rule.enabled} aria-label={t("resourceRules.enableLabel", { rule: t(`resourceRules.names.${TORRENT_SELECTION_RULE_KEYS[rule.type]}`) })}>
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
                        <div className="settings-form">
                          {rule.type === TorrentSelectionRuleType.TorrentSingleVideo ? (
                            <p className="torrent-rule__hint">
                              {t("resourceRules.singleVideoHint", { count: policyForm.torrentSelection.inspectionCandidateLimit || "5" })}
                            </p>
                          ) : null}

                          {rule.type === TorrentSelectionRuleType.TorrentFileNameMatch ? (
                            <>
                              <div className="drawer-card__head">
                                <div>
                                  <p className="torrent-rule__hint">{t("resourceRules.fileHint")}</p>
                                  <p className="torrent-rule__hint">{t("resourceRules.fileModeHint")}</p>
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
                                <p className="torrent-rule__hint">{t("resourceRules.noFileRules")}</p>
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
                                                            aria-label={t("resourceRules.filePattern")}
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
                                        aria-label={t("resourceRules.patternMode")}
                                      >
                                        <option value={TitleMatchPatternMode.Plain}>{t("resourceRules.modes.PLAIN")}</option>
                                        <option value={TitleMatchPatternMode.Regex}>{t("resourceRules.modes.REGEX")}</option>
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
                                        aria-label={t("resourceRules.effect")}
                                      >
                                        {[TorrentFileMatchEffect.Require, TorrentFileMatchEffect.Exclude, TorrentFileMatchEffect.Prefer, TorrentFileMatchEffect.Demote].map(effect => <option key={effect} value={effect}>{t(`resourceRules.effects.${effect}`)}</option>)}
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
        </> : null}
      </article>
    );
  }

  return null;
}
