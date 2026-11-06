import "../../styles/tooltip.scss";
import type { ReactNode } from "react";
import { useId, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

export function Tooltip({ info, children, className, label = info, onClick }: { info: string; children: ReactNode; className?: string; label?: string; onClick?: () => void }) {
  const anchor = useRef<HTMLSpanElement>(null);
  const tooltip = useRef<HTMLSpanElement>(null);
  const pointerFocus = useRef(false);
  const id = useId();
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [position, setPosition] = useState<{ left: number; top: number } | null>(null);
  const open = (hovered || focused) && !dismissed;

  useLayoutEffect(() => {
    if (!open) { setPosition(null); return; }
    const place = () => {
      if (!anchor.current || !tooltip.current) return;
      const target = anchor.current.getBoundingClientRect();
      const box = tooltip.current.getBoundingClientRect();
      const viewport = window.visualViewport;
      const left = viewport?.offsetLeft ?? 0, top = viewport?.offsetTop ?? 0;
      const right = left + (viewport?.width ?? window.innerWidth);
      const bottom = top + (viewport?.height ?? window.innerHeight);
      if (target.bottom < top || target.top > bottom || target.right < left || target.left > right) {
        setPosition(null); return;
      }
      const rtl = getComputedStyle(anchor.current).direction === "rtl";
      const preferredLeft = rtl ? target.right - box.width : target.left;
      const below = target.bottom + 8;
      const preferredTop = below + box.height <= bottom - 8 ? below : target.top - box.height - 8;
      setPosition({
        left: Math.max(left + 8, Math.min(preferredLeft, right - box.width - 8)),
        top: Math.max(top + 8, Math.min(preferredTop, bottom - box.height - 8))
      });
    };
    place();
    window.addEventListener("resize", place);
    window.addEventListener("scroll", place, true);
    window.visualViewport?.addEventListener("resize", place);
    window.visualViewport?.addEventListener("scroll", place);
    return () => {
      window.removeEventListener("resize", place);
      window.removeEventListener("scroll", place, true);
      window.visualViewport?.removeEventListener("resize", place);
      window.visualViewport?.removeEventListener("scroll", place);
    };
  }, [open, info]);

  return <span ref={anchor} className={className} tabIndex={0} role={onClick ? "button" : undefined} onClick={onClick} aria-label={label} aria-describedby={open ? id : undefined}
    onPointerDown={() => { pointerFocus.current = true; setFocused(false); }}
    onMouseEnter={() => { setHovered(true); setDismissed(false); }}
    onMouseLeave={() => setHovered(false)}
    onFocus={() => { setFocused(!pointerFocus.current); setDismissed(false); }}
    onBlur={() => { setFocused(false); pointerFocus.current = false; }}
    onKeyDown={event => { if (event.key !== "Escape") { pointerFocus.current = false; setFocused(true); } if (onClick && (event.key === "Enter" || event.key === " ")) { event.preventDefault(); onClick(); } if (event.key === "Escape" && open) { event.stopPropagation(); setDismissed(true); } }}>
    {children}
    {open ? createPortal(<span ref={tooltip} id={id} className="info-tooltip" role="tooltip"
      style={{ left: position?.left ?? 0, top: position?.top ?? 0, visibility: position ? "visible" : "hidden" }}>{info}</span>, document.body) : null}
  </span>;
}
