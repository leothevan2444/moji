interface ConfirmDeleteDrawerProps {
  taskLabel?: string;
  count?: number;
  deletePolicy: "KEEP_ONLY" | "REMOVE_TORRENT" | "REMOVE_TORRENT_AND_FILES";
  pending: boolean;
  hasStashScenes: boolean;
  onConfirm: (deleteStashScene: boolean) => void;
  onCancel: () => void;
  onConfigure: () => void;
}
import { useTranslation } from "react-i18next";
import { useEffect, useState } from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faTriangleExclamation } from "@fortawesome/free-solid-svg-icons/faTriangleExclamation";

export function ConfirmDeleteDrawer({
  taskLabel,
  count,
  deletePolicy,
  pending,
  hasStashScenes,
  onConfirm,
  onCancel,
  onConfigure
}: ConfirmDeleteDrawerProps) {
  const { t } = useTranslation();
  const [deleteStashScene, setDeleteStashScene] = useState(false);
  const removeStashScene = hasStashScenes && deleteStashScene;
  useEffect(() => { if (!hasStashScenes) setDeleteStashScene(false); }, [hasStashScenes]);
  const stashChoice = <label className={`confirm-delete__stash${!hasStashScenes ? " confirm-delete__stash--disabled" : ""}`}>
    <input type="checkbox" checked={removeStashScene} disabled={pending || !hasStashScenes}
      onChange={event => setDeleteStashScene(event.target.checked)} />
    <span>{t("taskRoute.deleteStashScene")}</span>
  </label>;
  return (
    <div className="confirm-delete">
      {(count || taskLabel) ? <p className="confirm-delete__target">{count ? t("taskBatch.confirmDeleteCount", { count }) : taskLabel}</p> : null}
      {hasStashScenes ? stashChoice : <span title={t("taskRoute.noStashScene")} aria-label={t("taskRoute.noStashScene")} tabIndex={0} className="confirm-delete__unlinked">{stashChoice}</span>}
      <p className={`confirm-delete__policy${deletePolicy !== "KEEP_ONLY" ? " tone-danger" : ""}`}>
        {deletePolicy !== "KEEP_ONLY" ? <FontAwesomeIcon icon={faTriangleExclamation} /> : null}
        <span>{t(deletePolicy === "KEEP_ONLY" && removeStashScene ? "taskRoute.deleteTaskRecord" : `taskBatch.deletePolicies.${deletePolicy}`)}{" "}
          <button type="button" className="confirm-delete__configure" disabled={pending} onClick={onConfigure}>{t("taskRoute.configureDeletion")}</button>
        </span>
      </p>
      <div className="confirm-delete__actions">
        <button type="button" className="ghost-button" onClick={onCancel} disabled={pending}>
          {t("taskRoute.cancel")}
        </button>
        <button type="button" className="confirm-delete__confirm" onClick={() => onConfirm(removeStashScene)} disabled={pending} aria-busy={pending}>
          {pending ? t("taskRoute.deleting") : t("taskRoute.confirm")}
        </button>
      </div>
    </div>
  );
}
