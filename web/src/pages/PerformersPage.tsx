import { faArrowLeft } from "@fortawesome/free-solid-svg-icons/faArrowLeft";
import { faBinoculars } from "@fortawesome/free-solid-svg-icons/faBinoculars";
import { faListCheck } from "@fortawesome/free-solid-svg-icons/faListCheck";
import { faBookmark } from "@fortawesome/free-solid-svg-icons/faBookmark";
import { faXmark } from "@fortawesome/free-solid-svg-icons/faXmark";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PerformerCard } from "../components/performers/PerformerCard";
import { PerformerError, PerformerLoading } from "../components/performers/PerformerFeedback";
import { PerformerToolbar } from "../components/performers/PerformerToolbar";
import { PerformerListFilters } from "../components/performers/PerformerListFilters";
import { PerformerOverview } from "../components/performers/PerformerOverview";
import { PerformerPagination } from "../components/performers/PerformerPagination";
import { PerformerSceneCard } from "../components/performers/PerformerSceneCard";
import type { PerformersPageProps } from "../components/performers/types";
import { SUBSCRIPTION_PAGE_SIZE_OPTIONS } from "../constants";
import {
  LibraryFilter,
  SceneSourceFilter
} from "../graphql/generated/graphql";
import {
  formatDateTime,
  performerBatchEligibility
} from "../utils";

