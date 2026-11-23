import "../../styles/resource-selection.scss";
import { hasResourceConditionValue } from "./resourceSelectionDraft";
import { ResourceSelectionEditor } from "./ResourceSelectionEditor";
import { TaskProcessingFrequency } from "../../graphql/generated/graphql";
import { SettingsSubpanel, useSettingsPanel } from "./SettingsPanels";
import { useMemo, useState } from "react";
import { useMutation } from "urql";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowDown } from "@fortawesome/free-solid-svg-icons/faArrowDown";
import { faArrowUp } from "@fortawesome/free-solid-svg-icons/faArrowUp";
import { faChevronDown } from "@fortawesome/free-solid-svg-icons/faChevronDown";
import { faChevronUp } from "@fortawesome/free-solid-svg-icons/faChevronUp";
import { faGripVertical } from "@fortawesome/free-solid-svg-icons/faGripVertical";
import { faRotate } from "@fortawesome/free-solid-svg-icons/faRotate";
import { faTrash } from "@fortawesome/free-solid-svg-icons/faTrash";
import {
  RefreshStashBoxesDocument,
  SubscriptionReleaseBehavior,
  SubscriptionReleaseDateRange,
  type SettingsPolicySectionQuery,
  type RefreshStashBoxesMutation,
} from "../../graphql/generated/graphql";
import type { ToastTone } from "../../types";
import {
  EMPTY_POLICY_SETTINGS_FORM,
} from "../../constants";
import { describeQueryError } from "../../services/queryError";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { FieldLabel, SettingsSaveStatus, SettingsLoading } from "./SettingsTabCommon";

export const POLICY_SECTION_TITLE_KEYS = { tasks: "taskSettings.title", resources: "resourceRules.title", metadata: "metadataSources.title" } as const;
export type SettingsPolicyArea = keyof typeof POLICY_SECTION_TITLE_KEYS;

type RuntimeSettings = SettingsPolicySectionQuery["settings"];
type RuntimeSettingsStatus = SettingsPolicySectionQuery["settingsStatus"];

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

type PolicySettingsForm = typeof EMPTY_POLICY_SETTINGS_FORM;
type PolicySettingsFormShape = PolicySettingsForm;

const numeric = (value: string) => value.trim() ? Number(value) : NaN;
function encodePolicy(form: PolicySettingsForm, area: SettingsPolicyArea) {
  if (area === "metadata") return {stash:{metadata_priority:form.stashBoxEndpoints}};
  if (area === "tasks") return {tasks:{
    task_processing_frequency:form.taskProcessingFrequency,

    sourcing_retry_interval_hours:numeric(form.sourcingRetryIntervalHours),
    subscription_release_policy:{solo_behavior:form.subscriptionReleasePolicy.soloBehavior,group_behavior:form.subscriptionReleasePolicy.groupBehavior,compilation_behavior:form.subscriptionReleasePolicy.compilationBehavior,vr_behavior:form.subscriptionReleasePolicy.vrBehavior,max_group_performer_count:numeric(form.subscriptionReleasePolicy.maxGroupPerformerCount),release_date_range:form.subscriptionReleasePolicy.releaseDateRange}
  }};
  const p=form.torrentSelection;
  return {tasks:{torrent_selection:{version:p.version,active_group:p.activeGroup,rules:p.rules.map(({id,name,conditions})=>({id,name,conditions:conditions.filter(hasResourceConditionValue).map(({scope,value,minimum,maximum})=>({scope,value,minimum:minimum.trim(),maximum:maximum.trim()}))})),groups:p.groups.map(({id,name,fallback,require,exclude,tiers})=>({id,name,fallback,require:[...require],exclude:[...exclude],tiers:tiers.map(({id,name,rules})=>({id,name,rules:[...rules]}))}))}}};
}

function renderSubscriptionReleaseBehaviorOptions(t: TFunction) {
  return SUBSCRIPTION_RELEASE_BEHAVIOR_OPTIONS.map((behavior) => (
    <option key={behavior} value={behavior}>
      {t(`releasePolicy.behaviors.${SUBSCRIPTION_RELEASE_BEHAVIOR_KEYS[behavior]}`)}
    </option>
  ));
}

