import { useTranslation } from "react-i18next";

export function CandidatePagination({ page, pages, disabled, onChange }: { page: number; pages: number; disabled: boolean; onChange: (page: number) => void }) {
  const { t } = useTranslation();
  if (pages <= 1) return null;
  const numbers = [...new Set([1, page - 1, page, page + 1, pages])].filter(value => value >= 1 && value <= pages).sort((a, b) => a - b);
  return <nav className="candidate-pagination" aria-label={t("candidateFilters.pagination")}>
    <button type="button" disabled={disabled || page <= 1} aria-label={t("discoverUi.toolbar.previous")} onClick={() => onChange(page - 1)}>‹</button>
    {numbers.map((value, index) => <span key={value}>{index > 0 && value - numbers[index - 1] > 1 ? <span className="candidate-pagination__gap">…</span> : null}<button type="button" aria-current={value === page ? "page" : undefined} disabled={disabled} aria-label={t("candidateFilters.page", { page: value })} onClick={() => onChange(value)}>{value}</button></span>)}
    <button type="button" disabled={disabled || page >= pages} aria-label={t("discoverUi.toolbar.next")} onClick={() => onChange(page + 1)}>›</button>
  </nav>;
}
