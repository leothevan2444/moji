import { useLayoutEffect, useRef, useState } from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import type { IconDefinition } from "@fortawesome/free-solid-svg-icons";

type Layout = "labels" | "icons" | "wrapped-labels" | "wrapped-icons";
interface HeadingItem {
  id: string;
  label: string;
  icon: IconDefinition;
  count?: number;
  selected?: boolean;
  onSelect?: () => void;
}
interface Props {
  title: string;
  items: readonly HeadingItem[];
  variant: "metrics" | "filters";
  itemsLabel?: string;
  itemsClassName?: string;
}

/** Use icons before moving the controls to their own row, then reassess labels. */
export function ResponsivePageHeading({ title, items, variant, itemsLabel, itemsClassName = "" }: Props) {
  const [layout, setLayout] = useState<Layout>("icons");
  const root = useRef<HTMLDivElement>(null);
  const titleMeasure = useRef<HTMLHeadingElement>(null);
  const fullMeasure = useRef<HTMLDivElement>(null);
  const iconMeasure = useRef<HTMLDivElement>(null);
  const compact = layout === "icons" || layout === "wrapped-icons";

  useLayoutEffect(() => {
    const element = root.current!, leading = titleMeasure.current!, full = fullMeasure.current!, icons = iconMeasure.current!;
    const update = () => {
      const style = getComputedStyle(element);
      const available = element.clientWidth - parseFloat(style.paddingLeft) - parseFloat(style.paddingRight) - 2;
      const singleRowSpace = available - leading.scrollWidth - parseFloat(style.columnGap);
      // Use layout CSS pixels consistently, including under page zoom.
      if (full.scrollWidth <= singleRowSpace) setLayout("labels");
      else if (icons.scrollWidth <= singleRowSpace) setLayout("icons");
      else setLayout(full.scrollWidth <= available ? "wrapped-labels" : "wrapped-icons");
    };
    update();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(update);
    for (const target of [element, leading, full, icons]) observer.observe(target);
    return () => observer.disconnect();
  }, []);

  const renderItems = (iconsOnly: boolean, measuring = false) => items.map(item => {
    const name = item.count === undefined ? item.label : `${item.label} ${item.count}`;
    const content = <>
      {iconsOnly ? <FontAwesomeIcon icon={item.icon} aria-hidden="true" /> : <span className="responsive-heading__label">{item.label}</span>}
      {item.count !== undefined ? <span className="responsive-heading__count">{item.count}</span> : null}
    </>;
    const className = `responsive-heading__item${item.selected ? " is-active" : ""}`;
    return item.onSelect && !measuring ? (
      <button key={item.id} type="button" className={className} onClick={item.onSelect} aria-label={name} aria-pressed={item.selected} title={name}>{content}</button>
    ) : (
      <span key={item.id} className={className} role={measuring ? undefined : "img"} aria-label={measuring ? undefined : name} title={measuring ? undefined : name}>{content}</span>
    );
  });

  return (
    <div ref={root} className={`band-head responsive-heading responsive-heading--${variant}${compact ? " is-compact" : ""}${layout.startsWith("wrapped-") ? " is-wrapped" : ""}`}>
      <div className="responsive-heading__title"><h2>{title}</h2></div>
      <div className={`responsive-heading__items ${itemsClassName}`} role={itemsLabel ? "group" : undefined} aria-label={itemsLabel}>{renderItems(compact)}</div>
      <div className="responsive-heading__measurement" aria-hidden="true" inert>
        <h2 ref={titleMeasure}>{title}</h2>
        <div ref={fullMeasure} className="responsive-heading__items">{renderItems(false, true)}</div>
        <div ref={iconMeasure} className="responsive-heading__items">{renderItems(true, true)}</div>
      </div>
    </div>
  );
}
