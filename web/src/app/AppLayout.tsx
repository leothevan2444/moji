import { lazy, Suspense, useState } from "react";
import { Outlet } from "react-router";
import { Drawer } from "../components/layout/Drawer";
import { Header } from "../components/layout/Header";
import { ToastStack } from "../components/layout/ToastStack";
import { useTheme } from "../hooks/useTheme";
import { useToast } from "../hooks/useToast";
import type { HelpTopicId } from "../help";
import { HELP_TOPICS } from "../help";

const HelpDrawer = lazy(() => import("../components/drawers/HelpDrawer").then((module) => ({ default: module.HelpDrawer })));

export interface AppOutletContext {
  pushToast: ReturnType<typeof useToast>["pushToast"];
  copyText: ReturnType<typeof useToast>["copyText"];
  openHelp: () => void;
}

export function AppLayout() {
  const theme = useTheme();
  const { toasts, pushToast, dismissToast, copyText } = useToast();
  const [helpOpen, setHelpOpen] = useState(false);
  const [helpTopicId, setHelpTopicId] = useState<HelpTopicId>(HELP_TOPICS[0].id);

  return (
    <div className="app-shell">
      <ToastStack toasts={toasts} onDismiss={dismissToast} />
      <div className="ambient ambient-a" />
      <div className="ambient ambient-b" />
      <div className="ambient ambient-c" />
      <Header onOpenHelp={() => setHelpOpen(true)} theme={theme} />
      <main className="content">
        <Suspense fallback={<div className="skeleton skeleton-card" aria-label="页面加载中" />}>
          <Outlet context={{ pushToast, copyText, openHelp: () => setHelpOpen(true) } satisfies AppOutletContext} />
        </Suspense>
      </main>
      {helpOpen ? (
        <Drawer visibleDrawer="help" closing={false} title="Markdown 帮助" onClose={() => setHelpOpen(false)}>
          <Suspense fallback={<div className="skeleton skeleton-card" aria-label="帮助加载中" />}>
            <HelpDrawer topicId={helpTopicId} onTopicChange={setHelpTopicId} />
          </Suspense>
        </Drawer>
      ) : null}
    </div>
  );
}
