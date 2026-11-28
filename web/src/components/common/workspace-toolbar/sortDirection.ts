import {workspaceToolbarIcons} from "./icons";
import type {TFunction} from "i18next";
import type {ToolbarAction} from "./types";

export function sortDirectionAction(t:TFunction, field:string, direction:"asc"|"desc", onChange:(direction:"asc"|"desc")=>void):ToolbarAction {
 const next=direction==="asc"?"desc":"asc";
 return {id:"sort-direction",icon:direction==="asc"?workspaceToolbarIcons.sortAsc:workspaceToolbarIcons.sortDesc,
  label:t("sortControls.direction",{field,current:t(`sortControls.${direction}`),next:t(`sortControls.${next}`)}),onClick:()=>onChange(next)};
}
