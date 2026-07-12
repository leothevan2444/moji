import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

interface SortOption<V extends string> {
  value: V;
  label: string;
}

interface SortAndPaginationProps<V extends string> {
  sortValue: V;
  sortOptions: ReadonlyArray<SortOption<V>>;
  onSortChange: (value: V) => void;
  page: number;
  totalPages: number;
  total: number;
  onPrevPage: () => void;
  onNextPage: () => void;
  label?: string;
  extraContent?: ReactNode;
}

/**
 * 排序下拉 + 翻页控件。结果为 0 时仅显示排序下拉（page/totalPages 退化）。
 */
export function SortAndPagination<V extends string>({
  sortValue,
  sortOptions,
  onSortChange,
  page,
  totalPages,
  total,
  onPrevPage,
  onNextPage,
  label,
  extraContent
}: SortAndPaginationProps<V>) {
  const { t } = useTranslation();
  const hasResults = total > 0;
  const safeTotalPages = Math.max(totalPages, 1);

  return (
    <div className="discovery-toolbar">
      <div className="discovery-toolbar__sort">
        <span>{t("discoverUi.toolbar.sort")}</span>
        <select
          value={sortValue}
          onChange={(event) => onSortChange(event.target.value as V)}
          aria-label={t("discoverUi.toolbar.sortAria")}
        >
          {sortOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {t(option.label)}
            </option>
          ))}
        </select>
      </div>

      {extraContent ? <div className="discovery-toolbar__extra">{extraContent}</div> : null}

      <div className="discovery-toolbar__spacer" />

      {hasResults && (
        <div className="discovery-toolbar__page">
          <span className="discovery-toolbar__page-summary">
            {t("discoverUi.toolbar.summary", { label: label ?? t("discoverUi.toolbar.results"), page, pages: safeTotalPages, total })}
          </span>
          <button
            type="button"
            className="discovery-toolbar__page-btn"
            onClick={onPrevPage}
            disabled={page <= 1}
            aria-label={t("discoverUi.toolbar.previous")}
          >
            ‹
          </button>
          <button
            type="button"
            className="discovery-toolbar__page-btn"
            onClick={onNextPage}
            disabled={page >= safeTotalPages}
            aria-label={t("discoverUi.toolbar.next")}
          >
            ›
          </button>
        </div>
      )}
    </div>
  );
}
