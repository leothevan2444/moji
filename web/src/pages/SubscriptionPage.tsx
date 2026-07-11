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
  type DashboardDocumentQuery,
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
type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;

function performerSceneTaskLabel(task: NonNullable<StashPerformerSceneEntry["mojiTask"]>) {
  if (task.stage === TaskStage.Downloading && task.progress > 0) {
    return `下载 ${Math.round(task.progress * 100)}%`;
  }
  if (task.stageStatus === TaskStageStatus.Blocked) {
    return task.stageStatusLabel;
  }
  return task.stageLabel;
}

function performerSceneTaskTone(task: NonNullable<StashPerformerSceneEntry["mojiTask"]>) {
  if (task.stageStatus === TaskStageStatus.Blocked) return "tone-danger";
  if (task.stage === TaskStage.Completed) return "tone-success";
  return "tone-info";
}

function performerSceneSourceLabel(scene: StashPerformerSceneEntry) {
  if (scene.hasStashSource && scene.hasStashBoxSource) return "双源";
  if (scene.hasStashBoxSource) return "StashBox";
  return "Stash";
}

interface SubscriptionPageProps {
  runtimeSettings: RuntimeSettings | null;
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

export function SubscriptionPage({
  runtimeSettings,
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
}: SubscriptionPageProps) {
  const subscribedByID = useMemo(() => {
    return new Map(subscribedPerformers.map((item) => [item.performer.id, item]));
  }, [subscribedPerformers]);

  const performerSubscription = performerDetail ? (subscribedByID.get(performerDetail.performer.id) ?? null) : null;
  const currentPageKeys = performerScenes
    .filter((scene) => !scene.inLibrary && !scene.mojiTask && !pendingSceneKeys.includes(scene.key))
    .map((scene) => scene.key);
  const detailStashURL = performerDetail
    ? stashPerformerURL(performerDetail.performer.id, runtimeSettings?.stash.url)
    : null;
  const latestRelease = performerSubscription?.recentReleases[0] ?? null;

  if (selectedPerformerId) {
    return (
      <section className="section-band">
        <div className="band-head">
          <div>
            <h2>演员详情</h2>
          </div>
          <button type="button" className="ghost-button" onClick={onBackToList}>
            <FontAwesomeIcon icon={faArrowLeft} /> 返回列表
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
                {performerImageURL(performerDetail.performer.imagePath, runtimeSettings?.stash.url) ? (
                  <img
                    className="avatar avatar--image performer-detail-hero__avatar"
                    src={performerImageURL(performerDetail.performer.imagePath, runtimeSettings?.stash.url) ?? ""}
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
                        title={performerDetail.performer.favorite ? "Stash 已收藏" : "Stash 未收藏"}
                      >
                        <FontAwesomeIcon icon={faHeart} />
                        Stash {performerDetail.performer.favorite ? "已收藏" : "未收藏"}
                      </span>
                      <button
                        type="button"
                        className={`performer-detail-hero__state performer-detail-hero__subscription ${performerDetail.performer.subscribed ? "is-subscribed" : ""}`}
                        title={performerDetail.performer.subscribed ? "取消 Moji 订阅" : "添加 Moji 订阅"}
                        aria-label={performerDetail.performer.subscribed ? `取消订阅 ${performerDetail.performer.name}` : `订阅 ${performerDetail.performer.name}`}
                        disabled={pendingSubscriptionID === performerDetail.performer.id}
                        onClick={() => onToggle(performerDetail.performer)}
                      >
                        <FontAwesomeIcon icon={faBookmark} />
                        Moji {performerDetail.performer.subscribed ? "已订阅" : "未订阅"}
                      </button>
                      <span className="performer-detail-hero__stashbox" title={performerDetail.matchedStashBox?.name ?? "未匹配到首选 StashBox"}>
                        <span>首选 StashBox</span>
                        <strong>{performerDetail.matchedStashBox?.name ?? "未匹配"}</strong>
                      </span>
                    </div>

                    <div className="performer-detail-card__icons">
                      <button
                        type="button"
                        className="profile-action-icon"
                        title="立即检查"
                        aria-label={`立即检查 ${performerDetail.performer.name}`}
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
                          title="前往 Stash 主页"
                          aria-label={`前往 ${performerDetail.performer.name} 的 Stash 主页`}
                        >
                          <FontAwesomeIcon icon={faPlayCircle} />
                        </a>
                      ) : null}
                    </div>
                  </div>

                  <div className="performer-detail-hero__description">
                    <p title={performerDetail.performer.aliasList.join(" / ")}>
                      <span>别名</span>
                      {performerDetail.performer.aliasList.length > 0 ? performerDetail.performer.aliasList.join(" / ") : "暂无"}
                    </p>
                    <p title={performerDetail.disambiguation ?? ""}>
                      <span>说明</span>
                      {performerDetail.disambiguation || "无补充说明"}
                    </p>
                    <p>
                      <span>资料</span>
                      {[performerDetail.birthdate, performerDetail.country, performerDetail.heightCm ? `${performerDetail.heightCm} cm` : null]
                        .filter(Boolean)
                        .join(" · ") || "暂无"}
                    </p>
                    {latestRelease ? (
                      <p title={latestRelease.title}>
                        <span>最近发行</span>
                        {latestRelease.code || latestRelease.title} · {formatRelativeDate(latestRelease.date || latestRelease.seenAt)}
                      </p>
                    ) : null}
                  </div>

                  <dl className="performer-detail-metrics">
                    <div>
                      <dt>上次检查</dt>
                      <dd title={formatDateTime(performerSubscription?.lastCheckedAt)}>
                        {formatRelative(performerSubscription?.lastCheckedAt) ?? "尚未检查"}
                      </dd>
                    </div>
                    <div className={performerSubscription?.lastError ? "tone-danger" : undefined}>
                      <dt>检查状态</dt>
                      <dd title={performerSubscription?.lastError ?? ""}>
                        {performerSubscription?.lastError ? "检查失败" : performerDetail.performer.subscribed ? "正常" : "未订阅"}
                      </dd>
                    </div>
                    <div>
                      <dt>待处理发行</dt>
                      <dd>{performerSubscription?.pendingReleaseCount ?? 0}</dd>
                    </div>
                    <div>
                      <dt>已处理发行</dt>
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
                      <dt>去重后</dt>
                      <dd>{performerScenePage?.dedupedCount ?? performerDetail.dedupedSceneCount}</dd>
                    </div>
                  </dl>
                </div>
              </div>
            </article>

            <div className="toolbar-inline toolbar-inline--subscription">
              <input
                placeholder="搜索影片标题、番号、片商"
                value={performerSceneSearch}
                onChange={(event) => onPerformerSceneSearchChange(event.target.value)}
              />
              <select value={performerSceneSourceFilter} onChange={(event) => onPerformerSceneSourceChange(event.target.value as SceneSourceFilter)}>
                <option value={SceneSourceFilter.All}>全部来源</option>
                <option value={SceneSourceFilter.Stash}>Stash</option>
                <option value={SceneSourceFilter.Stashbox}>StashBox</option>
              </select>
              <select value={performerSceneLibraryFilter} onChange={(event) => onPerformerSceneLibraryChange(event.target.value as LibraryFilter)}>
                <option value={LibraryFilter.All}>全部状态</option>
                <option value={LibraryFilter.InLibrary}>已入库</option>
                <option value={LibraryFilter.NotInLibrary}>未入库</option>
              </select>
              <select value={performerScenePageSize} onChange={(event) => onPerformerScenePageSizeChange(Number(event.target.value))}>
                {SUBSCRIPTION_PAGE_SIZE_OPTIONS.map((size) => (
                  <option key={size} value={size}>
                    每页 {size}
                  </option>
                ))}
              </select>
              <button
                type="button"
                className="ghost-button"
                disabled={currentPageKeys.length === 0 || fetchingPerformerScenes}
                onClick={() => onSelectCurrentScenePage(currentPageKeys)}
              >
                全选本页（{currentPageKeys.length}）
              </button>
              <button
                type="button"
                className="ghost-button"
                disabled={selectedSceneKeys.length === 0 || queueingPerformerScenes}
                onClick={onClearSceneSelection}
              >
                清空选择
              </button>
              <button
                type="button"
                className="primary-button"
                disabled={selectedSceneKeys.length === 0 || queueingPerformerScenes || fetchingPerformerScenes}
                onClick={onQueueSelectedScenes}
              >
                {queueingPerformerScenes
                  ? `正在创建 ${selectedSceneKeys.length} 个任务…`
                  : selectedSceneKeys.length > 0
                    ? `创建下载任务（${selectedSceneKeys.length}）`
                    : "创建下载任务"}
              </button>
            </div>

            <div className="settings-meta">
              <span>当前页条目: {performerScenePage?.totalCount ?? 0}</span>
              <span>分页: {performerScenePage?.page ?? 1} / {performerScenePage?.totalPages ?? 0}</span>
              <span>状态: {fetchingPerformerDetail || fetchingPerformerScenes ? "加载中" : "已就绪"}</span>
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
                        title={`来源：${scene.sourceLabels.join(" + ")}`}
                      >
                        {performerSceneSourceLabel(scene)}
                      </span>
                    </div>
                    {selectable ? (
                      <button
                        type="button"
                        className="performer-scene-card__selector"
                        aria-pressed={selected}
                        aria-label={`${selected ? "取消选择" : "选择"} ${scene.code || scene.title || scene.sourceSceneId}`}
                        title={selected ? "取消选择" : "选择影片"}
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
                          {scene.date || "无日期"}
                        </time>
                      </div>
                      <p className="performer-scene-card__meta" title={scene.studioName || "未知工作室"}>
                        {scene.studioName || "未知工作室"}
                      </p>
                      <p className="performer-scene-card__title" title={scene.title || "无标题"}>{scene.title || "无标题"}</p>
                      <div className="performer-scene-card__counts">
                        <span
                          title={scene.performers.length > 0 ? `参演演员：${scene.performers.map((item) => item.name).join(" / ")}` : `${scene.performerCount} 位参演演员`}
                          aria-label={`${scene.performerCount} 位参演演员`}
                        >
                          <FontAwesomeIcon icon={faUsers} />
                          {scene.performerCount}
                        </span>
                        <span
                          title={scene.tags.length > 0 ? `标签：${scene.tags.map((item) => item.name).join(" / ")}` : `${scene.tagCount} 个标签`}
                          aria-label={`${scene.tagCount} 个标签`}
                        >
                          <FontAwesomeIcon icon={faTags} />
                          {scene.tagCount}
                        </span>
                      </div>
                      <div className="performer-scene-card__bottom-actions">
                        {scene.inLibrary ? (
                          <span className="status-chip performer-scene-card__business-state tone-success" title="影片已入库">
                            已入库
                          </span>
                        ) : scene.mojiTask ? (
                          <button
                            type="button"
                            className={`status-chip performer-scene-card__business-state performer-scene-card__task ${performerSceneTaskTone(scene.mojiTask)}`}
                            title={`${scene.mojiTask.stageLabel} · ${scene.mojiTask.stageStatusLabel}，点击查看任务`}
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
                            title="为当前影片创建下载任务"
                            disabled={pendingSingleQueue}
                            onClick={(event) => {
                              event.stopPropagation();
                              onQueueScene(scene);
                            }}
                          >
                            {pendingSingleQueue ? (
                              "创建中…"
                            ) : (
                              <>
                                <FontAwesomeIcon icon={faPlus} />
                                创建任务
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
                            title="查看原始页"
                            aria-label={`查看 ${scene.code || scene.title || scene.sourceSceneId} 的原始页`}
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
                  <h3>没有匹配影片</h3>
                  <p>调整搜索词或筛选条件后再试。</p>
                </article>
              ) : null}
            </div>

            {performerScenePage && performerScenePage.totalPages > 1 ? (
              <div className="pagination-bar">
                <button type="button" className="ghost-button" disabled={!performerScenePage.hasPrevPage || fetchingPerformerScenes} onClick={onPrevPerformerScenePage}>
                  上一页
                </button>
                <span className="status-chip tone-neutral">
                  第 {performerScenePage.page} / {performerScenePage.totalPages} 页
                </span>
                <button type="button" className="ghost-button" disabled={!performerScenePage.hasNextPage || fetchingPerformerScenes} onClick={onNextPerformerScenePage}>
                  下一页
                </button>
              </div>
            ) : null}
          </>
        ) : (
          <article className="empty-card empty-card--wide">
            <h3>{fetchingPerformerDetail ? "演员详情加载中" : "未找到演员详情"}</h3>
            <p>可以返回列表后重新选择演员。</p>
          </article>
        )}
      </section>
    );
  }

  return (
    <section className="section-band">
      <div className="band-head">
        <div>
          <h2>演员列表</h2>
        </div>
        <p className="band-note">
          演员数 {stashPerformerPage?.totalCount ?? 0} · 已订阅 {subscribedPerformers.length}{fetchingStashPerformers || fetchingSubscription ? " · 加载中" : ""}
        </p>
      </div>

      <div className="toolbar-inline toolbar-inline--subscription">
        <input
          placeholder="按名称或别名搜索 Stash 演员"
          value={subscriptionSearch}
          onChange={(event) => onSearchChange(event.target.value)}
        />
        <select value={subscriptionPageSize} onChange={(event) => onPageSizeChange(Number(event.target.value))}>
          {SUBSCRIPTION_PAGE_SIZE_OPTIONS.map((size) => (
            <option key={size} value={size}>
              每页 {size} 条
            </option>
          ))}
        </select>
        <button type="button" className="ghost-button" onClick={onReload}>
          刷新列表
        </button>
        <button
          type="button"
          className="ghost-button"
          disabled={refreshingSubscriptionNow || subscribedPerformers.length === 0}
          onClick={onRefreshAll}
        >
          {refreshingSubscriptionNow ? "检查中..." : "检查全部演员"}
        </button>
      </div>

      {stashPerformerPage && stashPerformerPage.totalPages > 1 ? (
        <div className="pagination-bar">
          <button type="button" className="ghost-button" disabled={!stashPerformerPage.hasPrevPage || fetchingStashPerformers} onClick={onPrevPage}>
            上一页
          </button>
          <span className="status-chip tone-neutral">
            第 {stashPerformerPage.page} / {stashPerformerPage.totalPages} 页
          </span>
          <button type="button" className="ghost-button" disabled={!stashPerformerPage.hasNextPage || fetchingStashPerformers} onClick={onNextPage}>
            下一页
          </button>
        </div>
      ) : null}


      {subscriptionError || stashPerformersError ? (
        <p className="settings-feedback tone-danger">{describeQueryError(subscriptionError || stashPerformersError)}</p>
      ) : null}

      <div className="profile-grid">
        {stashPerformers.length === 0 && !fetchingStashPerformers ? (
          <article className="empty-card empty-card--wide">
            <h3>没有找到匹配的演员</h3>
            <p>可以尝试修改关键词，或先确认 Stash 已正确返回演员数据。</p>
          </article>
        ) : null}
        {stashPerformers.map((performer, index) => {
          const subscriptionEntry = subscribedByID.get(performer.id) ?? null;
          const latestRelease = subscriptionEntry?.recentReleases[0] ?? null;
          const imageURL = performerImageURL(performer.imagePath, runtimeSettings?.stash.url);
          const stashURL = stashPerformerURL(performer.id, runtimeSettings?.stash.url);

          return (
            <article
              key={performer.id}
              className="profile-card"
              style={{ animationDelay: `${index * 80}ms`, cursor: "pointer" }}
              onClick={() => onOpenPerformer(performer.id)}
            >
              {imageURL ? (
                <img className="avatar avatar--image" src={imageURL} alt={performer.name} loading="lazy" onError={(event) => { event.currentTarget.style.display = "none"; }} />
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
                      <span className="profile-icon profile-icon--favorite is-active" title="Stash 已收藏" aria-label="Stash 已收藏">
                        <FontAwesomeIcon icon={faHeart} />
                      </span>
                    ) : null}
                    <button
                      type="button"
                      className={`profile-icon profile-icon--subscribe ${performer.subscribed ? "is-active" : ""}`}
                      title={performer.subscribed ? "取消订阅" : "订阅"}
                      aria-label={performer.subscribed ? "取消订阅" : "订阅"}
                      disabled={pendingSubscriptionID === performer.id}
                      onClick={() => onToggle(performer)}
                    >
                      <FontAwesomeIcon icon={faBookmark} />
                    </button>
                  </div>
                </div>
                <dl className="profile-facts">
                  <div>
                    <dt>作品</dt>
                    <dd>{performer.sceneCount}</dd>
                  </div>
                </dl>
                <p className="profile-note">
                  {latestRelease
                    ? `最近记录: ${latestRelease.code || latestRelease.title} · ${formatRelativeDate(latestRelease.date || latestRelease.seenAt)}`
                    : performer.subscribed
                      ? "已订阅，等待首次检查结果。"
                      : "尚未订阅。"}
                </p>
                <div className="profile-card__footer">
                  <p className="profile-check-meta">最近检查：{formatDateTime(subscriptionEntry?.lastCheckedAt)}</p>
                  <div className="profile-actions" onClick={(event) => event.stopPropagation()}>
                    <button
                      type="button"
                      className="profile-action-icon"
                      title="立即检查"
                      aria-label={`立即检查 ${performer.name}`}
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
                        title="前往 Stash 主页"
                        aria-label={`前往 ${performer.name} 的 Stash 主页`}
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
            上一页
          </button>
          <span className="status-chip tone-neutral">
            第 {stashPerformerPage.page} / {stashPerformerPage.totalPages} 页
          </span>
          <button type="button" className="ghost-button" disabled={!stashPerformerPage.hasNextPage || fetchingStashPerformers} onClick={onNextPage}>
            下一页
          </button>
        </div>
      ) : null}
    </section>
  );
}
