import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faTriangleExclamation } from "@fortawesome/free-solid-svg-icons/faTriangleExclamation";
import { useTranslation } from "react-i18next";

export type PerformerBatchConfirmAction = "unsubscribe" | "refresh";

export function PerformerBatchConfirmDrawer({ action, count, pending, onConfirm, onCancel }: { action: PerformerBatchConfirmAction; count: number; pending: boolean; onConfirm: () => void; onCancel: () => void }) {
  const { t } = useTranslation();
  return <div className="drawer-stack">
    <article className="drawer-card">
      <div className="drawer-card__head"><div>
        <h3>{t(`performerBatch.confirm.${action}.title`, { count })}</h3>
        <p className={action === "unsubscribe" ? "tone-danger" : "tone-warn"}>
          <FontAwesomeIcon icon={faTriangleExclamation} /> {t(`performerBatch.confirm.${action}.detail`, { count })}
        </p>
      </div></div>
      <div className="settings-actions">
        <button type="button" className="ghost-button" disabled={pending} onClick={onCancel}>{t("performerBatch.cancel")}</button>
        <button type="button" className={action === "unsubscribe" ? "task-ops__button task-ops__button--danger" : "primary-button"} disabled={pending} onClick={onConfirm}>
          {pending ? t("performerBatch.processing") : t("performerBatch.confirmAction")}
        </button>
      </div>
    </article>
  </div>;
}
