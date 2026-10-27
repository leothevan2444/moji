import { Provider } from "urql";
import { settingsClient } from "../graphql/client";
import { useDeveloperMode } from "../hooks/useDeveloperMode";
import { lazy, Suspense, useEffect, useState } from "react";
import { Outlet, useLocation } from "react-router";
import { Drawer } from "../components/layout/Drawer";
import { Header } from "../components/layout/Header";
import { ToastStack } from "../components/layout/ToastStack";
import { useTheme } from "../hooks/useTheme";
import { useToast } from "../hooks/useToast";
import type { HelpTopicId } from "../help/types";
import { useTranslation } from "react-i18next";
import { TaskProcessingNotice } from "../components/layout/TaskProcessingNotice";
import { TaskEventBridge } from "../components/events/TaskEventBridge";

const PerformanceDrawer = lazy(() => import("../components/drawers/PerformanceDrawer"));

const HelpDrawer = lazy(() => import("../components/drawers/HelpDrawer").then((module) => ({ default: module.HelpDrawer })));

export interface AppOutletContext {
  onDeveloperModeChange?: (enabled: boolean) => void;
  pushToast: ReturnType<typeof useToast>["pushToast"];
  copyText: ReturnType<typeof useToast>["copyText"];
  openHelp: () => void;
}

export function AppLayout() {
  const location = useLocation();
  const { t, i18n } = useTranslation();
  const theme = useTheme();
  const { toasts, pushToast, dismissToast, copyText } = useToast();
  const [helpOpen, setHelpOpen] = useState(false);
  const developerMode = useDeveloperMode();
  const [performanceOpen, setPerformanceOpen] = useState(false);
  useEffect(() => { if (!developerMode.enabled) setPerformanceOpen(false); }, [developerMode.enabled]);
  const [helpTopicId, setHelpTopicId] = useState<HelpTopicId>("introduction");
  useEffect(() => {
    const path = location.pathname;
    const key = path.startsWith("/tasks") ? "tasks" : path.startsWith("/performers") ? "performers" : path.startsWith("/discover") ? "discover" : path.startsWith("/settings") ? "settings" : path === "/stats" ? "stats" : "home";
    document.title = t(`titles.${key}`);
  }, [i18n.resolvedLanguage, location.pathname, t]);

  return (
    <div className="app-shell">
      <TaskEventBridge />
      <ToastStack toasts={toasts} onDismiss={dismissToast} />
      <div className="ambient ambient-a" />
      <div className="ambient ambient-b" />
      <div className="ambient ambient-c" />
      <Header developerMode={developerMode.enabled} onOpenPerformance={() => setPerformanceOpen(true)} onOpenHelp={() => setHelpOpen(true)} theme={theme} />
      <main className="content">
        <TaskProcessingNotice />
        <Suspense fallback={<div className="skeleton skeleton-card" aria-label={t("common.loading")} />}>
          <Outlet context={{ pushToast, copyText, openHelp: () => setHelpOpen(true), onDeveloperModeChange: developerMode.update } satisfies AppOutletContext} />
        </Suspense>
      </main>
      {developerMode.enabled && performanceOpen ? (
        <Drawer visibleDrawer="performance" closing={false} title={t("performance.title")} onClose={() => setPerformanceOpen(false)}>
          <Provider value={settingsClient}><Suspense fallback={<div className="skeleton skeleton-card" aria-label={t("common.loading")} />}><PerformanceDrawer /></Suspense></Provider>
        </Drawer>
      ) : null}
      {helpOpen ? (
        <Drawer visibleDrawer="help" closing={false} title={t("help.title")} onClose={() => setHelpOpen(false)}>
          <Suspense fallback={<div className="skeleton skeleton-card" aria-label={t("common.helpLoading")} />}>
            <HelpDrawer topicId={helpTopicId} onTopicChange={setHelpTopicId} />
          </Suspense>
        </Drawer>
      ) : null}
    </div>
  );
}
