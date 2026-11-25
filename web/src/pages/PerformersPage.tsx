import { workspaceToolbarIcons } from "../components/common/workspace-toolbar/icons";
import { ResponsivePageHeading } from "../components/layout/ResponsivePageHeading";
import { Pagination } from "../components/common/pagination/Pagination";
import { faUsers, faHeart, faClipboardQuestion, faTriangleExclamation } from "@fortawesome/free-solid-svg-icons";
import { useAppTimezone } from "../hooks/useAppTimezone";
import { faArrowLeft } from "@fortawesome/free-solid-svg-icons/faArrowLeft";
import { faBookmark } from "@fortawesome/free-solid-svg-icons/faBookmark";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PerformerCard } from "../components/performers/PerformerCard";
import { PerformerError, PerformerLoading } from "../components/performers/PerformerFeedback";
import { WorkspaceToolbar } from "../components/common/workspace-toolbar/WorkspaceToolbar";
import { PerformerListFilters } from "../components/performers/PerformerListFilters";
import { performerCheckRun } from "../hooks/useSubscriptionChecks";
import { PerformerOverview } from "../components/performers/PerformerOverview";
import { PerformerSceneCard } from "../components/performers/PerformerSceneCard";
import type { PerformersPageProps } from "../components/performers/types";
import {
  LibraryFilter,
  SceneSourceFilter, SceneCastFilter
} from "../graphql/generated/graphql";
import {
  formatDateTime,
  performerBatchEligibility
} from "../utils";

