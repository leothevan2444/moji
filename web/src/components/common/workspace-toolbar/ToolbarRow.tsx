import { workspaceToolbarIcons } from "./icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useCallback, useId, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ActionControl, FilterControl } from "./ToolbarControls";
import { ToolbarOverflow } from "./ToolbarOverflow";
import type { ToolbarContent } from "./types";

interface Plan { compact: boolean; actions: string[]; filters: string[] }
interface Width { id: string; kind: string; width: number; priority: number }

export function ToolbarRow({ search, filters = [], actions, summary, active, onHeight }: ToolbarContent & { active: boolean; onHeight: (height: number) => void }) {
  const { t } = useTranslation();
  const id = useId();
  const row = useRef<HTMLDivElement>(null);
  const full = useRef<HTMLDivElement>(null);
  const small = useRef<HTMLDivElement>(null);
  const more = useRef<HTMLButtonElement>(null);
  const [plan, setPlan] = useState<Plan>({ compact: false, actions: [], filters: [] });
  const [open, setOpen] = useState(false);
  const [restore, setRestore] = useState(false);
  const overflowing = plan.actions.length + plan.filters.length > 0;
  const signature = `${plan.compact}:${plan.actions.join(",")}:${plan.filters.join(",")}`;
  useLayoutEffect(() => {
    const update = () => {
      if (!row.current || !full.current || !small.current) return;
      const available = row.current.clientWidth - 2;
      const gap = parseFloat(getComputedStyle(row.current).columnGap) || 0;
      if (available <= 0) return;
      const widths = (node: HTMLElement): Width[] => [...node.children].map(child => {
        const control = child as HTMLElement;
        return {
          id: control.dataset.control ?? "",
          kind: control.dataset.kind ?? "",
          priority: Number(control.dataset.priority ?? 0),
          width: control.offsetWidth,
        };
      });
      const fits = (items: Width[], rows: number) => {
        let used = 0, lines = 1;
        for (const item of items) {
          const width = Math.min(item.width, available);
          if (used && used + gap + width > available) {
            lines++;
            used = width;
          } else {
            used += (used ? gap : 0) + width;
          }
        }
        return lines <= rows;
      };
      const wide = widths(full.current), compact = widths(small.current), next: Plan = { compact: false, actions: [], filters: [] };
      const visible = (items: Width[]) => items.filter(item => {
        if (item.kind === "more") return next.actions.length + next.filters.length > 0;
        if (item.kind === "action") return !next.actions.includes(item.id);
        if (item.kind === "filter") return !next.filters.includes(item.id);
        return true;
      });
      if (!fits(visible(wide), 1)) {
        next.compact = true;
        // Functions move first; filters remain available inline until the three-row budget is exhausted.
        for (const action of compact.filter(item => item.kind === "action").reverse().sort((a, b) => a.priority - b.priority)) {
          if (fits(visible(compact), 1)) break;
          next.actions.push(action.id);
        }
        for (const filter of compact.filter(item => item.kind === "filter").reverse()) {
          if (fits(visible(compact), 3)) break;
          next.filters.push(filter.id);
        }
      }
      const focused = document.activeElement instanceof HTMLElement ? document.activeElement.closest<HTMLElement>("[data-control]") : null;
      if (focused && row.current.contains(focused) && (next.actions.includes(focused.dataset.control ?? "") || next.filters.includes(focused.dataset.control ?? ""))) setRestore(true);
      setPlan(current => JSON.stringify(current) === JSON.stringify(next) ? current : next);
    };
    update();
    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(update);
    for (const target of [row.current, full.current, small.current]) if (target) observer?.observe(target);
    return () => observer?.disconnect();
  }, [search, filters, actions, summary]);
  useLayoutEffect(() => {
    if (open) {
      setOpen(false);
      if (active) {
        const target = overflowing ? more.current : row.current?.querySelector<HTMLElement>("input,select:not(:disabled),button:not(:disabled),a");
        target?.focus();
      }
    }
  }, [signature, active]);
  useLayoutEffect(() => { if (restore && active) { setRestore(false); more.current?.focus(); } }, [restore, active]);
  useLayoutEffect(() => {
    const update = () => { if (active && row.current) onHeight(row.current.offsetHeight); };
    update(); const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(update);
    if (row.current) observer?.observe(row.current);
    return () => observer?.disconnect();
  }, [active, onHeight]);
  const close = useCallback((focus: boolean) => { setOpen(false); if (focus) more.current?.focus(); }, []);
  const render = (measuring: boolean) => {
    const hidden = (kind: "action" | "filter", control: string) => !measuring && plan[kind === "action" ? "actions" : "filters"].includes(control);
    return <>
      {summary ? <span data-kind="summary" className="workspace-toolbar__summary" role={measuring ? undefined : "status"} title={measuring ? undefined : summary}>{summary}</span> : null}
      {search ? <input type="search" data-kind="search" className="workspace-toolbar__search" aria-label={measuring ? undefined : search.label} placeholder={search.label} value={search.value} disabled={measuring} onChange={measuring ? undefined : event => search.onChange(event.target.value)} /> : null}
      {filters.map(filter => <span key={filter.id} data-kind="filter" data-control={filter.id} style={{ display: hidden("filter", filter.id) ? "none" : undefined }}><FilterControl filter={filter} measuring={measuring} /></span>)}
      {actions.map(action => <span key={action.id} data-kind="action" data-control={action.id} data-priority={action.primary ? "1" : "0"} style={{ display: hidden("action", action.id) ? "none" : undefined }}><ActionControl action={action} measuring={measuring} /></span>)}
      {measuring ? <span data-kind="more" className="ghost-button workspace-toolbar__more workspace-toolbar__action"><FontAwesomeIcon icon={workspaceToolbarIcons.more} /></span> : overflowing ? <button ref={more} type="button" data-kind="more" className="ghost-button workspace-toolbar__more workspace-toolbar__action" aria-label={t("workspaceToolbar.more")} title={t("workspaceToolbar.more")} aria-expanded={open} aria-haspopup="dialog" aria-controls={open ? id : undefined} onClick={() => setOpen(value => !value)}><FontAwesomeIcon icon={workspaceToolbarIcons.more} aria-hidden="true" /></button> : null}
    </>;
  };
  return <>
    <div ref={row} className={`workspace-toolbar__row${plan.compact ? " is-compact" : ""}`}>{render(false)}</div>
    <div className="workspace-toolbar__measurement" aria-hidden="true" inert>
      <div ref={full} className="workspace-toolbar__row workspace-toolbar__row--measure">{render(true)}</div>
      <div ref={small} className="workspace-toolbar__row workspace-toolbar__row--measure is-compact">{render(true)}</div>
    </div>
    {open && active && overflowing ? <ToolbarOverflow id={id} anchor={more} filters={filters.filter(filter => plan.filters.includes(filter.id))} actions={actions.filter(action => plan.actions.includes(action.id))} onClose={close} /> : null}
  </>;
}
