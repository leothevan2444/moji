import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faGaugeHigh, faChartColumn, faCircleQuestion, faGear, faBars, faXmark } from "@fortawesome/free-solid-svg-icons";
import { NAV_ITEMS } from "../../constants/navigation";
import { Link, NavLink, useLocation } from "react-router";
import { ThemeMenu } from "./ThemeMenu";
import type { ThemePreference } from "../../hooks/useTheme";
import { useEffect, useRef, useState, type RefObject } from "react";
import { useTranslation } from "react-i18next";
import { useClickOutside } from "../../hooks/useClickOutside";

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
  const location = useLocation();
  const [panel, setPanel] = useState<"navigation" | null>(null);
  const headerRef = useRef<HTMLElement>(null);
  const navigationButton = useRef<HTMLButtonElement>(null);
  useClickOutside([headerRef], () => setPanel(null), panel === "navigation");
  useEffect(() => { setPanel(null); }, [location]);
  useEffect(() => { if (theme.open) setPanel(null); }, [theme.open]);
  const toggle = (next: "navigation") => {
    theme.setOpen(false);
    setPanel(current => current === next ? null : next);
  };
  // At the desktop breakpoint the disclosure becomes a permanent navigation bar.
  useEffect(() => {
    const media = window.matchMedia("(min-width: 1000px)");
    const close = () => {
      if (media.matches) {
        if (document.activeElement === navigationButton.current) headerRef.current?.querySelector<HTMLAnchorElement>(".masthead__navigation a")?.focus();
        setPanel(null);
      }
    };
    media.addEventListener("change", close);
    return () => media.removeEventListener("change", close);
  }, []);
  const currentLabel = location.pathname.startsWith("/settings") ? t("navigation.settings")
    : location.pathname === "/stats" ? t("navigation.stats")
    : t((NAV_ITEMS.find(item => item.end ? location.pathname === item.to : location.pathname.startsWith(item.to)) ?? NAV_ITEMS[0]).labelKey);

  return (
    <header ref={headerRef} className="masthead" onBlur={event => {
      if (!event.currentTarget.contains(event.relatedTarget)) setPanel(current => current === "navigation" ? null : current);
    }} onKeyDown={event => {
      if (event.key === "Escape" && panel) {
        event.preventDefault();
        navigationButton.current?.focus();
        setPanel(null);
      }
    }}>
      <Link className="masthead__brand" to="/" aria-label={`Moji · ${t("navigation.home")}`}>Moji</Link>
      <button ref={navigationButton} type="button" className="masthead__toggle" aria-label={`${t("navigation.label")}: ${currentLabel}`}
        aria-expanded={panel === "navigation"} aria-controls="primary-navigation" onClick={() => toggle("navigation")}>
        <span>{currentLabel}</span><FontAwesomeIcon icon={panel === "navigation" ? faXmark : faBars} aria-hidden="true" />
      </button>
      <nav id="primary-navigation" className={`masthead__navigation${panel === "navigation" ? " is-open" : ""}`} aria-label={t("navigation.label")}>
        {NAV_ITEMS.map(item => (
          <NavLink key={item.to} to={item.to} end={item.end} className={({ isActive }) => `masthead__link${isActive ? " is-active" : ""}`}
            onClick={() => { if (panel === "navigation") navigationButton.current?.focus(); setPanel(null); }}>
            <FontAwesomeIcon icon={item.icon} aria-hidden="true" /><span>{t(item.labelKey)}</span>
          </NavLink>
        ))}
      </nav>
      <div className="masthead__tools">
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
      </div>
    </header>
  );
}
