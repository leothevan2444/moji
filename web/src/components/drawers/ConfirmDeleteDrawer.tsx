interface ConfirmDeleteDrawerProps {
  taskLabel?: string;
  count?: number;
  deletePolicy: "KEEP_ONLY" | "REMOVE_TORRENT" | "REMOVE_TORRENT_AND_FILES";
  pending: boolean;
  onConfirm: (deleteStashScene: boolean) => void;
  onCancel: () => void;
  onConfigure: () => void;
}
import { useTranslation } from "react-i18next";
import { useState } from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faTriangleExclamation } from "@fortawesome/free-solid-svg-icons/faTriangleExclamation";

export function ConfirmDeleteDrawer({
  taskLabel,
  count,
  deletePolicy,
  pending,
  onConfirm,
  onCancel,
  onConfigure
}: ConfirmDeleteDrawerProps) {
  const { t } = useTranslation();
  const [deleteStashScene, setDeleteStashScene] = useState(false);
  return (
    <div className="confirm-delete">
      {(count || taskLabel) ? <p className="confirm-delete__target">{count ? t("taskBatch.confirmDeleteCount", { count }) : taskLabel}</p> : null}
      <label className="confirm-delete__stash">
        <input type="checkbox" checked={deleteStashScene} disabled={pending}
          onChange={event => setDeleteStashScene(event.target.checked)} />
        <span>{t("taskRoute.deleteStashScene")}</span>
      </label>
      <p className={`confirm-delete__policy${deletePolicy !== "KEEP_ONLY" ? " tone-danger" : ""}`}>
        {deletePolicy !== "KEEP_ONLY" ? <FontAwesomeIcon icon={faTriangleExclamation} /> : null}
        <span>{t(deletePolicy === "KEEP_ONLY" && deleteStashScene ? "taskRoute.deleteTaskRecord" : `taskBatch.deletePolicies.${deletePolicy}`)}{" "}
          <button type="button" className="confirm-delete__configure" disabled={pending} onClick={onConfigure}>{t("taskRoute.configureDeletion")}</button>
        </span>
      </p>
      <div className="confirm-delete__actions">
        <button type="button" className="ghost-button" onClick={onCancel} disabled={pending}>
          {t("taskRoute.cancel")}
        </button>
        <button type="button" className="confirm-delete__confirm" onClick={() => onConfirm(deleteStashScene)} disabled={pending} aria-busy={pending}>
          {pending ? t("taskRoute.deleting") : t("taskRoute.confirm")}
        </button>
      </div>
    </div>
  );
}
