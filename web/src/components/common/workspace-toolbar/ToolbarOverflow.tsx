import { useLayoutEffect, useRef, useState, type RefObject } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { useClickOutside } from "../../../hooks/useClickOutside";
import { ActionControl, FilterControl } from "./ToolbarControls";
import type { ToolbarAction, ToolbarFilter } from "./types";

export function ToolbarOverflow({ id, anchor, filters, actions, onClose }: {
  id: string;
  anchor: RefObject<HTMLButtonElement | null>;
  filters: readonly ToolbarFilter[];
  actions: readonly ToolbarAction[];
  onClose: (restoreFocus: boolean) => void;
}) {
  const { t } = useTranslation();
  const panel = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState<{ left: number; top: number; maxHeight: number; maxWidth: number } | null>(null);
  useClickOutside([anchor, panel], () => onClose(false), true);
  useLayoutEffect(() => {
    const place = () => {
      if (!panel.current || !anchor.current) return;
      const target = anchor.current.getBoundingClientRect(), box = panel.current.getBoundingClientRect(), viewport = window.visualViewport;
      const left = viewport?.offsetLeft ?? 0, top = viewport?.offsetTop ?? 0, right = left + (viewport?.width ?? innerWidth), bottom = top + (viewport?.height ?? innerHeight);
      if (target.bottom < top || target.top > bottom || target.right < left || target.left > right) { onClose(false); return; }
      // Keep fractional CSS widths: offsetWidth rounding can oscillate at non-integer zoom.
      const scale = box.width / parseFloat(getComputedStyle(panel.current).width) || 1;
      const desiredLeft = getComputedStyle(anchor.current).direction === "rtl" ? target.left : target.right - box.width;
      const desiredTop = target.bottom + 8 + box.height <= bottom - 8 ? target.bottom + 8 : target.top - box.height - 8;
      setPosition({ left: Math.max(left + 8, Math.min(desiredLeft, right - box.width - 8)) / scale, top: Math.max(top + 8, Math.min(desiredTop, bottom - box.height - 8)) / scale, maxHeight: (bottom - top - 16) / scale, maxWidth: (right - left - 16) / scale });
    };
    place();
    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(place);
    if (panel.current) observer?.observe(panel.current);
    window.addEventListener("resize", place); window.addEventListener("scroll", place, true);
    window.visualViewport?.addEventListener("resize", place); window.visualViewport?.addEventListener("scroll", place);
    return () => { observer?.disconnect(); window.removeEventListener("resize", place); window.removeEventListener("scroll", place, true); window.visualViewport?.removeEventListener("resize", place); window.visualViewport?.removeEventListener("scroll", place); };
  }, [anchor, onClose]);
  const positioned = position !== null;
  useLayoutEffect(() => { if (positioned) (panel.current?.querySelector<HTMLElement>("select:not(:disabled),button:not(:disabled),a") ?? panel.current)?.focus({ preventScroll: true }); }, [positioned]);
  return createPortal(<div ref={panel} id={id} className="workspace-toolbar-menu" tabIndex={-1} role="dialog" aria-label={t("workspaceToolbar.more")}
    style={{ left: position?.left ?? 0, top: position?.top ?? 0, maxHeight: position?.maxHeight, maxWidth: position?.maxWidth, visibility: position ? "visible" : "hidden" }}
    onBlur={event => { if (event.relatedTarget instanceof Node && !panel.current?.contains(event.relatedTarget) && !anchor.current?.contains(event.relatedTarget)) onClose(false); }}
    onKeyDown={event => {
      if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); onClose(true); }
      if (event.key === "Tab") {
        const controls = [...(panel.current?.querySelectorAll<HTMLElement>("select:not(:disabled),button:not(:disabled),a") ?? [])];
        if (!controls.length) onClose(true);
        else if (event.shiftKey && event.target === controls[0]) { event.preventDefault(); onClose(true); }
        else if (!event.shiftKey && event.target === controls.at(-1)) onClose(true);
      }
    }}>
    {filters.map(filter => <FilterControl key={filter.id} filter={filter} />)}
    {actions.map(action => <span key={action.id} onClick={event => { if (!action.disabled && event.target instanceof Element && event.target.closest("button,a")) onClose(true); }}><ActionControl action={action} menu /></span>)}
  </div>, document.body);
}
