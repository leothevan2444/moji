import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck } from "@fortawesome/free-solid-svg-icons/faCheck";
import { faCircleHalfStroke } from "@fortawesome/free-solid-svg-icons/faCircleHalfStroke";
import { faMoon } from "@fortawesome/free-solid-svg-icons/faMoon";
import { faSun } from "@fortawesome/free-solid-svg-icons/faSun";
import type { RefObject } from "react";
import type { ThemePreference } from "../../hooks/useTheme";
import { useTranslation } from "react-i18next";

interface ThemeMenuProps {
  preference: ThemePreference;
  resolved: "light" | "dark";
  onSelect: (next: ThemePreference) => void;
  open: boolean;
  setOpen: (next: boolean) => void;
  buttonRef: RefObject<HTMLButtonElement | null>;
  menuRef: RefObject<HTMLDivElement | null>;
}

// 按钮图标按 preference 映射——而不是 resolved。
// 这样 auto 模式下用户看到的依然是「半圆」图标，明确表达「跟随系统」语义。
const PREFERENCE_ICONS: Record<ThemePreference, typeof faSun> = {
  light: faSun,
  dark: faMoon,
  auto: faCircleHalfStroke
};

const OPTIONS: ThemePreference[] = ["light", "dark", "auto"];

export function ThemeMenu({
  preference,
  resolved,
  onSelect,
  open,
  setOpen,
  buttonRef,
  menuRef
}: ThemeMenuProps) {
  const { t } = useTranslation();
  const ButtonIcon = PREFERENCE_ICONS[preference];
  const label = (value: ThemePreference | "light" | "dark") => t(`theme.${value}`);
  const buttonLabel = t("theme.label", { theme: label(preference) });

  return (
    <div className="theme-menu">
      <button
        ref={buttonRef}
        type="button"
        className="utility-button utility-icon-button"
        onClick={() => setOpen(!open)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={buttonLabel}
        title={buttonLabel}
      >
        <FontAwesomeIcon icon={ButtonIcon} aria-hidden="true" />
      </button>

      {open ? (
        <div
          ref={menuRef}
          className="theme-menu__panel"
          role="menu"
          aria-label={t("theme.choose")}
        >
          {OPTIONS.map((option) => {
            const Icon = PREFERENCE_ICONS[option];
            const isActive = option === preference;
            return (
              <button
                key={option}
                type="button"
                role="menuitem"
                className={`theme-menu__item ${isActive ? "is-active" : ""}`}
                aria-current={isActive ? "true" : undefined}
                onClick={() => onSelect(option)}
              >
                <FontAwesomeIcon icon={Icon} aria-hidden="true" />
                <span>{label(option)}</span>
                {isActive ? (
                  <FontAwesomeIcon icon={faCheck} className="theme-menu__check" aria-hidden="true" />
                ) : null}
                <span className="theme-menu__sr">
                  {option === "auto" ? t("theme.resolved", { theme: label(resolved) }) : ""}
                </span>
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
