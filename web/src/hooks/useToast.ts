import { useEffect, useRef, useState } from "react";
import type { ToastItem, ToastTone } from "../types";
import { TOAST_LIFETIME_MS, TOAST_EXIT_MS } from "../constants";
import { useTranslation } from "react-i18next";

// 模块级单调计数器：避免连续 1ms 内 pushToast 时 Date.now() 相同，
// 同时去掉 Math.random 引入的潜在重复。
let nextToastId = 0;

export function useToast() {
  const { t } = useTranslation();
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

  const pushToast = (tone: ToastTone, message: string, lifetimeMs: number = TOAST_LIFETIME_MS) => {
    const id = ++nextToastId;
    setToasts((current) => [...current, { id, tone, message, phase: "entering", lifetimeMs }]);
    const exit = window.setTimeout(() => {
      setToasts((current) => current.map((item) => (item.id === id ? { ...item, phase: "leaving" } : item)));
    }, lifetimeMs - TOAST_EXIT_MS);
    const remove = window.setTimeout(() => {
      setToasts((current) => current.filter((item) => item.id !== id));
      toastTimersRef.current.delete(id);
    }, lifetimeMs);
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
      pushToast("tone-danger", t("toast.copyFailed"));
    }
  };

  return { toasts, pushToast, dismissToast, copyText };
}
