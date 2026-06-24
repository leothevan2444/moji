import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowLeft, faBookmark, faHeart } from "@fortawesome/free-solid-svg-icons";
import { useMemo } from "react";
import { SUBSCRIPTION_PAGE_SIZE_OPTIONS } from "../constants";
import { describeQueryError } from "../services/queryError";
import { formatDateTime, formatRelativeDate, performerImageURL, performerInitials } from "../utils";
import {
  LibraryFilter,
  SceneSourceFilter,
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
  subscriptionSearch: string;
  subscriptionPageSize: number;
  selectedPerformerId: string | null;
  performerSceneSearch: string;
  performerSceneSourceFilter: SceneSourceFilter;
  performerSceneLibraryFilter: LibraryFilter;
  performerScenePageSize: number;
  selectedSceneKeys: string[];
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
  subscriptionSearch,
  subscriptionPageSize,
  selectedPerformerId,
  performerSceneSearch,
  performerSceneSourceFilter,
  performerSceneLibraryFilter,
  performerScenePageSize,
  selectedSceneKeys,
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
  onBackToList,
  onPerformerSceneSearchChange,
  onPerformerSceneSourceChange,
  onPerformerSceneLibraryChange,
  onPerformerScenePageSizeChange,
  onPrevPerformerScenePage,
  onNextPerformerScenePage,
  onToggleSceneSelection,
  onSelectCurrentScenePage,
  onClearSceneSelection
}: SubscriptionPageProps) {
  const subscribedByID = useMemo(() => {
    return new Map(subscribedPerformers.map((item) => [item.performer.id, item]));
  }, [subscribedPerformers]);

  const performerSubscription = performerDetail ? (subscribedByID.get(performerDetail.performer.id) ?? null) : null;
  const currentPageKeys = performerScenes.map((scene) => scene.key);

  if (selectedPerformerId) {
    return (
      <section className="section-band">
        <div className="band-head">
          <div>
            <p className="section-kicker">演员</p>
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
            <article className="performer-detail-card" style={{ marginBottom: 16 }}>
              <div className="task-detail-hero">
                <div className="performer-detail-hero__main">
                  {performerImageURL(performerDetail.performer.imagePath, runtimeSettings?.stash.url) ? (
                    <img
                      className="avatar avatar--image performer-detail-hero__avatar"
                      src={performerImageURL(performerDetail.performer.imagePath, runtimeSettings?.stash.url) ?? ""}
                      alt={performerDetail.performer.name}
                      loading="lazy"
                    />
                  ) : (
                    <div className="avatar avatar--placeholder performer-detail-hero__avatar">
                      {performerInitials(performerDetail.performer.name)}
                    </div>
                  )}
                  <div className="task-detail-hero__copy performer-detail-hero__copy">
                    <strong className="performer-detail-hero__name">{performerDetail.performer.name}</strong>
                    <div className="performer-detail-hero__meta">
                      <span
                        className={`status-chip ${performerDetail.matchedStashBox ? "tone-info" : "tone-warn"}`}
                      >
                        {performerDetail.matchedStashBox
                          ? `首选 StashBox: ${performerDetail.matchedStashBox.name}`
                          : "未匹配到首选 StashBox"}
                      </span>
                    </div>
                    <div className="performer-detail-hero__section">
                      <span className="performer-detail-hero__label">别名</span>
                      <p>
                        {performerDetail.performer.aliasList.length > 0
                          ? performerDetail.performer.aliasList.join(" / ")
                          : "暂无别名"}
                      </p>
                    </div>
                    <div className="performer-detail-hero__section">
                      <span className="performer-detail-hero__label">说明</span>
                      <p>{performerDetail.disambiguation || "无补充说明"}</p>
                    </div>
                  </div>
                </div>
                <div className="performer-detail-card__icons">
                  {performerDetail.performer.favorite ? (
                    <span className="profile-icon profile-icon--favorite is-active" title="Stash 已收藏">
                      <FontAwesomeIcon icon={faHeart} />
                    </span>
                  ) : null}
                  <button
                    type="button"
                    className={`profile-icon profile-icon--subscribe ${performerDetail.performer.subscribed ? "is-active" : ""}`}
                    disabled={pendingSubscriptionID === performerDetail.performer.id}
                    onClick={() => onToggle(performerDetail.performer)}
                  >
                    <FontAwesomeIcon icon={faBookmark} />
                  </button>
                </div>
              </div>

              <div className="settings-meta" style={{ marginTop: 16 }}>
                <span>Stash 作品: {performerScenePage?.stashSceneCount ?? performerDetail.stashSceneCount}</span>
                <span>StashBox 作品: {performerScenePage?.stashBoxCount ?? performerDetail.stashBoxSceneCount}</span>
                <span>去重后: {performerScenePage?.dedupedCount ?? performerDetail.dedupedSceneCount}</span>
                <span>最近检查: {formatDateTime(performerSubscription?.lastCheckedAt)}</span>
              </div>

              <div className="profile-facts" style={{ marginTop: 16 }}>
                <div>
                  <dt>生日</dt>
                  <dd>{performerDetail.birthdate || "-"}</dd>
                </div>
                <div>
                  <dt>国家</dt>
                  <dd>{performerDetail.country || "-"}</dd>
                </div>
                <div>
                  <dt>眼睛</dt>
                  <dd>{performerDetail.eyeColor || "-"}</dd>
                </div>
                <div>
                  <dt>身高</dt>
                  <dd>{performerDetail.heightCm ? `${performerDetail.heightCm} cm` : "-"}</dd>
                </div>
              </div>
            </article>

            <div className="toolbar-inline toolbar-inline--subscription" style={{ gridTemplateColumns: "minmax(220px,1.3fr) repeat(4,minmax(120px,160px)) auto auto" }}>
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
              <button type="button" className="ghost-button" onClick={() => onSelectCurrentScenePage(currentPageKeys)}>
                全选本页
              </button>
              <button type="button" className="ghost-button" onClick={onClearSceneSelection}>
                清空选择
              </button>
              <span className="status-chip tone-neutral">已选 {selectedSceneKeys.length}</span>
            </div>

            <div className="settings-meta">
              <span>当前页条目: {performerScenePage?.totalCount ?? 0}</span>
              <span>分页: {performerScenePage?.page ?? 1} / {performerScenePage?.totalPages ?? 0}</span>
              <span>状态: {fetchingPerformerDetail || fetchingPerformerScenes ? "加载中" : "已就绪"}</span>
            </div>

            <div
              className="card-grid"
              style={{ gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))", marginTop: 16 }}
            >
              {performerScenes.map((scene) => {
                const selected = selectedSceneKeys.includes(scene.key);
                return (
                  <article key={scene.key} className="candidate-card">
                    {scene.imageUrl ? (
                      <img
                        src={scene.imageUrl}
                        alt={scene.title || scene.code || "scene"}
                        loading="lazy"
                        style={{ width: "100%", aspectRatio: "16 / 9", objectFit: "cover", borderRadius: 10, marginBottom: 12 }}
                      />
                    ) : null}
                    <div className="candidate-card__head">
                      <div>
                        <h3>{scene.code || scene.title || scene.sourceSceneId}</h3>
                        <p>{scene.title || "无标题"}</p>
                      </div>
                      <input type="checkbox" checked={selected} onChange={() => onToggleSceneSelection(scene.key)} />
                    </div>
                    <p style={{ marginTop: 10 }}>{scene.studioName || "未知片商"} · {scene.date || "无日期"}</p>
                    <div className="candidate-card__foot">
                      <div className="chip-row">
                        <span className="status-chip tone-info">{scene.sourceLabels.join(" + ")}</span>
                        <span className={`status-chip ${scene.inLibrary ? "tone-success" : "tone-warn"}`}>
                          {scene.inLibrary ? "已入库" : "未入库"}
                        </span>
                      </div>
                      {scene.url ? (
                        <a href={scene.url} target="_blank" rel="noreferrer">
                          查看原始页
                        </a>
                      ) : (
                        <span>{scene.hasStashBoxSource && !scene.inLibrary ? "等待入库" : "已归并"}</span>
                      )}
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
          <p className="section-kicker">演员</p>
          <h2>演员更新</h2>
        </div>
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

      <div className="settings-meta">
        <span>Stash 候选: {stashPerformerPage?.totalCount ?? 0}</span>
        <span>当前页: {stashPerformerPage?.page ?? 1} / {stashPerformerPage?.totalPages ?? 0}</span>
        <span>每页: {stashPerformerPage?.pageSize ?? subscriptionPageSize}</span>
        <span>已订阅: {subscribedPerformers.length}</span>
        <span>载入状态: {fetchingStashPerformers || fetchingSubscription ? "同步中" : "已就绪"}</span>
      </div>
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

          return (
            <article
              key={performer.id}
              className="profile-card"
              style={{ animationDelay: `${index * 80}ms`, cursor: "pointer" }}
              onClick={() => onOpenPerformer(performer.id)}
            >
              {imageURL ? (
                <img className="avatar avatar--image" src={imageURL} alt={performer.name} loading="lazy" />
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
                  <div>
                    <dt>检查</dt>
                    <dd>{formatDateTime(subscriptionEntry?.lastCheckedAt)}</dd>
                  </div>
                </dl>
                <p className="profile-note">
                  {subscriptionEntry?.lastError
                    ? `最近错误: ${subscriptionEntry.lastError}`
                    : latestRelease
                      ? `最近记录: ${latestRelease.code || latestRelease.title} · ${formatRelativeDate(latestRelease.date || latestRelease.seenAt)}`
                      : performer.subscribed
                        ? "已订阅，等待首次检查结果。"
                        : "尚未订阅。"}
                </p>
                <div className="profile-actions" onClick={(event) => event.stopPropagation()}>
                  <button
                    type="button"
                    className="ghost-button"
                    disabled={!performer.subscribed || pendingSubscriptionID === performer.id}
                    onClick={() => onRefreshOne(performer)}
                  >
                    立即检查
                  </button>
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
