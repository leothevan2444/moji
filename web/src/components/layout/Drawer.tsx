import { useId, useRef, type ReactNode } from "react";
import { useModalFocus } from "../../hooks/useModalFocus";
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
  const panelRef = useRef<HTMLElement>(null);
  const titleId = useId();
  useModalFocus(panelRef, onClose);
  const isTaskDrawer = visibleDrawer === "task" || visibleDrawer === "task-resolution";
  return (
    <div
      className={`drawer-scrim ${isTaskDrawer ? "drawer-scrim--task" : "drawer-scrim--modal"} ${closing ? "is-closing" : ""}`}
      onClick={onClose}
    >
      <aside
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className={`drawer ${isTaskDrawer ? "drawer--task" : "drawer--modal"} ${closing ? "is-closing" : ""}`}
        onClick={(event) => event.stopPropagation()}
      >
        <div className="drawer__head">
          <div>
            <h2 id={titleId}>{title}</h2>
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
