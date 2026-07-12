import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faArrowLeft,
  faArrowUpRightFromSquare,
  faBookmark,
  faCheck,
  faFilm,
  faHeart,
  faPlus,
  faPlayCircle,
  faRotate,
  faTags,
  faUsers
} from "@fortawesome/free-solid-svg-icons";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PerformerDetailView, PerformerListView } from "../components/performers/PerformerViews";
import i18n from "../i18n/i18n";
import { SUBSCRIPTION_PAGE_SIZE_OPTIONS } from "../constants";
import { describeQueryError } from "../services/queryError";
import {
  formatDateTime,
  formatRelative,
  formatRelativeDate,
  performerImageURL,
  performerInitials,
  stashPerformerURL
} from "../utils";
import {
  LibraryFilter,
  SceneSourceFilter,
  TaskStage,
  TaskStageStatus,
  type StashPerformerDetailQuery,
  type StashPerformerScenesQuery,
  type StashPerformersQuery,
  type SubscribedPerformersQuery
} from "../graphql/generated/graphql";

type StashPerformerEntry = StashPerformersQuery["stashPerformers"]["items"][number];
type StashPerformerPage = StashPerformersQuery["stashPerformers"];
type StashPerformerDetail = StashPerformerDetailQuery["stashPerformerDetail"];
type StashPerformerScenePage = StashPerformerScenesQuery["stashPerformerScenes"];
type StashPerformerSceneEntry = StashPerformerScenePage["items"][number];
type SubscribedPerformerEntry = SubscribedPerformersQuery["subscribedPerformers"][number];
function performerSceneTaskLabel(task: NonNullable<StashPerformerSceneEntry["mojiTask"]>) {
  if (task.stage === TaskStage.Downloading && task.progress > 0) {
    return i18n.t("performerUi.downloadProgress", { progress: Math.round(task.progress * 100) });
  }
  if (task.stageStatus === TaskStageStatus.Blocked) {
    return i18n.t("taskModel.states.blocked");
  }
  if (task.stage === TaskStage.Completed) return i18n.t("taskModel.states.completed");
  return i18n.t("taskModel.states.queued");
}

function performerSceneTaskTone(task: NonNullable<StashPerformerSceneEntry["mojiTask"]>) {
  if (task.stageStatus === TaskStageStatus.Blocked) return "tone-danger";
  if (task.stage === TaskStage.Completed) return "tone-success";
  return "tone-info";
}

function performerSceneSourceLabel(scene: StashPerformerSceneEntry) {
  if (scene.hasStashSource && scene.hasStashBoxSource) return i18n.t("performerUi.dualSource");
  if (scene.hasStashBoxSource) return "StashBox";
  return "Stash";
}

interface PerformersPageProps {
  stashBaseURL: string | null;
  stashPerformerPage: StashPerformerPage | null;
  stashPerformers: StashPerformerEntry[];
  performerDetail: StashPerformerDetail | null;
  performerScenePage: StashPerformerScenePage | null;
  performerScenes: StashPerformerSceneEntry[];
  subscribedPerformers: SubscribedPerformerEntry[];
  fetchingStashPerformers: boolean;
  fetchingPerformerDetail: boolean;
  fetchingPerformerScenes: boolean;
  fetchingSubscription: boolean;
  refreshingSubscriptionNow: boolean;
  queueingPerformerScenes: boolean;
  subscriptionSearch: string;
  subscriptionPageSize: number;
  selectedPerformerId: string | null;
  performerSceneSearch: string;
  performerSceneSourceFilter: SceneSourceFilter;
  performerSceneLibraryFilter: LibraryFilter;
  performerScenePageSize: number;
  selectedSceneKeys: string[];
  pendingSceneKeys: string[];
  pendingSubscriptionID: string | null;
  subscriptionError: Error | null;
  stashPerformersError: Error | null;
  performerDetailError: Error | null;
  performerScenesError: Error | null;
  onSearchChange: (value: string) => void;
  onPageSizeChange: (size: number) => void;
  onReload: () => void;
  onRefreshAll: () => void;
  onToggle: (performer: StashPerformerEntry) => void;
  onRefreshOne: (performer: StashPerformerEntry) => void;
  onPrevPage: () => void;
  onNextPage: () => void;
  onOpenPerformer: (performerId: string) => void;
  onOpenTask: (taskId: string) => void;
  onBackToList: () => void;
  onPerformerSceneSearchChange: (value: string) => void;
  onPerformerSceneSourceChange: (value: SceneSourceFilter) => void;
  onPerformerSceneLibraryChange: (value: LibraryFilter) => void;
  onPerformerScenePageSizeChange: (value: number) => void;
  onPrevPerformerScenePage: () => void;
  onNextPerformerScenePage: () => void;
  onToggleSceneSelection: (key: string) => void;
  onSelectCurrentScenePage: (keys: string[]) => void;
  onClearSceneSelection: () => void;
  onQueueSelectedScenes: () => void;
  onQueueScene: (scene: StashPerformerSceneEntry) => void;
}

