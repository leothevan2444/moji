interface ConfirmDeleteDrawerProps {
  taskLabel: string;
  destructive: boolean;
  pending: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}
import { useTranslation } from "react-i18next";

export function ConfirmDeleteDrawer({
  taskLabel,
  destructive,
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
            <p>{taskLabel}</p>
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
