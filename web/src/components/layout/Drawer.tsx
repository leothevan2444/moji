import { ReactNode } from "react";
import type { DrawerKey } from "../../types";
import { useTranslation } from "react-i18next";

interface DrawerProps {
  visibleDrawer: Exclude<DrawerKey, null>;
  closing: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
}

export function Drawer({ visibleDrawer, closing, title, onClose, children }: DrawerProps) {
  const { t } = useTranslation();
  const isTaskDrawer = visibleDrawer === "task" || visibleDrawer === "task-resolution";
  return (
    <div
      className={`drawer-scrim ${isTaskDrawer ? "drawer-scrim--task" : "drawer-scrim--modal"} ${closing ? "is-closing" : ""}`}
      onClick={onClose}
    >
      <aside
        className={`drawer ${isTaskDrawer ? "drawer--task" : "drawer--modal"} ${closing ? "is-closing" : ""}`}
        onClick={(event) => event.stopPropagation()}
      >
        <div className="drawer__head">
          <div>
            <h2>{title}</h2>
          </div>
          <button type="button" className="ghost-button" onClick={onClose}>
            {t("common.close")}
          </button>
        </div>

        <div className="drawer-body">{children}</div>
      </aside>
    </div>
  );
}
