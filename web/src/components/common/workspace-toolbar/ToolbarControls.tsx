import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Link } from "react-router";
import type { ToolbarAction, ToolbarFilter } from "./types";

export function ActionControl({ action, measuring = false, menu = false }: { action: ToolbarAction; measuring?: boolean; menu?: boolean }) {
  const className = `${action.primary ? "primary-button" : "ghost-button"} workspace-toolbar__action${action.showLabel || menu ? " has-label" : ""}${action.danger ? " task-ops__button--danger" : ""}`;
  const content = <><FontAwesomeIcon icon={action.icon} aria-hidden="true" className={action.spinning ? "is-spinning" : undefined} />{action.showLabel || menu ? <span className="workspace-toolbar__action-label">{action.label}</span> : null}</>;
  const data = { "data-kind": "action", "data-control": action.id, "data-priority": action.primary ? "1" : "0" };
  if (measuring) return <span className={className} {...data}>{content}</span>;
  if (action.href) return <Link className={className} {...data} to={action.href} aria-label={action.label} title={action.title ?? action.label}>{content}</Link>;
  return <button type="button" className={className} {...data} onClick={action.onClick} aria-pressed={action.pressed} disabled={action.disabled} aria-label={action.label} title={action.title ?? action.label} aria-busy={action.busy || undefined}>{content}</button>;
}

export function FilterControl({ filter, measuring = false }: { filter: ToolbarFilter; measuring?: boolean }) {
  const control = <select className="workspace-toolbar__filter" data-kind="filter" data-control={filter.id} aria-label={measuring ? undefined : filter.label}
    title={measuring ? undefined : filter.label} value={filter.value} disabled={measuring} onChange={measuring ? undefined : event => filter.onChange(event.target.value)}>
    {filter.options.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
  </select>;
  return filter.accessory ? <span className="workspace-toolbar__sort" data-kind="filter" data-control={filter.id}>{control}<ActionControl action={filter.accessory} measuring={measuring} /></span> : control;
}
