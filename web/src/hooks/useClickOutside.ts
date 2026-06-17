import { useEffect, type RefObject } from "react";

/**
 * 监听点击外部 / Esc 键关闭行为。
 *
 *  - 仅在 `active` 为 true 时订阅，节省不必要的事件监听。
 *  - 判断「外部」时把传入的 ref 视为内部区域——这些 ref（通常是按钮本体和面板）
 *    上的点击不会触发 `onOutside`，避免按钮和面板打架。
 *  - 组件卸载或 `active` 变 false 时自动清理监听。
 */
export function useClickOutside(
  refs: RefObject<HTMLElement | null>[],
  onOutside: () => void,
  active: boolean
) {
  useEffect(() => {
    if (!active) return;

    const handleMouseDown = (event: MouseEvent) => {
      const target = event.target;
      if (!(target instanceof Node)) return;
      for (const ref of refs) {
        if (ref.current && ref.current.contains(target)) return;
      }
      onOutside();
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onOutside();
    };

    document.addEventListener("mousedown", handleMouseDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handleMouseDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [active, onOutside, refs]);
}
