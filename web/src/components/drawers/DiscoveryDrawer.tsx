import { formatBytes, formatDurationSeconds } from "../../utils";
import type {
  DiscoverScenesDocumentQuery,
  SearchDocumentQuery
} from "../../graphql/generated/graphql";

type DiscoverResult = DiscoverScenesDocumentQuery["discoverScenes"]["items"][number];
type DiscoverConnection = DiscoverScenesDocumentQuery["discoverScenes"];
type JackettResult = SearchDocumentQuery["jackettSearch"][number];

interface DiscoveryDrawerProps {
  mode: "stashbox" | "jackett";
  query: string;
  searching: boolean;
  error: Error | null;
  pendingAddId: string | null;
  discoverResult: DiscoverConnection | null;
  discoverItems: DiscoverResult[];
  jackettItems: JackettResult[];
  onQueueDiscovered: (result: DiscoverResult) => void;
  onAddJackett: (result: JackettResult) => void;
}

export function DiscoveryDrawer({
  mode,
  query,
  searching,
  error,
  pendingAddId,
  discoverResult,
  discoverItems,
  jackettItems,
  onQueueDiscovered,
  onAddJackett
}: DiscoveryDrawerProps) {
  const isStashBox = mode === "stashbox";
  const items = isStashBox ? discoverItems : jackettItems;

  return (
    <div className="drawer-stack">
      <article className="drawer-card">
        <div className="drawer-card__head">
          <div>
            <h3>{query || "未提供搜索词"}</h3>
            <p>
              {isStashBox
                ? discoverResult?.usedStashBox
                  ? `来源 ${discoverResult.usedStashBox.name} · 回退 ${discoverResult.fallbackCount} 次`
                  : "按 StashBox 优先顺序搜索"
                : "备用 Jackett 搜索结果"}
            </p>
          </div>
          {isStashBox && discoverResult?.usedStashBox ? (
            <span className="status-chip tone-info">{discoverResult.usedStashBox.name}</span>
          ) : null}
        </div>

        {error ? <p className="inline-error">{error.message}</p> : null}

        {searching ? (
          <p>搜索中…</p>
        ) : items.length === 0 ? (
          <div className="empty-card empty-card--wide">
            <h3>没有结果</h3>
            <p>{isStashBox ? "StashBox 没有返回匹配影片。" : "Jackett 没有返回可用候选。"}</p>
          </div>
        ) : (
          <div className="discovery-results">
            {isStashBox
              ? discoverItems.map((result) => (
                  <article key={result.key} className="candidate-card candidate-card--discovery">
                    {result.imageUrl ? (
                      <div className="candidate-card__poster candidate-card__poster--discovery">
                        <img src={result.imageUrl} alt={result.title} loading="lazy" />
                      </div>
                    ) : null}
                    <div className="candidate-card__content">
                      <div className="candidate-card__head">
                        <div>
                          <div className="candidate-card__title-row">
                            <h3>{result.title}</h3>
                            {result.durationSeconds ? (
                              <span className="candidate-card__meta-chip">
                                {formatDurationSeconds(result.durationSeconds)}
                              </span>
                            ) : null}
                          </div>
                          <p>
                            {[result.code, result.studioName, result.performerNames.slice(0, 3).join(" / ")]
                              .filter(Boolean)
                              .join(" · ") || "StashBox 影片"}
                          </p>
                        </div>
                      </div>
                      <div className="candidate-card__foot">
                        <span>{result.date || "无日期"}</span>
                        <div className="inline-actions">
                          {result.url ? (
                            <a href={result.url} target="_blank" rel="noreferrer">
                              原始页面
                            </a>
                          ) : null}
                          <button
                            type="button"
                            onClick={() => onQueueDiscovered(result)}
                            disabled={pendingAddId === result.key}
                          >
                            {pendingAddId === result.key ? "加入中" : "加入任务队列"}
                          </button>
                        </div>
                      </div>
                    </div>
                  </article>
                ))
              : jackettItems.map((result) => (
                  <article key={`${result.tracker}-${result.link}`} className="candidate-card">
                    <div className="candidate-card__head">
                      <div>
                        <h3>{result.title}</h3>
                        <p>
                          {result.tracker} · {formatBytes(Number(result.size) || 0)} · {result.seeders} seeders
                        </p>
                      </div>
                      <span className="status-chip tone-warn">Jackett</span>
                    </div>
                    <div className="candidate-card__foot">
                      <span>{result.publishDate || "无日期"}</span>
                      <div className="inline-actions">
                        <a href={result.link} target="_blank" rel="noreferrer">
                          原始链接
                        </a>
                        <button
                          type="button"
                          onClick={() => onAddJackett(result)}
                          disabled={pendingAddId === result.link}
                        >
                          {pendingAddId === result.link ? "添加中" : "创建任务"}
                        </button>
                      </div>
                    </div>
                  </article>
                ))}
          </div>
        )}
      </article>
    </div>
  );
}
