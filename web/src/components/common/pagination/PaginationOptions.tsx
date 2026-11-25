import { useLayoutEffect, useRef, useState, type RefObject } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { PAGE_SIZE_OPTIONS } from "../../../constants";
import { useClickOutside } from "../../../hooks/useClickOutside";

interface Props {
  id: string;
  showEdges: boolean;
  anchor: RefObject<HTMLButtonElement | null>;
  pageSize: number;
  disabled: boolean;
  firstDisabled: boolean;
  lastDisabled: boolean;
  onPageSizeChange: (size: number) => void;
  onFirst: () => void;
  onLast: () => void;
  onClose: (restoreFocus: boolean) => void;
}

export function PaginationOptions({ id, showEdges, anchor, pageSize, disabled, firstDisabled, lastDisabled, onPageSizeChange, onFirst, onLast, onClose }: Props) {
  const { t } = useTranslation();
  const panel = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState<{ left: number; top: number } | null>(null);
  useClickOutside([anchor, panel], () => onClose(false), true);

  useLayoutEffect(() => {
    const place = () => {
      if (!anchor.current || !panel.current) return;
      const target = anchor.current.getBoundingClientRect();
      const box = panel.current.getBoundingClientRect();
      const viewport = window.visualViewport;
      const left = viewport?.offsetLeft ?? 0, top = viewport?.offsetTop ?? 0;
      const right = left + (viewport?.width ?? innerWidth), bottom = top + (viewport?.height ?? innerHeight);
      if (target.bottom < top || target.top > bottom || target.right < left || target.left > right) {
        onClose(false);
        return;
      }
      // Fixed portal coordinates must account for CSS zoom as well as viewport bounds.
      const scale = box.width / panel.current.offsetWidth || 1;
      const preferredLeft = getComputedStyle(anchor.current).direction === "rtl" ? target.left : target.right - box.width;
      const preferredTop = target.bottom + 8 + box.height <= bottom - 8 ? target.bottom + 8 : target.top - box.height - 8;
      setPosition({ left: Math.max(left + 8, Math.min(preferredLeft, right - box.width - 8)) / scale,
        top: Math.max(top + 8, Math.min(preferredTop, bottom - box.height - 8)) / scale });
    };
    place();
    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(place);
    if (panel.current) observer?.observe(panel.current);
    window.addEventListener("resize", place);
    window.addEventListener("scroll", place, true);
    window.visualViewport?.addEventListener("resize", place);
    window.visualViewport?.addEventListener("scroll", place);
    return () => {
      observer?.disconnect();
      window.removeEventListener("resize", place);
      window.removeEventListener("scroll", place, true);
      window.visualViewport?.removeEventListener("resize", place);
      window.visualViewport?.removeEventListener("scroll", place);
    };
  }, [anchor, onClose]);

  const positioned = position !== null;
  useLayoutEffect(() => {
    if (positioned) panel.current?.querySelector<HTMLElement>("select:not(:disabled), button:not(:disabled)")?.focus();
  }, [positioned]);

  return createPortal(
    <div ref={panel} id={id} className="pagination-options" role="dialog" aria-label={t("pagination.options")}
      style={{ left: position?.left ?? 0, top: position?.top ?? 0, visibility: position ? "visible" : "hidden" }}
      onBlur={event => {
        const next = event.relatedTarget;
        if (next instanceof Node && !panel.current?.contains(next) && !anchor.current?.contains(next)) onClose(false);
      }}
      onKeyDown={event => {
        if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); onClose(true); }
        if (event.key === "Tab") {
          const controls = [...(panel.current?.querySelectorAll<HTMLElement>("select:not(:disabled),button:not(:disabled)") ?? [])];
          if (event.shiftKey && event.target === controls[0]) { event.preventDefault(); onClose(true); }
          else if (!event.shiftKey && event.target === controls.at(-1)) onClose(true);
        }
      }}>
      <label>{t("pagination.pageSize")}<select aria-label={t("pagination.pageSize")} value={pageSize} disabled={disabled}
        onChange={event => { onPageSizeChange(Number(event.target.value)); onClose(true); }}>
        {PAGE_SIZE_OPTIONS.map(size => <option key={size} value={size}>{t("pagination.sizeOption", { size })}</option>)}
      </select></label>
      {showEdges ? <div className="pagination-options__actions">
        <button type="button" className="ghost-button" disabled={firstDisabled} onClick={() => { onFirst(); onClose(true); }}>{t("pagination.first")}</button>
        <button type="button" className="ghost-button" disabled={lastDisabled} onClick={() => { onLast(); onClose(true); }}>{t("pagination.last")}</button>
      </div> : null}
    </div>, document.body,
  );
}
