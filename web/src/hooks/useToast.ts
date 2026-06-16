import { useEffect, useRef, useState } from "react";
import type { ToastItem, ToastTone } from "../types";
import { TOAST_LIFETIME_MS, TOAST_EXIT_MS } from "../constants";

export function useToast() {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const toastTimersRef = useRef(new Map<number, { exit: number; remove: number }>());

  // Clean up all timers on unmount
  useEffect(() => {
    return () => {
      for (const timers of toastTimersRef.current.values()) {
        window.clearTimeout(timers.exit);
        window.clearTimeout(timers.remove);
      }
      toastTimersRef.current.clear();
    };
  }, []);

  const pushToast = (tone: ToastTone, message: string) => {
    const id = Date.now() + Math.floor(Math.random() * 1000);
    setToasts((current) => [...current, { id, tone, message, phase: "entering" }]);
    const exit = window.setTimeout(() => {
      setToasts((current) => current.map((item) => (item.id === id ? { ...item, phase: "leaving" } : item)));
    }, TOAST_LIFETIME_MS - TOAST_EXIT_MS);
    const remove = window.setTimeout(() => {
      setToasts((current) => current.filter((item) => item.id !== id));
      toastTimersRef.current.delete(id);
    }, TOAST_LIFETIME_MS);
    toastTimersRef.current.set(id, { exit, remove });
  };

  const dismissToast = (id: number) => {
    const timers = toastTimersRef.current.get(id);
    if (timers) {
      window.clearTimeout(timers.exit);
      window.clearTimeout(timers.remove);
      toastTimersRef.current.delete(id);
    }
    setToasts((current) => current.map((item) => (item.id === id ? { ...item, phase: "leaving" } : item)));
    const remove = window.setTimeout(() => {
      setToasts((current) => current.filter((item) => item.id !== id));
    }, TOAST_EXIT_MS);
    toastTimersRef.current.set(id, { exit: 0, remove });
  };

  const copyText = async (value: string, successMessage: string) => {
    try {
      await navigator.clipboard.writeText(value);
      pushToast("tone-success", successMessage);
    } catch {
      pushToast("tone-danger", "复制失败，请检查浏览器剪贴板权限。");
    }
  };

  return { toasts, pushToast, dismissToast, copyText };
}
