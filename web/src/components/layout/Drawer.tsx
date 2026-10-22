import { useId, useRef, type ReactNode } from "react";
import { useModalFocus } from "../../hooks/useModalFocus";
import type { DrawerKey } from "../../types";
import { useTranslation } from "react-i18next";

interface DrawerProps {
  visibleDrawer: Exclude<DrawerKey, null>;
  closing: boolean;
  title: string;
  titleIcon?: ReactNode;
  onClose: () => void;
  children: ReactNode;
  footer?: ReactNode;
  closeDisabled?: boolean;
}

export function Drawer({ visibleDrawer, closing, title, titleIcon, onClose, children, footer, closeDisabled = false }: DrawerProps) {
  const { t } = useTranslation();
  const panelRef = useRef<HTMLElement>(null);
  const titleId = useId();
  useModalFocus(panelRef, onClose);
  const isTaskDrawer = visibleDrawer === "task" || visibleDrawer === "task-resolution" || visibleDrawer === "task-queue";
  const variant = isTaskDrawer ? "task" : visibleDrawer === "confirm" ? "confirm" : "modal";
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
        className={`drawer drawer--${variant} ${closing ? "is-closing" : ""}`}
        onClick={(event) => event.stopPropagation()}
      >
        <div className="drawer__head">
          <div>
            <h2 id={titleId}>{titleIcon ? <span className="drawer__title-icon" aria-hidden="true">{titleIcon}</span> : null}{title}</h2>
          </div>
          <button type="button" className="ghost-button" disabled={closeDisabled} onClick={onClose}>
            {t("common.close")}
          </button>
        </div>

        <div className="drawer-body">{children}</div>
        {footer ? <div className="drawer__footer">{footer}</div> : null}
      </aside>
    </div>
  );
}
