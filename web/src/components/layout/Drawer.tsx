import { ReactNode } from "react";
import type { DrawerKey } from "../../types";

interface DrawerProps {
  visibleDrawer: Exclude<DrawerKey, null>;
  closing: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
}

export function Drawer({ visibleDrawer, closing, title, onClose, children }: DrawerProps) {
  return (
    <div
      className={`drawer-scrim ${visibleDrawer === "task" ? "drawer-scrim--task" : "drawer-scrim--modal"} ${closing ? "is-closing" : ""}`}
      onClick={onClose}
    >
      <aside
        className={`drawer ${visibleDrawer === "task" ? "drawer--task" : "drawer--modal"} ${closing ? "is-closing" : ""}`}
        onClick={(event) => event.stopPropagation()}
      >
        <div className="drawer__head">
          <div>
            <h2>{title}</h2>
          </div>
          <button type="button" className="ghost-button" onClick={onClose}>
            关闭
          </button>
        </div>

        <div className="drawer-body">{children}</div>
      </aside>
    </div>
  );
}
