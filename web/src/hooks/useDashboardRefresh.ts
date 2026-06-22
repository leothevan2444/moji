import { useEffect } from "react";

type RefreshFn = (opts?: { requestPolicy: string }) => void;

/**
 * Periodically refresh the dashboard query while the document is visible.
 *
 *  - Uses Page Visibility API to skip refresh ticks when the tab is hidden.
 *  - Re-aligns the timer when the tab becomes visible again so users don't
 *    wait a full interval after returning to the tab.
 *  - Mirrors the setInterval pattern in [components/settings/SettingsPanel.tsx].
 */
export function useDashboardRefresh(
  refresh: RefreshFn,
  intervalMs = 30000
) {
  useEffect(() => {
    let timer: number | null = null;

    const tick = () => {
      if (typeof document !== "undefined" && document.visibilityState === "visible") {
        try {
          refresh({ requestPolicy: "network-only" });
        } catch {
          // ignore — caller (urql) handles its own error surfaces
        }
      }
    };

    const start = () => {
      if (timer != null) window.clearInterval(timer);
      timer = window.setInterval(tick, intervalMs);
    };

    const onVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        tick();
        start();
      }
    };

    tick();
    start();
    document.addEventListener("visibilitychange", onVisibilityChange);

    return () => {
      if (timer != null) window.clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [refresh, intervalMs]);
}