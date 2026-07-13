import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faChartColumn } from "@fortawesome/free-solid-svg-icons/faChartColumn";
import { faCircleQuestion } from "@fortawesome/free-solid-svg-icons/faCircleQuestion";
import { faGear } from "@fortawesome/free-solid-svg-icons/faGear";
import { NAV_ITEMS } from "../../constants/navigation";
import { NavLink, useNavigate } from "react-router";
import { ThemeMenu } from "./ThemeMenu";
import type { ThemePreference } from "../../hooks/useTheme";
import type { RefObject } from "react";
import { useTranslation } from "react-i18next";

interface HeaderProps {
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

export function Header({ onOpenHelp, theme }: HeaderProps) {
  const navigate = useNavigate();
  const { t } = useTranslation();
  return (
    <header className="masthead">
      <div className="masthead__brand">
        <div className="title-row">
          <h1>Moji</h1>
        </div>
      </div>
      <div className="masthead__actions" aria-label={t("navigation.label")}>
        <div className="masthead__navgroup">
          <div className="tab-group">
            {NAV_ITEMS.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) => `nav-tab ${isActive ? "is-active" : ""}`}
              >
                {t(item.labelKey)}
              </NavLink>
            ))}
          </div>
        </div>

        <div className="masthead__toolgroup">
          <div className="utility-group">
            <button
              type="button"
              className="utility-button utility-icon-button"
              onClick={() => navigate("/stats")}
              aria-label={t("navigation.stats")}
              title={t("navigation.stats")}
            >
              <FontAwesomeIcon icon={faChartColumn} aria-hidden="true" />
            </button>
            <button
              type="button"
              className="utility-button utility-icon-button"
              onClick={() => navigate("/settings/connections")}
              aria-label={t("navigation.settings")}
              title={t("navigation.settings")}
            >
              <FontAwesomeIcon icon={faGear} aria-hidden="true" />
            </button>
            <button
              type="button"
              className="utility-button utility-icon-button"
              onClick={onOpenHelp}
              aria-label={t("navigation.help")}
              title={t("navigation.help")}
            >
              <FontAwesomeIcon icon={faCircleQuestion} aria-hidden="true" />
            </button>
            <ThemeMenu
              preference={theme.preference}
              resolved={theme.resolved}
              onSelect={theme.onSelect}
              open={theme.open}
              setOpen={theme.setOpen}
              buttonRef={theme.buttonRef}
              menuRef={theme.menuRef}
            />
          </div>
        </div>
      </div>
    </header>
  );
}
