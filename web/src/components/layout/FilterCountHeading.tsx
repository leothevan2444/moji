import type { IconDefinition } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import "../../styles/filter-count-heading.scss";

const densities = ["inline", "stacked", "selected", "icons"] as const;
type Density = typeof densities[number];

interface FilterLayout { density: Density; wrapped: boolean; widths: number[] }
interface WindowRect { x: number; y: number; width: number; height: number }

interface Item { id: string; label: string; icon: IconDefinition; count?: number | null }
interface Props {
  title: string;
  kind: "tasks" | "performers";
  value: string;
  items: readonly Item[];
  loading?: boolean;
  onChange: (value: string) => void;
}

/** A single-choice filter, with counts for the search scope rather than the current page. */
export function FilterCountHeading({ title, kind, value, items, loading = false, onChange }: Props) {
  const { t } = useTranslation();
  const root = useRef<HTMLDivElement>(null), titleProbe = useRef<HTMLHeadingElement>(null);
  const group = useRef<HTMLDivElement>(null);
  const [windowReady, setWindowReady] = useState(false);
  const motion = useRef<{ value: string; current: WindowRect | null; from: WindowRect | null; clock: Animation | null; layout: FilterLayout | null }>({ value, current: null, from: null, clock: null, layout: null });
  useLayoutEffect(() => () => { motion.current.clock?.cancel(); }, []);
  const probes = useRef<Partial<Record<Density, HTMLDivElement>>>({});
  const [layout, setLayout] = useState<FilterLayout>({ density: "inline", wrapped: false, widths: [] });
  useLayoutEffect(() => {
    const update = () => {
      if (!root.current || !titleProbe.current || densities.some(mode => !probes.current[mode])) return;
      const style = getComputedStyle(root.current);
      const available = root.current.clientWidth - (parseFloat(style.paddingLeft) || 0) - (parseFloat(style.paddingRight) || 0) - 2;
      const gap = parseFloat(style.columnGap) || 0;
      const inline = densities.find(mode => {
        const titleWidth = titleProbe.current!.scrollWidth;
        return probes.current[mode]!.scrollWidth + titleWidth + gap <= available;
      });
      const density = inline ?? densities.find(mode => probes.current[mode]!.scrollWidth <= available) ?? "icons";
      // Read final widths from inert probes, never from buttons mid-transition.
      const widths = [...probes.current[density]!.children].map(node => parseFloat(getComputedStyle(node).width) || 0);
      const next = { density, wrapped: !inline, widths };
      setLayout(old => old.density === next.density && old.wrapped === next.wrapped && old.widths.length === widths.length
        && old.widths.every((width, index) => Math.abs(width - widths[index]) < .01) ? old : next);
    };
    update();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(update);
    for (const node of [root.current, titleProbe.current, ...Object.values(probes.current)]) if (node) observer.observe(node);
    return () => observer.disconnect();
  }, [items, value, title]);

  useLayoutEffect(() => {
    const node = group.current;
    const selected = node?.querySelector<HTMLButtonElement>('button[aria-pressed="true"]');
    if (!node || !selected) { motion.current.clock?.cancel(); motion.current.clock = null; setWindowReady(false); return; }
    const state = motion.current;
    const buttons = [...node.querySelectorAll("button")];
    const reduced = typeof matchMedia === "undefined" ? null : matchMedia("(prefers-reduced-motion: reduce)");
    if (state.value !== value || (state.clock && state.layout !== layout)) {
      state.clock?.cancel(); state.clock = null;
      state.value = value; state.from = state.current;
      const style = getComputedStyle(selected);
      const durationText = style.transitionDuration.split(",")[0].trim();
      const duration = parseFloat(durationText) * (durationText.endsWith("ms") ? 1 : 1000);
      const easing = style.transitionTimingFunction.split(/,(?![^(]*\))/)[0].trim();
      if (state.from && duration > 0 && !reduced?.matches && typeof Animation !== "undefined" && typeof KeyframeEffect !== "undefined") {
        // Use the browser's document timeline, duration and easing, just like button widths.
        state.clock = new Animation(new KeyframeEffect(null, [], { duration, easing, fill: "both" }), document.timeline);
        state.clock.play();
        state.clock.startTime = document.timeline.currentTime;
      }
    }
    state.layout = layout;
    let frameId = 0;
    const update = () => {
      if (frameId) cancelAnimationFrame(frameId);
      frameId = 0;
      if (reduced?.matches) { state.clock?.cancel(); state.clock = null; }
      const frame = node.getBoundingClientRect(), target = selected.getBoundingClientRect();
      const style = getComputedStyle(node);
      const scale = frame.width / (parseFloat(style.width) || node.offsetWidth || 1) || 1;
      const actual: WindowRect = { x: (target.left - frame.left) / scale - node.clientLeft,
        y: (target.top - frame.top) / scale - node.clientTop, width: target.width / scale, height: target.height / scale };
      const progress = state.clock ? Number(state.clock.effect?.getComputedTiming().progress ?? 0) : 1;
      const from = state.from;
      const finalWidth = layout.widths[items.findIndex(item => item.id === value)] || actual.width;
      const rect = from && progress < 1 ? {
        x: from.x + (actual.x - from.x) * progress, y: from.y + (actual.y - from.y) * progress,
        width: from.width + (finalWidth - from.width) * progress, height: from.height + (actual.height - from.height) * progress,
      } : actual;
      node.style.setProperty("--selection-x", `${rect.x}px`);
      node.style.setProperty("--selection-y", `${rect.y}px`);
      node.style.setProperty("--selection-width", `${rect.width}px`);
      node.style.setProperty("--selection-height", `${rect.height}px`);
      state.current = rect;
      setWindowReady(selected.offsetWidth > 0 && selected.offsetHeight > 0);
      if (progress >= 1) { state.clock?.cancel(); state.clock = null; state.from = null; }
      const resizing = buttons.some(button => (button.getAnimations?.() ?? []).some(animation =>
        "transitionProperty" in animation && animation.transitionProperty === "width" && (animation.playState === "running" || animation.pending)));
      // Width changes are already eased by CSS. Follow their current frame directly.
      if (state.clock || resizing) frameId = requestAnimationFrame(update);
    };
    update();
    reduced?.addEventListener("change", update);
    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(update);
    observer?.observe(node);
    buttons.forEach(button => observer?.observe(button));
    return () => { if (frameId) cancelAnimationFrame(frameId); observer?.disconnect(); reduced?.removeEventListener("change", update); };
  }, [value, items, layout]);

  const render = (measuring = false) => items.map((item, index) => {
    const known = item.count !== undefined && item.count !== null;
    const count = known ? String(item.count) : "—";
    const name = `${item.label} ${count}`;
    const help = `${t("filterCount.showOnly", { label: item.label })} · ${known ? t(`filterCount.${kind}Count`, { count: item.count }) : t("filterCount.unknown")}`;
    const content = <><span className="filter-count__identity"><FontAwesomeIcon icon={item.icon} aria-hidden="true" /><span className="filter-count__label">{item.label}</span></span><span className="filter-count__number">{count}</span></>;
    const className = `filter-count__item${item.id === value ? " is-active" : ""}`;
    return measuring ? <span key={item.id} className={className}>{content}</span> : <button key={item.id} type="button" className={className}
      style={layout.widths[index] > 0 ? { width: layout.widths[index] } : undefined}
      aria-label={name} title={help} aria-pressed={item.id === value} tabIndex={item.id === value ? 0 : -1}
      onClick={() => { if (item.id !== value) onChange(item.id); }}>{content}</button>;
  });
  return <div ref={root} className={`band-head filter-count-heading filter-count-heading--${kind}${layout.density === "icons" ? " is-compact" : ""}${layout.wrapped ? " is-wrapped" : ""}`}>
    <h2>{title}</h2>
    <div ref={group} data-window-ready={windowReady || undefined} className={`filter-count__items is-${layout.density}`} data-density={layout.density} role="toolbar" aria-label={t(`filterCount.${kind}Label`)} aria-busy={loading || undefined}
      onKeyDown={event => {
        const keys = ["ArrowLeft", "ArrowRight", "Home", "End"];
        if (!keys.includes(event.key)) return;
        const buttons = [...event.currentTarget.querySelectorAll<HTMLButtonElement>("button")];
        const index = buttons.indexOf(document.activeElement as HTMLButtonElement);
        const direction = getComputedStyle(event.currentTarget).direction === "rtl" ? -1 : 1;
        const next = event.key === "Home" ? 0 : event.key === "End" ? buttons.length - 1
          : (index + (event.key === "ArrowRight" ? direction : -direction) + buttons.length) % buttons.length;
        event.preventDefault(); buttons[next]?.focus();
      }}>{render()}</div>
    <div className="filter-count__measurement" aria-hidden="true" inert>
      <h2 ref={titleProbe}>{title}</h2>
      {densities.map(mode => <div key={mode} ref={node => { if (node) probes.current[mode] = node; else delete probes.current[mode]; }} className={`filter-count__items is-${mode}`} data-density={mode}>{render(true)}</div>)}
    </div>
  </div>;
}
