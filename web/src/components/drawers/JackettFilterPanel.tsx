import type { JackettIndexersDocumentQuery } from "../../graphql/generated/graphql";

type Indexer = JackettIndexersDocumentQuery["jackettIndexers"][number];

interface JackettFilterPanelProps {
  indexers: Indexer[];
  fetching: boolean;
  enabledIds: string[];
  onToggle: (id: string) => void;
  onClear: () => void;
}

/**
 * Jackett 索引器多选过滤面板。indexer 数据由 useJackettIndexers 提供；
 * 未配置 Jackett 时后端返回空数组，前端据此渲染友好提示。
 */
export function JackettFilterPanel({
  indexers,
  fetching,
  enabledIds,
  onToggle,
  onClear
}: JackettFilterPanelProps) {
  const selectedCount = enabledIds.length;

  if (!fetching && indexers.length === 0) {
    return (
      <details className="filter-panel" open>
        <summary>
          <span>索引器筛选</span>
          <span className="filter-panel__count">未连接 Jackett</span>
        </summary>
        <div className="filter-panel__body">
          <p className="filter-panel__hint">
            当前没有可用的 Jackett 索引器。请到设置里检查 Jackett URL / API Key。
          </p>
        </div>
      </details>
    );
  }

  return (
    <details className="filter-panel">
      <summary>
        <span>索引器筛选</span>
        <span className="filter-panel__count">已选 {selectedCount} 个</span>
      </summary>
      <div className="filter-panel__body">
        {fetching ? (
          <p className="filter-panel__hint">正在加载索引器列表…</p>
        ) : (
          <>
            <div className="filter-panel__chips">
              {indexers.map((indexer) => {
                const active = enabledIds.includes(indexer.id);
                const disabled = !indexer.enabled;
                const className = [
                  "filter-panel__chip",
                  active ? "is-active" : "",
                  disabled ? "is-disabled" : ""
                ]
                  .filter(Boolean)
                  .join(" ");
                return (
                  <button
                    key={indexer.id}
                    type="button"
                    className={className}
                    disabled={disabled}
                    aria-pressed={active}
                    onClick={() => !disabled && onToggle(indexer.id)}
                    title={disabled ? "Jackett 端未配置此索引器" : indexer.name}
                  >
                    {indexer.name}
                  </button>
                );
              })}
            </div>
            {selectedCount > 0 && (
              <button
                type="button"
                className="filter-panel__hint"
                style={{
                  background: "transparent",
                  border: "none",
                  padding: 0,
                  cursor: "pointer",
                  textAlign: "left",
                  color: "var(--accent)"
                }}
                onClick={onClear}
              >
                清空筛选
              </button>
            )}
          </>
        )}
      </div>
    </details>
  );
}