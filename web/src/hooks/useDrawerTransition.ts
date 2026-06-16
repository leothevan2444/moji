import { useEffect, useState } from "react";
import type { DrawerKey } from "../types";

const DRAWER_EXIT_MS = 240;

/**
 * Manage drawer mounted/closing transitions so the leave animation can play
 * before the drawer is removed from the DOM.
 */
export function useDrawerTransition(drawer: DrawerKey) {
  const [renderedDrawer, setRenderedDrawer] = useState<Exclude<DrawerKey, null> | null>(null);
  const [drawerClosing, setDrawerClosing] = useState(false);

  useEffect(() => {
    if (drawer) {
      setRenderedDrawer(drawer);
      setDrawerClosing(false);
      return;
    }

    if (!renderedDrawer) return;

    setDrawerClosing(true);
    const timer = window.setTimeout(() => {
      setRenderedDrawer(null);
      setDrawerClosing(false);
    }, DRAWER_EXIT_MS);

    return () => window.clearTimeout(timer);
  }, [drawer, renderedDrawer]);

  const visibleDrawer = renderedDrawer ?? drawer;
  return { renderedDrawer, drawerClosing, visibleDrawer };
}
