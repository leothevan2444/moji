import { useEffect } from "react";

export function useSubscriptionCheckPolling(enabled: boolean, active: boolean, refresh: () => void) {
  useEffect(() => {
    if (!enabled) return;
    const onVisible = () => { if (document.visibilityState !== "hidden") refresh(); };
    const timer = window.setInterval(onVisible, active ? 2000 : 10000);
    window.addEventListener("focus", onVisible);
    document.addEventListener("visibilitychange", onVisible);
    return () => { window.clearInterval(timer); window.removeEventListener("focus", onVisible); document.removeEventListener("visibilitychange", onVisible); };
  }, [enabled, active, refresh]);
}
