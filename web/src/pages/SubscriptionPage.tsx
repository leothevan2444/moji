import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faBookmark, faHeart } from "@fortawesome/free-solid-svg-icons";
import { useMemo } from "react";
import { SUBSCRIPTION_PAGE_SIZE_OPTIONS } from "../constants";
import { describeQueryError } from "../services/queryError";
import {
  formatDateTime,
  formatRelativeDate,
  performerImageURL,
  performerInitials
} from "../utils";
import type {
  DashboardDocumentQuery,
  StashPerformersQuery,
  SubscribedPerformersQuery
} from "../graphql/generated/graphql";

type StashPerformerEntry = StashPerformersQuery["stashPerformers"]["items"][number];
type StashPerformerPage = StashPerformersQuery["stashPerformers"];
type SubscribedPerformerEntry = SubscribedPerformersQuery["subscribedPerformers"][number];
type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;

interface SubscriptionPageProps {
  runtimeSettings: RuntimeSettings | null;
  stashPerformerPage: StashPerformerPage | null;
  stashPerformers: StashPerformerEntry[];
  subscribedPerformers: SubscribedPerformerEntry[];
  fetchingStashPerformers: boolean;
  fetchingSubscription: boolean;
  refreshingSubscriptionNow: boolean;
  subscriptionSearch: string;
  subscriptionPageSize: number;
  pendingSubscriptionID: string | null;
  subscriptionError: Error | null;
  stashPerformersError: Error | null;
  onSearchChange: (value: string) => void;
  onPageSizeChange: (size: number) => void;
  onReload: () => void;
  onRefreshAll: () => void;
  onToggle: (performer: StashPerformerEntry) => void;
  onRefreshOne: (performer: StashPerformerEntry) => void;
  onPrevPage: () => void;
  onNextPage: () => void;
}

export function SubscriptionPage({
  runtimeSettings,
  stashPerformerPage,
  stashPerformers,
  subscribedPerformers,
  fetchingStashPerformers,
  fetchingSubscription,
  refreshingSubscriptionNow,
  subscriptionSearch,
  subscriptionPageSize,
  pendingSubscriptionID,
  subscriptionError,
  stashPerformersError,
  onSearchChange,
  onPageSizeChange,
  onReload,
  onRefreshAll,
  onToggle,
  onRefreshOne,
  onPrevPage,
  onNextPage
}: SubscriptionPageProps) {
  const subscribedByID = useMemo(() => {
    return new Map(subscribedPerformers.map((item) => [item.performer.id, item]));
  }, [subscribedPerformers]);

  return (
    <section className="section-band">
      <div className="band-head">
        <div>
          <p className="section-kicker">订阅</p>
          <h2>订阅更新</h2>
        </div>
      </div>

      <div className="toolbar-inline toolbar-inline--subscription">
        <input
          placeholder="按名称或别名搜索 Stash performer"
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
          {refreshingSubscriptionNow ? "检查中..." : "检查全部订阅"}
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
            <h3>没有找到匹配的 performer</h3>
            <p>可以尝试修改关键词，或先确认 Stash 已正确返回 performer 数据。</p>
          </article>
        ) : null}
        {stashPerformers.map((performer, index) => {
          const subscriptionEntry = subscribedByID.get(performer.id) ?? null;
          const latestRelease = subscriptionEntry?.recentReleases[0] ?? null;
          const imageURL = performerImageURL(performer.imagePath, runtimeSettings?.stash.url);

          return (
            <article key={performer.id} className="profile-card" style={{ animationDelay: `${index * 80}ms` }}>
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
                  <div className="profile-card__icons">
                    {performer.favorite ? (
                      <span
                        className="profile-icon profile-icon--favorite is-active"
                        title="Stash 已收藏"
                        aria-label="Stash 已收藏"
                      >
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
                <div className="profile-actions">
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
          <button
            type="button"
            className="ghost-button"
            disabled={!stashPerformerPage.hasPrevPage || fetchingStashPerformers}
            onClick={onPrevPage}
          >
            上一页
          </button>
          <span className="status-chip tone-neutral">
            第 {stashPerformerPage.page} / {stashPerformerPage.totalPages} 页
          </span>
          <button
            type="button"
            className="ghost-button"
            disabled={!stashPerformerPage.hasNextPage || fetchingStashPerformers}
            onClick={onNextPage}
          >
            下一页
          </button>
        </div>
      ) : null}
    </section>
  );
}
