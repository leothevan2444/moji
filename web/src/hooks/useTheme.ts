import { useCallback, useEffect, useRef, useState, type RefObject } from "react";
import { useClickOutside } from "./useClickOutside";

export type ThemePreference = "light" | "dark" | "auto";
export type ResolvedTheme = "light" | "dark";

const STORAGE_KEY = "moji:theme";

function readStoredPreference(): ThemePreference {
  if (typeof window === "undefined") return "auto";
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (raw === "light" || raw === "dark" || raw === "auto") return raw;
  } catch {
    /* 忽略 localStorage 异常（隐私模式 / SSR / 沙箱） */
  }
  return "auto";
}

function writeStoredPreference(value: ThemePreference): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(STORAGE_KEY, value);
  } catch {
    /* 忽略 */
  }
}

function readSystemPreference(): ResolvedTheme {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return "light";
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyToDocument(value: ThemePreference): void {
  if (typeof document === "undefined") return;
  if (value === "light" || value === "dark") {
    document.documentElement.setAttribute("data-theme", value);
  } else {
    // auto 模式移除属性，让 styles.css 顶部的 prefers-color-scheme 媒体查询接管。
    document.documentElement.removeAttribute("data-theme");
  }
}

/**
 * 三态主题状态机：用户偏好（light / dark / auto）+ 系统偏好。
 *
 *  - `resolved` 在 render 中派生，永不存到 state，避免与系统变化脱钩。
 *  - 持久化到 localStorage 键 `moji:theme`；防 FOUC 由 `web/index.html` 中的
 *    同步内联脚本处理，逻辑保持一致。
 *  - 仅在 `preference === "auto"` 时订阅 `matchMedia.change`，非 auto 模式无开销。
 *  - 同步管理下拉菜单的 `open` 状态与 `useClickOutside` 协作。
 */
export function useTheme() {
  const [preference, setPreferenceState] = useState<ThemePreference>(readStoredPreference);
  const [systemPref, setSystemPref] = useState<ResolvedTheme>(readSystemPreference);
  const [open, setOpen] = useState(false);

  const buttonRef = useRef<HTMLButtonElement | null>(null);
  const menuRef = useRef<HTMLDivElement | null>(null);

  const resolved: ResolvedTheme = preference === "auto" ? systemPref : preference;

  // 持久化 + 同步到 <html data-theme>
  useEffect(() => {
    writeStoredPreference(preference);
    applyToDocument(preference);
  }, [preference]);

  // 仅在 auto 模式下订阅系统偏好变化
  useEffect(() => {
    if (preference !== "auto") return;
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") return;

    const mql = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = (event: MediaQueryListEvent) => {
      setSystemPref(event.matches ? "dark" : "light");
    };
    // 同步一次当前状态（与初始化的 readSystemPreference 保持一致）
    setSystemPref(mql.matches ? "dark" : "light");
    mql.addEventListener("change", handler);
    return () => mql.removeEventListener("change", handler);
  }, [preference]);

  const setPreference = useCallback((next: ThemePreference) => {
    setPreferenceState(next);
    setOpen(false);
  }, []);

  useClickOutside([menuRef, buttonRef], () => setOpen(false), open);

  return {
    preference,
    resolved,
    onSelect: setPreference,
    open,
    setOpen,
    buttonRef: buttonRef as RefObject<HTMLButtonElement | null>,
    menuRef: menuRef as RefObject<HTMLDivElement | null>
  };
}
