import { useEffect, useLayoutEffect, useState } from "react";

const STORAGE_KEY = "moji:nsfw-mask";

function readMasked(): boolean {
  try {
    return window.localStorage.getItem(STORAGE_KEY) === "true";
  } catch {
    return false;
  }
}

/** Browser-only preference, owned by the persistent application header. */
export function useNsfwMask() {
  const [masked, setMasked] = useState(readMasked);

  // Apply before paint, including when restoring the preference on a reload.
  // The document attribute also covers media rendered in drawer portals.
  useLayoutEffect(() => {
    document.documentElement.dataset.nsfwMask = String(masked);
  }, [masked]);

  useEffect(() => {
    const sync = (event: StorageEvent) => {
      if (event.storageArea === window.localStorage && (event.key === STORAGE_KEY || event.key === null)) {
        setMasked(readMasked());
      }
    };
    window.addEventListener("storage", sync);
    return () => window.removeEventListener("storage", sync);
  }, []);

  const toggle = () => {
    const next = !masked;
    setMasked(next);
    try {
      window.localStorage.setItem(STORAGE_KEY, String(next));
    } catch {
      // Storage can be unavailable; masking still works for this page session.
    }
  };

  return { masked, toggle };
}
