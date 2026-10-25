import { useTranslation } from "react-i18next";
import { describeQueryError } from "../../services/queryError";

export function PerformerLoading({ detail = false }: { detail?: boolean; }) {
  const { t } = useTranslation();
  return <div className={`performer-loading ${detail ? "performer-loading--detail" : ""}`} role="status" aria-label={t("performerWorkspace.loading")}>
    {Array.from({ length: detail ? 1 : 6 }, (_, index) => <div className="performer-skeleton skeleton" key={index} aria-hidden="true"><span /><div><i /><i /><i /></div></div>)}
  </div>;
}

export function PerformerError({ error, onRetry }: { error: Error; onRetry?: () => void; }) {
  const { t } = useTranslation();
  return <div className="performer-query-error settings-feedback tone-danger" role="alert"><p>{describeQueryError(error)}</p>{onRetry ? <button type="button" className="ghost-button" onClick={onRetry}>{t("performerWorkspace.retry")}</button> : null}</div>;
}
