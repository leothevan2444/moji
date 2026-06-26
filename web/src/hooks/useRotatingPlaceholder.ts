import { useEffect, useState } from "react";

/**
 * 在 `items` 中轮播，每 `interval` 毫秒前进一项。
 * `paused` 为 true 时停止（输入框聚焦期间调用，避免视觉跳动）。
 * items 少于 2 项时直接返回第一项，不启定时器。
 */
export function useRotatingPlaceholder(items: readonly string[], interval = 4500, paused = false): string {
  const [index, setIndex] = useState(0);

  useEffect(() => {
    if (paused || items.length < 2) return undefined;
    const handle = window.setInterval(() => {
      setIndex((current) => (current + 1) % items.length);
    }, interval);
    return () => window.clearInterval(handle);
  }, [paused, items, interval]);

  if (items.length === 0) return "";
  return items[index] ?? items[0];
}