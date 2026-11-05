import "../../styles/settings-info.scss";
import { useId, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleInfo } from "@fortawesome/free-solid-svg-icons";

export function SettingsInfoTooltip({ info }: { info: string }) {
  const anchor = useRef<HTMLSpanElement>(null);
  const tooltip = useRef<HTMLSpanElement>(null);
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

  return <span ref={anchor} className="settings-info" tabIndex={0} aria-label={info} aria-describedby={open ? id : undefined}
    onMouseEnter={() => { setHovered(true); setDismissed(false); }}
    onMouseLeave={() => setHovered(false)}
    onFocus={() => { setFocused(true); setDismissed(false); }}
    onBlur={() => setFocused(false)}
    onKeyDown={event => { if (event.key === "Escape" && open) { event.stopPropagation(); setDismissed(true); } }}>
    <FontAwesomeIcon icon={faCircleInfo} aria-hidden="true" />
    {open ? createPortal(<span ref={tooltip} id={id} className="settings-info__tooltip" role="tooltip"
      style={{ left: position?.left ?? 0, top: position?.top ?? 0, visibility: position ? "visible" : "hidden" }}>{info}</span>, document.body) : null}
  </span>;
}
