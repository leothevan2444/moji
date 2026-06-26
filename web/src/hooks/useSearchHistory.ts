import { useCallback, useEffect, useState } from "react";

const STORAGE_KEY = "moji.discovery.searchHistory.v1";
const MAX_ITEMS = 8;

function readStorage(): string[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((entry): entry is string => typeof entry === "string").slice(0, MAX_ITEMS);
  } catch {
    // 隐私模式 / quota 异常 / 反序列化失败时静默降级为内存存储。
    return [];
  }
}

function writeStorage(items: string[]) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
  } catch {
    // 同上：写入失败时静默忽略，下次启动会重新为空。
  }
}

/**
 * 维护最近 MAX_ITEMS 条搜索历史，写入 localStorage。
 * SSR 友好：window 不存在时仅返回内存数组。
 */
export function useSearchHistory() {
  const [history, setHistory] = useState<string[]>(() => readStorage());

  useEffect(() => {
    writeStorage(history);
  }, [history]);

  const push = useCallback((query: string) => {
    const trimmed = query.trim();
    if (trimmed === "") return;
    setHistory((prev) => {
      const without = prev.filter((entry) => entry !== trimmed);
      return [trimmed, ...without].slice(0, MAX_ITEMS);
    });
  }, []);

  const remove = useCallback((query: string) => {
    setHistory((prev) => prev.filter((entry) => entry !== query));
  }, []);

  const clear = useCallback(() => {
    setHistory([]);
  }, []);

  return { history, push, remove, clear };
}