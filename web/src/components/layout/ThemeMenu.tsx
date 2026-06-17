import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faCheck,
  faCircleHalfStroke,
  faMoon,
  faSun
} from "@fortawesome/free-solid-svg-icons";
import type { RefObject } from "react";
import type { ThemePreference } from "../../hooks/useTheme";

interface ThemeMenuProps {
  preference: ThemePreference;
  resolved: "light" | "dark";
  onSelect: (next: ThemePreference) => void;
  open: boolean;
  setOpen: (next: boolean) => void;
  buttonRef: RefObject<HTMLButtonElement | null>;
  menuRef: RefObject<HTMLDivElement | null>;
}

const PREFERENCE_LABELS: Record<ThemePreference, string> = {
  light: "浅色",
  dark: "深色",
  auto: "自动"
};

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
  const ButtonIcon = PREFERENCE_ICONS[preference];
  const buttonLabel = `主题：${PREFERENCE_LABELS[preference]}`;

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
          aria-label="选择主题"
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
                <span>{PREFERENCE_LABELS[option]}</span>
                {isActive ? (
                  <FontAwesomeIcon icon={faCheck} className="theme-menu__check" aria-hidden="true" />
                ) : null}
                <span className="theme-menu__sr">
                  {option === "auto" ? `（当前显示：${PREFERENCE_LABELS[resolved]}）` : ""}
                </span>
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
