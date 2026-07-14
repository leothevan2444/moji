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
    <div className="drawer-stack">
      <article className="drawer-card">
        <div className="drawer-card__head">
          <div>
            <h3>{t("taskRoute.confirmDelete")}</h3>
            <p>{count ? t("taskBatch.confirmDeleteCount", { count }) : taskLabel}</p>
            <p className={deletePolicy === "KEEP_ONLY" ? undefined : "tone-danger"}>
              {deletePolicy !== "KEEP_ONLY" ? <FontAwesomeIcon icon={faTriangleExclamation} /> : null} {t(`taskBatch.deletePolicies.${deletePolicy}`)}
            </p>
          </div>
        </div>
        <div className="settings-actions">
          <button
            type="button"
            className="ghost-button"
            onClick={onCancel}
            disabled={pending}
          >
            {t("taskRoute.cancel")}
          </button>
          <button
            type="button"
            className="task-ops__button task-ops__button--danger"
            onClick={onConfirm}
            disabled={pending}
          >
            {pending ? t("taskRoute.deleting") : t("taskRoute.confirm")}
          </button>
        </div>
      </article>
    </div>
  );
}
