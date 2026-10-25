import { useId, type ReactNode } from "react";

export function PerformerToolbar({ expanded, renderControls, children }: {
  expanded: boolean;
  renderControls: (panelId: string) => ReactNode;
  children: ReactNode;
}) {
  const panelId = useId();
  return (
    <div className={`performer-toolbar${expanded ? " is-selecting" : ""}`}>
      <div className="performer-toolbar__main">{renderControls(panelId)}</div>
      <div id={panelId} className="performer-toolbar__collapse" aria-hidden={!expanded} inert={!expanded}>
        <div className="performer-toolbar__collapse-inner">
          <div className="performer-subtoolbar">{children}</div>
        </div>
      </div>
    </div>
  );
}