export function PerformersPage(props: PerformersPageProps) {
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
    performerSceneSourceFilter,
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
    onPerformerSceneSourceChange,
    onPerformerSceneLibraryChange,
    onPerformerScenePageSizeChange,
    onPrevPerformerScenePage,
    onNextPerformerScenePage,
    onSelectCurrentScenePage,
    onClearSceneSelection,
    onQueueSelectedScenes,
    reviewPendingCount = 0,
    reviewCounts = {}
  } = props;
  const { t } = useTranslation();
  const subscribedByID = useMemo(() => {
    return new Map(subscribedPerformers.map((item) => [item.performer.id, item]));
  }, [subscribedPerformers]);

  const performerSubscription = performerDetail ? (subscribedByID.get(performerDetail.performer.id) ?? null) : null;
  const { subscribedIDs: selectedSubscribedIds, unsubscribedIDs: selectedUnsubscribedIds } = performerBatchEligibility(selectedPerformerIds, subscribedPerformers.map((item) => item.performer.id));
  const hiddenSelectedPerformerCount = selectedPerformerIds.filter((id) => !stashPerformers.some((performer) => performer.id === id)).length;
  const currentPageKeys = performerScenes
    .filter((scene) => !scene.inLibrary && !scene.mojiTask && !pendingSceneKeys.includes(scene.key))
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

        {performerDetail ? (
          <PerformerOverview performerDetail={performerDetail} performerSubscription={performerSubscription} performerSummary={props.performerSummary} page={props} />
        ) : fetchingPerformerDetail ? <PerformerLoading detail /> : !performerDetailError ? <article className="empty-card empty-card--wide"><h3>{t("performerUi.detailMissing")}</h3><p>{t("performerUi.detailMissingHelp")}</p></article> : null}

        {props.performerSummaryError ? <PerformerError error={props.performerSummaryError} onRetry={props.onRetrySummary} /> : null}
        {props.fetchingPerformerSummary && !props.performerSummary ? <p className="settings-meta" role="status">{t("performerWorkspace.summaryLoading")}</p> : null}
        <PerformerToolbar expanded={sceneSelectionMode} renderControls={panelId => (
          <div className="toolbar-inline toolbar-inline--subscription performer-scene-filters">
            <input
              aria-label={t("performerUi.sceneSearch")}
              placeholder={t("performerUi.sceneSearch")}
              value={performerSceneSearch}
              onChange={(event) => onPerformerSceneSearchChange(event.target.value)}
            />
            <select aria-label={t("performerWorkspace.sourceFilter")} value={performerSceneSourceFilter} onChange={(event) => onPerformerSceneSourceChange(event.target.value as SceneSourceFilter)}>
              <option value={SceneSourceFilter.All}>{t("performerUi.allSources")}</option>
              <option value={SceneSourceFilter.Stash}>Stash</option>
              <option value={SceneSourceFilter.Stashbox}>StashBox</option>
            </select>
            <select aria-label={t("performerWorkspace.libraryFilter")} value={performerSceneLibraryFilter} onChange={(event) => onPerformerSceneLibraryChange(event.target.value as LibraryFilter)}>
              <option value={LibraryFilter.All}>{t("performerUi.allStates")}</option>
              <option value={LibraryFilter.InLibrary}>{t("performerUi.inLibrary")}</option>
              <option value={LibraryFilter.NotInLibrary}>{t("performerUi.notInLibrary")}</option>
            </select>
            <select aria-label={t("performerWorkspace.pageSize")} value={performerScenePageSize} onChange={(event) => onPerformerScenePageSizeChange(Number(event.target.value))}>
              {SUBSCRIPTION_PAGE_SIZE_OPTIONS.map((size) => (
                <option key={size} value={size}>
                  {t("performerUi.pageSize", { size })}
                </option>
              ))}
            </select>
            <button type="button" className="ghost-button task-icon-button task-icon-button--bordered performer-toolbar__toggle" aria-pressed={sceneSelectionMode} aria-expanded={sceneSelectionMode} aria-controls={panelId} aria-label={t(sceneSelectionMode ? "performerUi.exitSceneSelection" : "performerUi.selectScenes")} title={t(sceneSelectionMode ? "performerUi.exitSceneSelection" : "performerUi.selectScenes")} onClick={onToggleSceneSelectionMode} disabled={queueingPerformerScenes}>
              <FontAwesomeIcon icon={sceneSelectionMode ? faXmark : faListCheck} />
            </button>
          </div>
        )}>
          <div className="performer-scene-selection">
            <button
              type="button"
              className="ghost-button"
              disabled={currentPageKeys.length === 0 || fetchingPerformerScenes}
              onClick={() => onSelectCurrentScenePage(currentPageKeys)}
            >
              {t("performerUi.selectPage", { count: currentPageKeys.length })}
            </button>
            <button
              type="button"
              className="ghost-button"
              disabled={selectedSceneKeys.length === 0 || queueingPerformerScenes}
              onClick={onClearSceneSelection}
            >
              {t("performerUi.clearSelection")}
            </button>
            <button
              type="button"
              className="primary-button"
              disabled={selectedSceneKeys.length === 0 || queueingPerformerScenes || fetchingPerformerScenes}
              onClick={onQueueSelectedScenes}
            >
              {queueingPerformerScenes
                ? t("performerUi.creatingMany", { count: selectedSceneKeys.length })
                : selectedSceneKeys.length > 0
                  ? t("performerUi.createMany", { count: selectedSceneKeys.length })
                  : t("performerUi.createDownload")}
            </button>
          </div>

          {selectedSceneKeys.length ? <p className="performer-selection-summary" role="status">{t("performerWorkspace.selectedScenes", { count: selectedSceneKeys.length })}{selectedSceneKeys.some(key => !performerScenes.some(scene => scene.key === key)) ? ` · ${t("performerWorkspace.hiddenScenes", { count: selectedSceneKeys.filter(key => !performerScenes.some(scene => scene.key === key)).length })}` : ""}</p> : null}
        </PerformerToolbar>
        <div className="settings-meta performer-results-meta" aria-live="polite">
          <span>{t(performerScenePage?.totalCountExact ? "performerWorkspace.total" : "performerCacheUi.loadedItems", { count: performerScenePage?.totalCount ?? 0 })}</span>
          <span>{t("performerWorkspace.visible", { count: performerScenes.length })}</span>
          <span>{performerScenePage?.totalPages != null ? t("performerUi.pagination", { page: performerScenePage.page, pages: performerScenePage.totalPages }) : t("performerCacheUi.loadingMorePages", { page: performerScenePage?.page ?? 1 })}</span>
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

        {performerScenePage && (performerScenePage.hasPrevPage || performerScenePage.hasNextPage) ? (
          <div className="pagination-bar">
            <button type="button" className="ghost-button" disabled={!performerScenePage.hasPrevPage || fetchingPerformerScenes} onClick={onPrevPerformerScenePage}>
              {t("performerUi.previous")}
            </button>
            <span className="status-chip tone-neutral">
              {performerScenePage.totalPages != null ? t("performerUi.page", { page: performerScenePage.page, pages: performerScenePage.totalPages }) : t("performerCacheUi.pageUnknownTotal", { page: performerScenePage.page })}
            </span>
            <button type="button" className="ghost-button" disabled={!performerScenePage.hasNextPage || fetchingPerformerScenes} onClick={onNextPerformerScenePage}>
              {t("performerUi.next")}
            </button>
          </div>
        ) : null}

      </section>
    );
  }



  return (
    <section className="section-band performer-workspace">
      <div className="band-head">
        <div>
          <h2>{t("performerUi.list")}</h2>
        </div>
        <div className="performer-filter-group" role="group" aria-label={t("performerWorkspace.filter")}>
          {(["all", "subscribed", "favorite", "review", "error"] as const).map(value => <button type="button" className={`ghost-button${(props.listFilter ?? "all") === value ? " is-active" : ""}`} aria-pressed={(props.listFilter ?? "all") === value} key={value} onClick={() => props.onListFilterChange?.(value)}>{t(`performerWorkspace.filters.${value}`)}</button>)}
        </div>
      </div>

      <PerformerListFilters page={props}>
        <div role="region" aria-label={t("performerBatch.selectionActions")}>
          <div className="performer-list-selection">
            <button type="button" className="ghost-button" disabled={performerBatchPending || fetchingStashPerformers || stashPerformers.length === 0} onClick={() => onSelectVisiblePerformers(stashPerformers.map((performer) => performer.id))}>{t("performerBatch.selectVisible", { count: stashPerformers.length })}</button>
            <button type="button" className="ghost-button" disabled={performerBatchPending || selectedUnsubscribedIds.length === 0} onClick={() => onBatchSubscribePerformers(selectedUnsubscribedIds)}><FontAwesomeIcon icon={faBookmark} /> {t("performerBatch.subscribe", { count: selectedUnsubscribedIds.length })}</button>
            <button type="button" className="ghost-button" disabled={performerBatchPending || selectedSubscribedIds.length === 0} onClick={() => onBatchRefreshPerformers(selectedSubscribedIds)} title={t("performerBatch.refreshHelp")}><FontAwesomeIcon icon={faBinoculars} /> {t("performerBatch.refresh", { count: selectedSubscribedIds.length })}</button>
            <button type="button" className="ghost-button task-ops__button--danger" disabled={performerBatchPending || selectedSubscribedIds.length === 0} onClick={() => onBatchUnsubscribePerformers(selectedSubscribedIds)}><FontAwesomeIcon icon={faBookmark} /> {t("performerBatch.unsubscribe", { count: selectedSubscribedIds.length })}</button>
            <button type="button" className="ghost-button" disabled={performerBatchPending || selectedPerformerIds.length === 0} onClick={onClearPerformerSelection}><FontAwesomeIcon icon={faXmark} /> {t("performerBatch.clear")}</button>
          </div>
          <p className="performer-selection-summary" role="status"><strong>{t("performerBatch.selected", { count: selectedPerformerIds.length })}</strong>{hiddenSelectedPerformerCount ? ` · ${t("performerBatch.hidden", { count: hiddenSelectedPerformerCount })}` : ""}</p>
        </div>
      </PerformerListFilters>

      {lastRefreshedAt ? <p className="task-sync-meta">{t("performerUi.updatedAt", { time: formatDateTime(lastRefreshedAt) })}</p> : null}

      <PerformerPagination page={props} />


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
        {stashPerformers.map((performer) => <PerformerCard key={performer.id} performer={performer} subscription={subscribedByID.get(performer.id) ?? null} reviewCount={reviewCounts[performer.id] ?? subscribedByID.get(performer.id)?.reviewReleaseCount ?? 0} page={props} />)}
      </div>

      <PerformerPagination page={props} />
    </section>
  );
}