export function PerformersPage({
  stashBaseURL,
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
  refreshingSubscriptionNow,
  queueingPerformerScenes,
  subscriptionSearch,
  subscriptionPageSize,
  selectedPerformerId,
  performerSceneSearch,
  performerSceneSourceFilter,
  performerSceneLibraryFilter,
  performerScenePageSize,
  selectedSceneKeys,
  pendingSceneKeys,
  pendingSubscriptionID,
  subscriptionError,
  stashPerformersError,
  performerDetailError,
  performerScenesError,
  onSearchChange,
  onPageSizeChange,
  onReload,
  onRefreshAll,
  onToggle,
  onRefreshOne,
  onPrevPage,
  onNextPage,
  onOpenPerformer,
  onOpenTask,
  onBackToList,
  onPerformerSceneSearchChange,
  onPerformerSceneSourceChange,
  onPerformerSceneLibraryChange,
  onPerformerScenePageSizeChange,
  onPrevPerformerScenePage,
  onNextPerformerScenePage,
  onToggleSceneSelection,
  onSelectCurrentScenePage,
  onClearSceneSelection,
  onQueueSelectedScenes,
  onQueueScene
}: PerformersPageProps) {
  const { t } = useTranslation();
  const subscribedByID = useMemo(() => {
    return new Map(subscribedPerformers.map((item) => [item.performer.id, item]));
  }, [subscribedPerformers]);

  const performerSubscription = performerDetail ? (subscribedByID.get(performerDetail.performer.id) ?? null) : null;
  const currentPageKeys = performerScenes
    .filter((scene) => !scene.inLibrary && !scene.mojiTask && !pendingSceneKeys.includes(scene.key))
    .map((scene) => scene.key);
  const detailStashURL = performerDetail
    ? stashPerformerURL(performerDetail.performer.id, stashBaseURL)
    : null;
  const latestRelease = performerSubscription?.recentReleases[0] ?? null;

  if (selectedPerformerId) {
    return <PerformerDetailView>{(
      <section className="section-band">
        <div className="band-head">
          <div>
            <h2>{t("performerUi.detail")}</h2>
          </div>
          <button type="button" className="ghost-button" onClick={onBackToList}>
            <FontAwesomeIcon icon={faArrowLeft} /> {t("performerUi.back")}
          </button>
        </div>

        {performerDetailError || performerScenesError ? (
          <p className="settings-feedback tone-danger">
            {describeQueryError(performerDetailError || performerScenesError)}
          </p>
        ) : null}

        {performerDetail ? (
          <>
            <article className="performer-detail-card">
              <div className="performer-detail-hero">
                {performerImageURL(performerDetail.performer.imagePath, stashBaseURL) ? (
                  <img
                    className="avatar avatar--image performer-detail-hero__avatar"
                    src={performerImageURL(performerDetail.performer.imagePath, stashBaseURL) ?? ""}
                    alt={performerDetail.performer.name}
                    loading="lazy"
                    onError={(event) => { event.currentTarget.style.display = "none"; }}
                  />
                ) : (
                  <div className="avatar avatar--placeholder performer-detail-hero__avatar">
                    {performerInitials(performerDetail.performer.name)}
                  </div>
                )}

                <div className="performer-detail-hero__copy">
                  <div className="performer-detail-hero__topline">
                    <div className="performer-detail-hero__identity">
                      <strong className="performer-detail-hero__name">{performerDetail.performer.name}</strong>
                      <span
                        className={`performer-detail-hero__state ${performerDetail.performer.favorite ? "is-favorite" : ""}`}
                        title={t(performerDetail.performer.favorite ? "performerUi.favorite" : "performerUi.notFavorite")}
                      >
                        <FontAwesomeIcon icon={faHeart} />
                        {t(performerDetail.performer.favorite ? "performerUi.favorite" : "performerUi.notFavorite")}
                      </span>
                      <button
                        type="button"
                        className={`performer-detail-hero__state performer-detail-hero__subscription ${performerDetail.performer.subscribed ? "is-subscribed" : ""}`}
                        title={t(performerDetail.performer.subscribed ? "performerUi.removeSubscription" : "performerUi.addSubscription")}
                        aria-label={t(performerDetail.performer.subscribed ? "performerUi.unsubscribeName" : "performerUi.subscribeName", { name: performerDetail.performer.name })}
                        disabled={pendingSubscriptionID === performerDetail.performer.id}
                        onClick={() => onToggle(performerDetail.performer)}
                      >
                        <FontAwesomeIcon icon={faBookmark} />
                        Moji {t(performerDetail.performer.subscribed ? "performerUi.subscribed" : "performerUi.unsubscribed")}
                      </button>
                      <span className="performer-detail-hero__stashbox" title={performerDetail.matchedStashBox?.name ?? t("performerUi.noPreferredBox")}>
                        <span>{t("performerUi.preferredBox")}</span>
                        <strong>{performerDetail.matchedStashBox?.name ?? t("performerUi.unmatched")}</strong>
                      </span>
                    </div>

                    <div className="performer-detail-card__icons">
                      <button
                        type="button"
                        className="profile-action-icon"
                        title={t("performerUi.checkNow")}
                        aria-label={t("performerUi.checkName", { name: performerDetail.performer.name })}
                        disabled={pendingSubscriptionID === performerDetail.performer.id}
                        onClick={() => onRefreshOne(performerDetail.performer)}
                      >
                        <FontAwesomeIcon icon={faRotate} className={pendingSubscriptionID === performerDetail.performer.id ? "is-spinning" : undefined} />
                      </button>
                      {detailStashURL ? (
                        <a
                          className="profile-action-icon"
                          href={detailStashURL}
                          target="_blank"
                          rel="noreferrer"
                          title={t("performerUi.stashHome")}
                          aria-label={t("performerUi.stashHomeName", { name: performerDetail.performer.name })}
                        >
                          <FontAwesomeIcon icon={faPlayCircle} />
                        </a>
                      ) : null}
                    </div>
                  </div>

                  <div className="performer-detail-hero__description">
                    <p title={performerDetail.performer.aliasList.join(" / ")}>
                      <span>{t("performerUi.aliases")}</span>
                      {performerDetail.performer.aliasList.length > 0 ? performerDetail.performer.aliasList.join(" / ") : t("performerUi.none")}
                    </p>
                    <p title={performerDetail.disambiguation ?? ""}>
                      <span>{t("performerUi.description")}</span>
                      {performerDetail.disambiguation || t("performerUi.noDescription")}
                    </p>
                    <p>
                      <span>{t("performerUi.profile")}</span>
                      {[performerDetail.birthdate, performerDetail.country, performerDetail.heightCm ? `${performerDetail.heightCm} cm` : null]
                        .filter(Boolean)
                        .join(" · ") || t("performerUi.none")}
                    </p>
                    {latestRelease ? (
                      <p title={latestRelease.title}>
                        <span>{t("performerUi.latest")}</span>
                        {latestRelease.code || latestRelease.title} · {formatRelativeDate(latestRelease.date || latestRelease.seenAt)}
                      </p>
                    ) : null}
                  </div>

                  <dl className="performer-detail-metrics">
                    <div>
                      <dt>{t("performerUi.lastCheck")}</dt>
                      <dd title={formatDateTime(performerSubscription?.lastCheckedAt)}>
                        {formatRelative(performerSubscription?.lastCheckedAt) ?? t("performerUi.neverChecked")}
                      </dd>
                    </div>
                    <div className={performerSubscription?.lastError ? "tone-danger" : undefined}>
                      <dt>{t("performerUi.checkStatus")}</dt>
                      <dd title={performerSubscription?.lastError ?? ""}>
                        {t(performerSubscription?.lastError ? "performerUi.failed" : performerDetail.performer.subscribed ? "performerUi.normal" : "performerUi.unsubscribed")}
                      </dd>
                    </div>
                    <div>
                      <dt>{t("performerUi.pendingReleases")}</dt>
                      <dd>{performerSubscription?.pendingReleaseCount ?? 0}</dd>
                    </div>
                    <div>
                      <dt>{t("performerUi.processedReleases")}</dt>
                      <dd>{performerSubscription?.processedReleaseCount ?? 0}</dd>
                    </div>
                    <div>
                      <dt>Stash</dt>
                      <dd>{performerScenePage?.stashSceneCount ?? performerDetail.stashSceneCount}</dd>
                    </div>
                    <div>
                      <dt>StashBox</dt>
                      <dd>{performerScenePage?.stashBoxCount ?? performerDetail.stashBoxSceneCount}</dd>
                    </div>
                    <div>
                      <dt>{t("performerUi.deduped")}</dt>
                      <dd>{performerScenePage?.dedupedCount ?? performerDetail.dedupedSceneCount}</dd>
                    </div>
                  </dl>
                </div>
              </div>
            </article>

            <div className="toolbar-inline toolbar-inline--subscription">
              <input
                placeholder={t("performerUi.sceneSearch")}
                value={performerSceneSearch}
                onChange={(event) => onPerformerSceneSearchChange(event.target.value)}
              />
              <select value={performerSceneSourceFilter} onChange={(event) => onPerformerSceneSourceChange(event.target.value as SceneSourceFilter)}>
                <option value={SceneSourceFilter.All}>{t("performerUi.allSources")}</option>
                <option value={SceneSourceFilter.Stash}>Stash</option>
                <option value={SceneSourceFilter.Stashbox}>StashBox</option>
              </select>
              <select value={performerSceneLibraryFilter} onChange={(event) => onPerformerSceneLibraryChange(event.target.value as LibraryFilter)}>
                <option value={LibraryFilter.All}>{t("performerUi.allStates")}</option>
                <option value={LibraryFilter.InLibrary}>{t("performerUi.inLibrary")}</option>
                <option value={LibraryFilter.NotInLibrary}>{t("performerUi.notInLibrary")}</option>
              </select>
              <select value={performerScenePageSize} onChange={(event) => onPerformerScenePageSizeChange(Number(event.target.value))}>
                {SUBSCRIPTION_PAGE_SIZE_OPTIONS.map((size) => (
                  <option key={size} value={size}>
                    {t("performerUi.pageSize", { size })}
                  </option>
                ))}
              </select>
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

            <div className="settings-meta">
              <span>{t("performerUi.pageItems", { count: performerScenePage?.totalCount ?? 0 })}</span>
              <span>{t("performerUi.pagination", { page: performerScenePage?.page ?? 1, pages: performerScenePage?.totalPages ?? 0 })}</span>
              <span>{t("performerUi.state", { state: t(fetchingPerformerDetail || fetchingPerformerScenes ? "performerUi.loading" : "performerUi.ready") })}</span>
            </div>

            <div
              className="card-grid"
              style={{ marginTop: 16 }}
            >
              {performerScenes.map((scene) => {
                const selected = selectedSceneKeys.includes(scene.key);
                const pendingSingleQueue = pendingSceneKeys.includes(scene.key);
                const selectable = !scene.inLibrary && !scene.mojiTask && !pendingSingleQueue;
                return (
                  <article
                    key={scene.key}
                    className={`candidate-card performer-scene-card has-media ${selected ? "is-selected" : ""} ${selectable ? "" : "is-unselectable"}`}
                    onClick={() => {
                      if (selectable) onToggleSceneSelection(scene.key);
                    }}
                  >
                    <div className="performer-scene-card__media">
                      {scene.imageUrl ? (
                        <img
                          src={scene.imageUrl}
                          alt={scene.title || scene.code || "scene"}
                          loading="lazy"
                          onError={(event) => { event.currentTarget.style.display = "none"; }}
                          className="performer-scene-card__image"
                        />
                      ) : (
                        <div className="performer-scene-card__image-placeholder" aria-hidden="true">
                          <FontAwesomeIcon icon={faFilm} />
                        </div>
                      )}
                      <span
                        className="performer-scene-card__source-badge"
                        title={t("performerUi.source", { sources: scene.sourceLabels.join(" + ") })}
                      >
                        {performerSceneSourceLabel(scene)}
                      </span>
                    </div>
                    {selectable ? (
                      <button
                        type="button"
                        className="performer-scene-card__selector"
                        aria-pressed={selected}
                        aria-label={t(selected ? "performerUi.deselectName" : "performerUi.selectName", { name: scene.code || scene.title || scene.sourceSceneId })}
                        title={t(selected ? "performerUi.deselect" : "performerUi.selectScene")}
                        onClick={(event) => {
                          event.stopPropagation();
                          onToggleSceneSelection(scene.key);
                        }}
                      >
                        {selected ? <FontAwesomeIcon icon={faCheck} /> : null}
                      </button>
                    ) : null}
                    <div className="performer-scene-card__content">
                      <div className="performer-scene-card__status-row">
                        <h3 title={scene.code || scene.title || scene.sourceSceneId}>{scene.code || scene.title || scene.sourceSceneId}</h3>
                        <time className="performer-scene-card__date" dateTime={scene.date ?? undefined}>
                          {scene.date || t("performerUi.noDate")}
                        </time>
                      </div>
                      <p className="performer-scene-card__meta" title={scene.studioName || t("performerUi.unknownStudio")}>
                        {scene.studioName || t("performerUi.unknownStudio")}
                      </p>
                      <p className="performer-scene-card__title" title={scene.title || t("performerUi.noTitle")}>{scene.title || t("performerUi.noTitle")}</p>
                      <div className="performer-scene-card__counts">
                        <span
                          title={scene.performers.length > 0 ? t("performerUi.cast", { names: scene.performers.map((item) => item.name).join(" / ") }) : t("performerUi.castCount", { count: scene.performerCount })}
                          aria-label={t("performerUi.castCount", { count: scene.performerCount })}
                        >
                          <FontAwesomeIcon icon={faUsers} />
                          {scene.performerCount}
                        </span>
                        <span
                          title={scene.tags.length > 0 ? t("performerUi.tags", { names: scene.tags.map((item) => item.name).join(" / ") }) : t("performerUi.tagCount", { count: scene.tagCount })}
                          aria-label={t("performerUi.tagCount", { count: scene.tagCount })}
                        >
                          <FontAwesomeIcon icon={faTags} />
                          {scene.tagCount}
                        </span>
                      </div>
                      <div className="performer-scene-card__bottom-actions">
                        {scene.inLibrary ? (
                          <span className="status-chip performer-scene-card__business-state tone-success" title={t("performerUi.sceneInLibrary")}>
                            {t("performerUi.inLibrary")}
                          </span>
                        ) : scene.mojiTask ? (
                          <button
                            type="button"
                            className={`status-chip performer-scene-card__business-state performer-scene-card__task ${performerSceneTaskTone(scene.mojiTask)}`}
                            title={t("performerUi.taskTitle", { stage: performerSceneTaskLabel(scene.mojiTask), status: scene.mojiTask.stageStatus === TaskStageStatus.Blocked ? t("taskModel.states.blocked") : t("performerUi.normal") })}
                            onClick={(event) => {
                              event.stopPropagation();
                              onOpenTask(scene.mojiTask!.id);
                            }}
                          >
                            {performerSceneTaskLabel(scene.mojiTask)}
                          </button>
                        ) : (
                          <button
                            type="button"
                            className="status-chip performer-scene-card__business-state performer-scene-card__create-task tone-warn"
                            title={t("performerUi.createScene")}
                            disabled={pendingSingleQueue}
                            onClick={(event) => {
                              event.stopPropagation();
                              onQueueScene(scene);
                            }}
                          >
                            {pendingSingleQueue ? (
                              t("performerUi.creating")
                            ) : (
                              <>
                                <FontAwesomeIcon icon={faPlus} />
                                {t("performerUi.createTask")}
                              </>
                            )}
                          </button>
                        )}
                        {scene.url ? (
                          <a
                            className="profile-action-icon performer-scene-card__source-link"
                            href={scene.url}
                            target="_blank"
                            rel="noreferrer"
                            title={t("performerUi.original")}
                            aria-label={t("performerUi.originalName", { name: scene.code || scene.title || scene.sourceSceneId })}
                            onClick={(event) => event.stopPropagation()}
                          >
                            <FontAwesomeIcon icon={faArrowUpRightFromSquare} />
                          </a>
                        ) : null}
                      </div>
                    </div>
                  </article>
                );
              })}

              {!fetchingPerformerScenes && performerScenes.length === 0 ? (
                <article className="empty-card empty-card--wide">
                  <h3>{t("performerUi.noScenes")}</h3>
                  <p>{t("performerUi.noScenesDetail")}</p>
                </article>
              ) : null}
            </div>

            {performerScenePage && performerScenePage.totalPages > 1 ? (
              <div className="pagination-bar">
                <button type="button" className="ghost-button" disabled={!performerScenePage.hasPrevPage || fetchingPerformerScenes} onClick={onPrevPerformerScenePage}>
                  {t("performerUi.previous")}
                </button>
                <span className="status-chip tone-neutral">
                  {t("performerUi.page", { page: performerScenePage.page, pages: performerScenePage.totalPages })}
                </span>
                <button type="button" className="ghost-button" disabled={!performerScenePage.hasNextPage || fetchingPerformerScenes} onClick={onNextPerformerScenePage}>
                  {t("performerUi.next")}
                </button>
              </div>
            ) : null}
          </>
        ) : (
          <article className="empty-card empty-card--wide">
            <h3>{t(fetchingPerformerDetail ? "performerUi.detailLoading" : "performerUi.detailMissing")}</h3>
            <p>{t("performerUi.detailMissingHelp")}</p>
          </article>
        )}
      </section>
    )}</PerformerDetailView>;
  }

  return <PerformerListView>{(
    <section className="section-band">
      <div className="band-head">
        <div>
          <h2>{t("performerUi.list")}</h2>
        </div>
        <p className="band-note">
          {t("performerUi.listSummary", { count: stashPerformerPage?.totalCount ?? 0, subscribed: subscribedPerformers.length, loading: fetchingStashPerformers || fetchingSubscription ? t("performerUi.loadingSuffix") : "" })}
        </p>
      </div>

      <div className="toolbar-inline toolbar-inline--subscription">
        <input
          placeholder={t("performerUi.search")}
          value={subscriptionSearch}
          onChange={(event) => onSearchChange(event.target.value)}
        />
        <select value={subscriptionPageSize} onChange={(event) => onPageSizeChange(Number(event.target.value))}>
          {SUBSCRIPTION_PAGE_SIZE_OPTIONS.map((size) => (
            <option key={size} value={size}>
              {t("performerUi.pageSizeRows", { size })}
            </option>
          ))}
        </select>
        <button type="button" className="ghost-button" onClick={onReload}>
          {t("performerUi.refresh")}
        </button>
        <button
          type="button"
          className="ghost-button"
          disabled={refreshingSubscriptionNow || subscribedPerformers.length === 0}
          onClick={onRefreshAll}
        >
          {refreshingSubscriptionNow ? t("performerUi.checking") : t("performerUi.checkAll")}
        </button>
      </div>

      {stashPerformerPage && stashPerformerPage.totalPages > 1 ? (
        <div className="pagination-bar">
          <button type="button" className="ghost-button" disabled={!stashPerformerPage.hasPrevPage || fetchingStashPerformers} onClick={onPrevPage}>
            {t("performerUi.previous")}
          </button>
          <span className="status-chip tone-neutral">
            {t("performerUi.page", { page: stashPerformerPage.page, pages: stashPerformerPage.totalPages })}
          </span>
          <button type="button" className="ghost-button" disabled={!stashPerformerPage.hasNextPage || fetchingStashPerformers} onClick={onNextPage}>
            {t("performerUi.next")}
          </button>
        </div>
      ) : null}


      {subscriptionError || stashPerformersError ? (
        <p className="settings-feedback tone-danger">{describeQueryError(subscriptionError || stashPerformersError)}</p>
      ) : null}

      <div className="profile-grid">
        {stashPerformers.length === 0 && !fetchingStashPerformers ? (
          <article className="empty-card empty-card--wide">
            <h3>{t("performerUi.noPerformers")}</h3>
            <p>{t("performerUi.noPerformersDetail")}</p>
          </article>
        ) : null}
        {stashPerformers.map((performer, index) => {
          const subscriptionEntry = subscribedByID.get(performer.id) ?? null;
          const latestRelease = subscriptionEntry?.recentReleases[0] ?? null;
          const imageURL = performerImageURL(performer.imagePath, stashBaseURL);
          const stashURL = stashPerformerURL(performer.id, stashBaseURL);

          return (
            <article
              key={performer.id}
              className="profile-card"
              style={{ animationDelay: `${index * 80}ms`, cursor: "pointer" }}
              onClick={() => onOpenPerformer(performer.id)}
            >
              {imageURL ? (
                <div className="avatar avatar--frame">
                  <span className="avatar__fallback" aria-hidden="true">{performerInitials(performer.name)}</span>
                  <img
                    className="avatar__image"
                    src={imageURL}
                    alt={performer.name}
                    loading="lazy"
                    onError={(event) => { event.currentTarget.style.display = "none"; }}
                  />
                </div>
              ) : (
                <div className="avatar avatar--placeholder">{performerInitials(performer.name)}</div>
              )}
              <div className="profile-card__body">
                <div className="profile-card__head">
                  <div>
                    <h3>{performer.name}</h3>
                  </div>
                  <div className="profile-card__icons" onClick={(event) => event.stopPropagation()}>
                    {performer.favorite ? (
                      <span className="profile-icon profile-icon--favorite is-active" title={t("performerUi.favorite")} aria-label={t("performerUi.favorite")}>
                        <FontAwesomeIcon icon={faHeart} />
                      </span>
                    ) : null}
                    <button
                      type="button"
                      className={`profile-icon profile-icon--subscribe ${performer.subscribed ? "is-active" : ""}`}
                      title={t(performer.subscribed ? "performerUi.unsubscribe" : "performerUi.subscribe")}
                      aria-label={t(performer.subscribed ? "performerUi.unsubscribe" : "performerUi.subscribe")}
                      disabled={pendingSubscriptionID === performer.id}
                      onClick={() => onToggle(performer)}
                    >
                      <FontAwesomeIcon icon={faBookmark} />
                    </button>
                  </div>
                </div>
                <dl className="profile-facts">
                  <div>
                    <dt>{t("performerUi.works")}</dt>
                    <dd>{performer.sceneCount}</dd>
                  </div>
                </dl>
                <p className="profile-note">
                  {latestRelease
                    ? t("performerUi.recent", { release: latestRelease.code || latestRelease.title, date: formatRelativeDate(latestRelease.date || latestRelease.seenAt) })
                    : performer.subscribed
                      ? t("performerUi.waitingFirst")
                      : t("performerUi.notSubscribed")}
                </p>
                <div className="profile-card__footer">
                  <p className="profile-check-meta">{t("performerUi.lastChecked", { time: formatDateTime(subscriptionEntry?.lastCheckedAt) })}</p>
                  <div className="profile-actions" onClick={(event) => event.stopPropagation()}>
                    <button
                      type="button"
                      className="profile-action-icon"
                      title={t("performerUi.checkNow")}
                      aria-label={t("performerUi.checkName", { name: performer.name })}
                      disabled={pendingSubscriptionID === performer.id}
                      onClick={() => onRefreshOne(performer)}
                    >
                      <FontAwesomeIcon icon={faRotate} className={pendingSubscriptionID === performer.id ? "is-spinning" : undefined} />
                    </button>
                    {stashURL ? (
                      <a
                        className="profile-action-icon"
                        href={stashURL}
                        target="_blank"
                        rel="noreferrer"
                        title={t("performerUi.stashHome")}
                        aria-label={t("performerUi.stashHomeName", { name: performer.name })}
                      >
                        <FontAwesomeIcon icon={faPlayCircle} />
                      </a>
                    ) : null}
                  </div>
                </div>
              </div>
            </article>
          );
        })}
      </div>

      {stashPerformerPage && stashPerformerPage.totalPages > 1 ? (
        <div className="pagination-bar">
          <button type="button" className="ghost-button" disabled={!stashPerformerPage.hasPrevPage || fetchingStashPerformers} onClick={onPrevPage}>
            {t("performerUi.previous")}
          </button>
          <span className="status-chip tone-neutral">
            {t("performerUi.page", { page: stashPerformerPage.page, pages: stashPerformerPage.totalPages })}
          </span>
          <button type="button" className="ghost-button" disabled={!stashPerformerPage.hasNextPage || fetchingStashPerformers} onClick={onNextPage}>
            {t("performerUi.next")}
          </button>
        </div>
      ) : null}
    </section>
  )}</PerformerListView>;
}