function torrentSelectionFromRuntime(settings:RuntimeSettings) {
 const p=settings.policies.torrentSelection;
 return {version:p.version,activeGroup:p.activeGroup,rules:p.rules.map(r=>({id:r.id,name:r.name,conditions:r.conditions.map(({scope,value,minimum,maximum})=>({scope,value,minimum,maximum}))})),groups:p.groups.map(g=>({id:g.id,name:g.name,fallback:g.fallback,require:[...g.require],exclude:[...g.exclude],tiers:g.tiers.map(t=>({id:t.id,name:t.name,rules:[...t.rules]}))}))};
}

interface PolicySettingsEditorProps {
  runtimeSettings: RuntimeSettings | null;
  runtimeStatus: RuntimeSettingsStatus | null;
  pushToast: (tone: ToastTone, message: string) => void;
  refreshSection: (opts?: Record<string, unknown>) => unknown;
  area?: SettingsPolicyArea;
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
      taskProcessingFrequency: runtimeSettings.policies.taskProcessingFrequency,
      sourcingRetryIntervalHours: String(runtimeSettings.policies.sourcingRetryIntervalHours || 6),

      stashBoxEndpoints:[...(runtimeSettings.policies.stashBoxEndpoints ?? [])],
      subscriptionReleasePolicy:{...runtimeSettings.policies.subscriptionReleasePolicy,vrBehavior:runtimeSettings.policies.subscriptionReleasePolicy.vrBehavior ?? SubscriptionReleaseBehavior.Review,maxGroupPerformerCount:String(runtimeSettings.policies.subscriptionReleasePolicy.maxGroupPerformerCount || 3)},
      torrentSelection:torrentSelectionFromRuntime(runtimeSettings)
  }) : EMPTY_POLICY_SETTINGS_FORM,[runtimeSettings]);
  const autosave = useSettingsAutosave(`${area}:policies`,initial,value=>encodePolicy(value,area));
  const {value:policyForm,setValue:setPolicySettingsForm}=autosave;
  const [{ fetching: refreshingStashBoxes }, refreshStashBoxesMutation] = useMutation<
    RefreshStashBoxesMutation
  >(RefreshStashBoxesDocument);

  const refreshStashBoxes = async () => {
    const result = await refreshStashBoxesMutation({});
    if (result.error) {
      pushToast("tone-danger", t("metadataSources.loadFailed", { error: describeQueryError(result.error) }));
      return;
    }
    pushToast("tone-success", t("metadataSources.refresh"));
    await refreshSection({ requestPolicy: "network-only" });
  };

  if (!runtimeSettings || !runtimeStatus) {
    return <SettingsLoading title={t(POLICY_SECTION_TITLE_KEYS[area])} />;
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
            <label className="settings-field">
              <span>{t("taskSettings.frequency")}</span>
              <select aria-label={t("taskSettings.frequency")} aria-describedby="task-processing-frequency-help task-processing-frequency-scope" value={policyForm.taskProcessingFrequency}
                onChange={event => setPolicySettingsForm(current => ({ ...current, taskProcessingFrequency: event.target.value as TaskProcessingFrequency }))}>
                {[TaskProcessingFrequency.High, TaskProcessingFrequency.Medium, TaskProcessingFrequency.Low].map(frequency => <option value={frequency} key={frequency}>{t(`taskSettings.frequencyOptions.${frequency}`)}</option>)}
              </select>
              <small id="task-processing-frequency-help">{t(`taskSettings.frequencyHelp.${policyForm.taskProcessingFrequency}`)}</small>
              <small id="task-processing-frequency-scope">{t("taskSettings.frequencyScope")}</small>
            </label>
          </section>
          <label className="settings-field">
            <span>{t("resourceTracking.interval")}</span>
            <input type="number" min="1" max="168" required
              value={policyForm.sourcingRetryIntervalHours}
              onChange={event => setPolicySettingsForm(current => ({ ...current, sourcingRetryIntervalHours: event.target.value }))} />
            <small>{t("resourceTracking.intervalHelp")}</small>
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
        {area === "resources" ? <form className="settings-form" onSubmit={event=>event.preventDefault()}>
         <ResourceSelectionEditor active={panel===null||panel==="resources"} saveStatus={<SettingsSaveStatus {...autosave}/>} value={policyForm.torrentSelection} onChange={value=>setPolicySettingsForm(current=>({...current,torrentSelection:value}))}/>
        </form>:null}
      </article>
    );
  }

  return null;
}