export function PerformersPage(props: PerformersPageProps) {
  useAppTimezone();
  const {
    stashPerformerPage,
    stashPerformers,
    performerDetail,
    performerScenePage,
    performerScenes,
    subscribedPerformers,
    fetchingStashPerformers,
    fetchingPerformerDetail,
    fetchingPerformerScenes,
    fetchingSubscription,
    performerBatchPending,
    queueingPerformerScenes,
    subscriptionSearch,
    selectedPerformerId,
    performerSceneSearch,
    performerSceneSourceFilter, performerSceneCastFilter,
    performerSceneLibraryFilter,
    performerScenePageSize,
    selectedSceneKeys,
    sceneSelectionMode,
    onToggleSceneSelectionMode,
    selectedPerformerIds,
    lastRefreshedAt,
    pendingSceneKeys,
    subscriptionError,
    stashPerformersError,
    performerDetailError,
    performerScenesError,
    onReload,
    onSelectVisiblePerformers,
    onClearPerformerSelection,
    onBatchSubscribePerformers,
    onBatchUnsubscribePerformers,
    onBatchRefreshPerformers,
    onBackToList,
    onPerformerSceneSearchChange,
    onPerformerSceneSourceChange, onPerformerSceneCastChange,
    onPerformerSceneLibraryChange,
    onPerformerScenePageSizeChange,
    onPerformerScenePageChange,
    onSelectCurrentScenePage,
    onClearSceneSelection,
    onQueueSelectedScenes,
    reviewPendingCount = 0,
    reviewCounts = {}
  } = props;
  const { t } = useTranslation();
  const checkStatusError = props.checksError ? <PerformerError error={new Error(`${t("subscriptionChecks.syncError")} ${props.checksError}`)} onRetry={props.onRefreshChecks} /> : null;
  const subscribedByID = useMemo(() => {
    return new Map(subscribedPerformers.map((item) => [item.performer.id, item]));
  }, [subscribedPerformers]);

  const activityByID = useMemo(() => new Map((props.performerActivities ?? []).map(activity => [activity.id, activity])), [props.performerActivities]);
  const performerSubscription = performerDetail ? (subscribedByID.get(performerDetail.performer.id) ?? null) : null;
  const { subscribedIDs: selectedSubscribedIds, unsubscribedIDs: selectedUnsubscribedIds } = performerBatchEligibility(selectedPerformerIds, subscribedPerformers.map((item) => item.performer.id));
  const hiddenSelectedPerformerCount = selectedPerformerIds.filter((id) => !stashPerformers.some((performer) => performer.id === id)).length;
  const currentPageKeys = performerScenes
    .filter((scene) => !scene.inLibrary && !scene.mojiTask && !scene.upcomingState && !pendingSceneKeys.includes(scene.key))
    .map((scene) => scene.key);

  if (selectedPerformerId) {
    return (
      <section className="section-band performer-workspace">
        <div className="band-head">
          <div>
            <h2>{t("performerUi.detail")}</h2>
          </div>
          <button type="button" className="ghost-button" onClick={onBackToList}>
            <FontAwesomeIcon icon={faArrowLeft} /> {t("performerUi.back")}
          </button>
        </div>

        {performerDetailError ? <PerformerError error={performerDetailError} onRetry={props.onRetryDetail} /> : null}
        {performerScenesError ? <PerformerError error={performerScenesError} onRetry={props.onRetryScenes} /> : null}
        {checkStatusError}

        {performerDetail ? (
          <PerformerOverview performerDetail={performerDetail} performerSubscription={performerSubscription} performerSummary={props.performerSummary} page={props} />
        ) : fetchingPerformerDetail ? <PerformerLoading detail /> : !performerDetailError ? <article className="empty-card empty-card--wide"><h3>{t("performerUi.detailMissing")}</h3><p>{t("performerUi.detailMissingHelp")}</p></article> : null}

        {props.performerSummaryError ? <PerformerError error={props.performerSummaryError} onRetry={props.onRetrySummary} /> : null}
        {props.fetchingPerformerSummary && !props.performerSummary ? <p className="settings-meta" role="status">{t("performerWorkspace.summaryLoading")}</p> : null}
        <WorkspaceToolbar selecting={sceneSelectionMode} toggleDisabled={queueingPerformerScenes}
          toggleLabel={t(sceneSelectionMode ? "performerUi.exitSceneSelection" : "performerUi.selectScenes")} onToggleSelection={onToggleSceneSelectionMode}
          selectionLabel={t("performerUi.selectScenes")}
          normal={{
            search: { label: t("performerUi.sceneSearch"), value: performerSceneSearch, onChange: onPerformerSceneSearchChange }, filters: [
              { id: "source", label: t("performerWorkspace.sourceFilter"), value: performerSceneSourceFilter, onChange: value => onPerformerSceneSourceChange(value as SceneSourceFilter), options: [{ value: SceneSourceFilter.All, label: t("performerUi.allSources") }, { value: SceneSourceFilter.Stash, label: "Stash" }, { value: SceneSourceFilter.Stashbox, label: "StashBox" }] },
              { id: "cast", label: t("performerWorkspace.castFilter"), value: performerSceneCastFilter, onChange: value => onPerformerSceneCastChange(value as SceneCastFilter), options: [{ value: SceneCastFilter.All, label: t("performerWorkspace.castAll") }, { value: SceneCastFilter.Solo, label: t("performerWorkspace.castSolo") }, { value: SceneCastFilter.Group, label: t("performerWorkspace.castGroup") }] },
              { id: "library", label: t("performerWorkspace.libraryFilter"), value: performerSceneLibraryFilter, onChange: value => onPerformerSceneLibraryChange(value as LibraryFilter), options: [{ value: LibraryFilter.All, label: t("performerUi.allStates") }, { value: LibraryFilter.InLibrary, label: t("performerUi.inLibrary") }, { value: LibraryFilter.NotInLibrary, label: t("performerUi.notInLibrary") }] },
            ], actions: []
          }}
          selection={{
            summary: t("performerWorkspace.selectedScenes", { count: selectedSceneKeys.length }) + (selectedSceneKeys.some(key => !performerScenes.some(scene => scene.key === key)) ? ` · ${t("performerWorkspace.hiddenScenes", { count: selectedSceneKeys.filter(key => !performerScenes.some(scene => scene.key === key)).length })}` : ""), actions: [
              { id: "select-page", label: t("performerUi.selectPage", { count: currentPageKeys.length }), icon: workspaceToolbarIcons.selectPage, showLabel: true, disabled: !currentPageKeys.length || fetchingPerformerScenes, onClick: () => onSelectCurrentScenePage(currentPageKeys) },
              { id: "clear", label: t("performerUi.clearSelection"), icon: workspaceToolbarIcons.clearSelection, showLabel: true, disabled: !selectedSceneKeys.length || queueingPerformerScenes, onClick: onClearSceneSelection },
              { id: "create", label: t(queueingPerformerScenes ? "performerUi.creatingMany" : selectedSceneKeys.length ? "performerUi.createMany" : "performerUi.createDownload", { count: selectedSceneKeys.length }), icon: workspaceToolbarIcons.createDownload, primary: true, showLabel: true, disabled: !selectedSceneKeys.length || queueingPerformerScenes || fetchingPerformerScenes, onClick: onQueueSelectedScenes },
            ]
          }} />
        <Pagination label={t("performerUi.works")} position="top" page={performerScenePage?.page ?? 1} pageSize={performerScenePageSize}
          totalPages={performerScenePage?.totalCountExact ? performerScenePage.totalPages ?? null : null} totalCount={performerScenePage?.totalCount ?? null}
          hasNextPage={performerScenePage?.hasNextPage} loading={fetchingPerformerScenes}
          onPageChange={onPerformerScenePageChange} onPageSizeChange={onPerformerScenePageSizeChange} />
        <div className="settings-meta performer-results-meta" aria-live="polite">
          <span>{t("performerWorkspace.visible", { count: performerScenes.length })}</span>
          <span>{t("performerUi.state", { state: t(fetchingPerformerDetail || fetchingPerformerScenes ? "performerUi.loading" : "performerUi.ready") })}</span>
        </div>

        {fetchingPerformerScenes && performerScenes.length === 0 ? <PerformerLoading /> : null}
        <div
          className="card-grid"
          style={{ marginTop: 16 }}
        >
          {performerScenes.map((scene) => <PerformerSceneCard key={scene.key} scene={scene} page={props} />)}

          {!fetchingPerformerScenes && !performerScenesError && performerScenes.length === 0 ? (
            <article className="empty-card empty-card--wide">
              <h3>{t("performerUi.noScenes")}</h3>
              <p>{t("performerUi.noScenesDetail")}</p>
            </article>
          ) : null}
        </div>

        <Pagination label={t("performerUi.works")} position="bottom" page={performerScenePage?.page ?? 1} pageSize={performerScenePageSize}
          totalPages={performerScenePage?.totalCountExact ? performerScenePage.totalPages ?? null : null} totalCount={performerScenePage?.totalCount ?? null}
          hasNextPage={performerScenePage?.hasNextPage} loading={fetchingPerformerScenes}
          onPageChange={onPerformerScenePageChange} onPageSizeChange={onPerformerScenePageSizeChange} />

      </section>
    );
  }



  return (
    <section className="section-band performer-workspace">
      <ResponsivePageHeading title={t("performerUi.list")} variant="filters" itemsLabel={t("performerWorkspace.filter")} itemsClassName="performer-filter-group"
        items={([["all", faUsers], ["subscribed", faBookmark], ["favorite", faHeart], ["review", faClipboardQuestion], ["error", faTriangleExclamation]] as const).map(([value, icon]) => ({
          id: value, label: t(`performerWorkspace.filters.${value}`), icon,
          selected: (props.listFilter ?? "all") === value, onSelect: () => props.onListFilterChange?.(value),
        }))} />

      {checkStatusError}
      <PerformerListFilters page={props} selection={{
        summary: t("performerBatch.selected", { count: selectedPerformerIds.length }) + (hiddenSelectedPerformerCount ? ` · ${t("performerBatch.hidden", { count: hiddenSelectedPerformerCount })}` : ""),
        actions: [
          { id: "select-page", label: t("performerBatch.selectVisible", { count: stashPerformers.length }), icon: workspaceToolbarIcons.selectPage, showLabel: true, disabled: performerBatchPending || fetchingStashPerformers || !stashPerformers.length, onClick: () => onSelectVisiblePerformers(stashPerformers.map(performer => performer.id)) },
          { id: "subscribe", label: t("performerBatch.subscribe", { count: selectedUnsubscribedIds.length }), icon: workspaceToolbarIcons.subscribe, showLabel: true, disabled: performerBatchPending || !selectedUnsubscribedIds.length, onClick: () => onBatchSubscribePerformers(selectedUnsubscribedIds) },
          { id: "check", label: t("performerBatch.refresh", { count: selectedPerformerIds.length }), title: t("performerBatch.refreshHelp"), icon: workspaceToolbarIcons.check, showLabel: true, disabled: performerBatchPending || !selectedPerformerIds.length || Boolean(props.checkActivity?.allRunId || props.checkStarting || props.checksLoading || props.checksError || selectedPerformerIds.some(id => performerCheckRun(props.checkActivity, id))), onClick: () => onBatchRefreshPerformers(selectedPerformerIds) },
          { id: "unsubscribe", label: t("performerBatch.unsubscribe", { count: selectedSubscribedIds.length }), icon: workspaceToolbarIcons.unsubscribe, showLabel: true, danger: true, disabled: performerBatchPending || !selectedSubscribedIds.length, onClick: () => onBatchUnsubscribePerformers(selectedSubscribedIds) },
          { id: "clear", label: t("performerBatch.clear"), icon: workspaceToolbarIcons.clearSelection, showLabel: true, disabled: performerBatchPending || !selectedPerformerIds.length, onClick: onClearPerformerSelection },
        ],
      }} />

      {lastRefreshedAt ? <p className="task-sync-meta">{t("performerUi.updatedAt", { time: formatDateTime(lastRefreshedAt) })}</p> : null}

      <Pagination label={t("performerUi.list")} position="top" page={props.stashPerformerPage?.page ?? 1} pageSize={props.subscriptionPageSize}
        totalPages={props.stashPerformerPage?.totalPages ?? null} totalCount={props.stashPerformerPage?.totalCount ?? null} loading={fetchingStashPerformers}
        onPageChange={props.onPageChange} onPageSizeChange={props.onPageSizeChange} />


      {subscriptionError || stashPerformersError ? <PerformerError error={(subscriptionError || stashPerformersError)!} onRetry={onReload} /> : null}
      {fetchingStashPerformers && stashPerformers.length === 0 ? <PerformerLoading /> : null}

      <div className="profile-grid">
        {stashPerformers.length === 0 && !fetchingStashPerformers && !stashPerformersError && !subscriptionError ? (
          <article className="empty-card empty-card--wide">
            <h3>{t("performerUi.noPerformers")}</h3>
            <p>{t(props.listFilter && props.listFilter !== "all" ? "performerWorkspace.filterEmpty" : "performerUi.noPerformersDetail")}</p>
            {subscriptionSearch || (props.listFilter && props.listFilter !== "all") ? <button type="button" className="ghost-button" onClick={() => { props.onClearListFilters?.(); }}>{t("performerWorkspace.clearFilters")}</button> : null}
          </article>
        ) : null}
        {stashPerformers.map((performer) => <PerformerCard key={performer.id} performer={performer} activity={activityByID.get(performer.id)} subscription={subscribedByID.get(performer.id) ?? null} reviewCount={reviewCounts[performer.id] ?? subscribedByID.get(performer.id)?.reviewReleaseCount ?? 0} page={props} />)}
      </div>

      <Pagination label={t("performerUi.list")} position="bottom" page={props.stashPerformerPage?.page ?? 1} pageSize={props.subscriptionPageSize}
        totalPages={props.stashPerformerPage?.totalPages ?? null} totalCount={props.stashPerformerPage?.totalCount ?? null} loading={fetchingStashPerformers}
        onPageChange={props.onPageChange} onPageSizeChange={props.onPageSizeChange} />
    </section>
  );
}
