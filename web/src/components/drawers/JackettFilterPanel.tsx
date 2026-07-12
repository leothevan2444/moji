import type { JackettIndexersDocumentQuery } from "../../graphql/generated/graphql";
import { useTranslation } from "react-i18next";

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
  const { t } = useTranslation();
  const selectedCount = enabledIds.length;

  if (!fetching && indexers.length === 0) {
    return (
      <details className="filter-panel" open>
        <summary>
          <span>{t("discoverUi.filters.title")}</span>
          <span className="filter-panel__count">{t("discoverUi.filters.disconnected")}</span>
        </summary>
        <div className="filter-panel__body">
          <p className="filter-panel__hint">
            {t("discoverUi.filters.disconnectedDetail")}
          </p>
        </div>
      </details>
    );
  }

  return (
    <details className="filter-panel">
      <summary>
        <span>{t("discoverUi.filters.title")}</span>
        <span className="filter-panel__count">{t("discoverUi.filters.selected", { count: selectedCount })}</span>
      </summary>
      <div className="filter-panel__body">
        {fetching ? (
          <p className="filter-panel__hint">{t("discoverUi.filters.loading")}</p>
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
                    title={disabled ? t("discoverUi.filters.disabled") : indexer.name}
                  >
                    {indexer.name}
                  </button>
                );
              })}
            </div>
            {selectedCount > 0 && (
              <button
                type="button"
                className="filter-panel__clear"
                onClick={onClear}
              >
                {t("discoverUi.filters.clear")}
              </button>
            )}
          </>
        )}
      </div>
    </details>
  );
}
