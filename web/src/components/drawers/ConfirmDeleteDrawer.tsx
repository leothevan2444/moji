interface ConfirmDeleteDrawerProps {
  taskLabel?: string;
  count?: number;
  deletePolicy: "KEEP_ONLY" | "REMOVE_TORRENT" | "REMOVE_TORRENT_AND_FILES";
  pending: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faTriangleExclamation } from "@fortawesome/free-solid-svg-icons/faTriangleExclamation";

export function ConfirmDeleteDrawer({
  taskLabel,
  count,
  deletePolicy,
  pending,
  onConfirm,
  onCancel
}: ConfirmDeleteDrawerProps) {
  const { t } = useTranslation();
  return (
    <div className="confirm-delete">
      {(count || taskLabel) ? <p className="confirm-delete__target">{count ? t("taskBatch.confirmDeleteCount", { count }) : taskLabel}</p> : null}
      <p className={`confirm-delete__policy${deletePolicy !== "KEEP_ONLY" ? " tone-danger" : ""}`}>
        {deletePolicy !== "KEEP_ONLY" ? <FontAwesomeIcon icon={faTriangleExclamation} /> : null}
        <span>{t(`taskBatch.deletePolicies.${deletePolicy}`)}</span>
      </p>
      <div className="confirm-delete__actions">
        <button type="button" className="ghost-button" onClick={onCancel} disabled={pending}>
          {t("taskRoute.cancel")}
        </button>
        <button type="button" className="confirm-delete__confirm" onClick={onConfirm} disabled={pending} aria-busy={pending}>
          {pending ? t("taskRoute.deleting") : t("taskRoute.confirm")}
        </button>
      </div>
    </div>
  );
}
