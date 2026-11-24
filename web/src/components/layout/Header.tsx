import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faGaugeHigh, faChartColumn, faCircleQuestion, faGear } from "@fortawesome/free-solid-svg-icons";
import { NAV_ITEMS } from "../../constants/navigation";
import { Link, NavLink } from "react-router";
import { ThemeMenu } from "./ThemeMenu";
import type { ThemePreference } from "../../hooks/useTheme";
import { useLayoutEffect, useRef, useState, type RefObject } from "react";
import { useTranslation } from "react-i18next";

import { useNsfwMask } from "../../hooks/useNsfwMask";

interface HeaderProps {
  developerMode?: boolean;
  onOpenPerformance?: () => void;
  onOpenHelp: () => void;
  theme: {
    preference: ThemePreference;
    resolved: "light" | "dark";
    onSelect: (next: ThemePreference) => void;
    open: boolean;
    setOpen: (next: boolean) => void;
    buttonRef: RefObject<HTMLButtonElement | null>;
    menuRef: RefObject<HTMLDivElement | null>;
  };
}

export function Header({ onOpenHelp, theme, developerMode = false, onOpenPerformance }: HeaderProps) {
  const { t } = useTranslation();
  const { masked, toggle: toggleNsfwMask } = useNsfwMask();
  const [layout, setLayout] = useState<"labels" | "icons" | "wrapped-labels" | "wrapped-icons">("icons");
  const iconsOnly = layout === "icons" || layout === "wrapped-icons";
  const wrapped = layout.startsWith("wrapped-");
  const headerRef = useRef<HTMLElement>(null);
  const brandRef = useRef<HTMLAnchorElement>(null);
  const measureRef = useRef<HTMLDivElement>(null);
  const toolsRef = useRef<HTMLDivElement>(null);
  const iconsMeasureRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const header = headerRef.current!;
    const brand = brandRef.current!;
    const measure = measureRef.current!;
    const tools = toolsRef.current!;
    const iconsMeasure = iconsMeasureRef.current!;
    const update = () => {
      const style = getComputedStyle(header);
      const available = header.clientWidth - parseFloat(style.paddingLeft) - parseFloat(style.paddingRight);
      // clientWidth, offsetWidth and scrollWidth all use layout CSS pixels.
      // Mixing them with scaled client rectangles underestimates demand when
      // the page is zoomed out. Reserve 2px for integer measurement rounding.
      const gap = parseFloat(style.columnGap);
      const singleRowSpace = available - brand.offsetWidth - tools.offsetWidth - 2 * gap - 2;
      if (measure.scrollWidth <= singleRowSpace) {
        setLayout("labels");
      } else if (iconsMeasure.scrollWidth <= singleRowSpace) {
        setLayout("icons");
      } else {
        const wrappedRowSpace = available - brand.offsetWidth - gap - 2;
        setLayout(measure.scrollWidth <= wrappedRowSpace ? "wrapped-labels" : "wrapped-icons");
      }
    };
    update();
    const observer = new ResizeObserver(update);
    for (const element of [header, brand, measure, iconsMeasure, tools]) observer.observe(element);
    return () => observer.disconnect();
  }, [t]);

  return (
    <header ref={headerRef} className={`masthead${iconsOnly ? " masthead--icons" : ""}${wrapped ? " masthead--wrapped" : ""}`}>
      <Link ref={brandRef} className="masthead__brand" to="/" aria-label={`Moji · ${t("navigation.home")}`}>Moji</Link>
      <div className="masthead__measurement" aria-hidden="true" inert>
        <div ref={measureRef} className="masthead__measure">
          {NAV_ITEMS.map(item => (
            <span key={item.to} className="masthead__link">
              <FontAwesomeIcon icon={item.icon} /><span>{t(item.labelKey)}</span>
            </span>
          ))}
        </div>
        <div ref={iconsMeasureRef} className="masthead__measure masthead__measure--icons">
          {NAV_ITEMS.map(item => (
            <span key={item.to} className="masthead__link"><FontAwesomeIcon icon={item.icon} /></span>
          ))}
        </div>
      </div>
      <nav id="primary-navigation" className="masthead__navigation" aria-label={t("navigation.label")}>
        {NAV_ITEMS.map(item => (
          <NavLink key={item.to} to={item.to} end={item.end} className={({ isActive }) => `masthead__link${isActive ? " is-active" : ""}`}
            aria-label={t(item.labelKey)} title={iconsOnly ? t(item.labelKey) : undefined}>
            <FontAwesomeIcon icon={item.icon} aria-hidden="true" /><span>{t(item.labelKey)}</span>
          </NavLink>
        ))}
      </nav>
      <div ref={toolsRef} className="masthead__tools">
        {developerMode ? (
          <button type="button" className="utility-button utility-icon-button" onClick={onOpenPerformance}
            aria-label={t("performance.title")} title={t("performance.title")}>
            <FontAwesomeIcon icon={faGaugeHigh} aria-hidden="true" />
          </button>
        ) : null}
        <NavLink to="/stats" className="utility-button utility-icon-button" aria-label={t("navigation.stats")} title={t("navigation.stats")}>
          <FontAwesomeIcon icon={faChartColumn} aria-hidden="true" />
        </NavLink>
        <NavLink to="/settings" className="utility-button utility-icon-button" aria-label={t("navigation.settings")} title={t("navigation.settings")}>
          <FontAwesomeIcon icon={faGear} aria-hidden="true" />
        </NavLink>
        <button type="button" className="utility-button utility-icon-button" onClick={onOpenHelp}
          aria-label={t("navigation.help")} title={t("navigation.help")}>
          <FontAwesomeIcon icon={faCircleQuestion} aria-hidden="true" />
        </button>
        <ThemeMenu {...theme} />
        <button type="button" className="utility-button utility-icon-button nsfw-mask-toggle" onClick={toggleNsfwMask}
          aria-label={t("nsfwMask.label")} aria-pressed={masked}
          title={t(masked ? "nsfwMask.disable" : "nsfwMask.enable")}>
          NSFW
        </button>
      </div>
    </header>
  );
}
