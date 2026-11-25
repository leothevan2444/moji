import { useTranslation } from "react-i18next";
import { SUBSCRIPTION_PAGE_SIZE_OPTIONS } from "../../constants";

export function PerformerPageSize({ value, onChange, scenes = false }: { value: number; onChange: (size: number) => void; scenes?: boolean }) {
  const { t } = useTranslation();
  return (
    <select className="pagination-bar__page-size" aria-label={t("performerWorkspace.pageSize")} value={value} onChange={event => onChange(Number(event.target.value))}>
      {SUBSCRIPTION_PAGE_SIZE_OPTIONS.map(size => <option key={size} value={size}>{t(scenes ? "performerUi.pageSize" : "performerUi.pageSizeRows", { size })}</option>)}
    </select>
  );
}
