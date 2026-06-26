import { useEffect, type RefObject } from "react";

/**
 * 全局 `/` 快捷键：按下时聚焦传入的 input。
 *  - 已聚焦 input / textarea / contentEditable 时不拦截（让 `/` 正常输入）；
 *  - 携带 Cmd / Ctrl / Alt 修饰键时不触发（避免与 IDE / 系统快捷键冲突）。
 */
export function useGlobalSlashShortcut(inputRef: RefObject<HTMLInputElement | null>, active = true) {
  useEffect(() => {
    if (!active) return undefined;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey) return;
      const target = event.target;
      if (target instanceof HTMLElement) {
        const tag = target.tagName;
        if (tag === "INPUT" || tag === "TEXTAREA" || target.isContentEditable) return;
      }
      event.preventDefault();
      inputRef.current?.focus();
      inputRef.current?.select();
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [active, inputRef]);
}