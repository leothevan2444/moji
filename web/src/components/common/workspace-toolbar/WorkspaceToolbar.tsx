import { workspaceToolbarIcons } from "./icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useId, useState } from "react";
import { ToolbarRow } from "./ToolbarRow";
import type { ToolbarContent } from "./types";
import "../../../styles/workspace-toolbar.scss";

export function WorkspaceToolbar({ selecting, toggleLabel, toggleDisabled = false, onToggleSelection, normal, selection, selectionLabel }: {
  selecting: boolean;
  toggleLabel: string;
  toggleDisabled?: boolean;
  onToggleSelection: () => void;
  normal: ToolbarContent;
  selection: ToolbarContent;
  selectionLabel: string;
}) {
  const id = useId();
  const [height, setHeight] = useState(38);
  return <div className={`workspace-toolbar${selecting ? " is-selecting" : ""}`}>
    <button type="button" className="ghost-button workspace-toolbar__toggle workspace-toolbar__action" aria-label={toggleLabel} title={toggleLabel} aria-pressed={selecting} aria-controls={id} disabled={toggleDisabled} onClick={onToggleSelection}>
      <span className="workspace-toolbar__toggle-icon"><FontAwesomeIcon icon={selecting ? workspaceToolbarIcons.exitSelection : workspaceToolbarIcons.enterSelection} aria-hidden="true" /></span>
    </button>
    <div id={id} className="workspace-toolbar__body" style={{ height }}>
      <div className={`workspace-toolbar__panel${!selecting ? " is-active" : ""}`} aria-hidden={selecting} inert={selecting}>
        <ToolbarRow {...normal} active={!selecting} onHeight={setHeight} />
      </div>
      <div className={`workspace-toolbar__panel${selecting ? " is-active" : ""}`} role="region" aria-label={selectionLabel} aria-hidden={!selecting} inert={!selecting}>
        <ToolbarRow {...selection} active={selecting} onHeight={setHeight} />
      </div>
    </div>
  </div>;
}
