import { Component as ReactComponent, lazy, Suspense, type ErrorInfo, type ReactNode } from "react";
import { Navigate, useNavigate, useParams } from "react-router";
import { useTranslation } from "react-i18next";
import { Provider } from "urql";
import { settingsClient } from "../../graphql/client";
import { SettingsLoading } from "../../components/settings/SettingsTabCommon";
import { SettingsDraftProvider } from "../../components/settings/SettingsDraftStore";
import { SETTINGS_TABS } from "../../constants";
import type { SettingsTab } from "../../types";
import "../../styles/settings-shell.scss";

const tabs = {
  indexers: lazy(() => import("../../components/settings/IndexersSettingsTab")),
  stash: lazy(() => import("../../components/settings/StashSettingsTab")),
  tasks: lazy(() => import("../../components/settings/TaskSubscriptionSettingsTab")),
  downloader: lazy(() => import("../../components/settings/DownloaderSettingsTab")),
  system: lazy(() => import("../../components/settings/SystemSettingsTab")),
  logs: lazy(() => import("../../components/settings/LogsSettingsTab")),
  about: lazy(() => import("../../components/settings/AboutSettingsTab"))
} satisfies Record<SettingsTab, ReturnType<typeof lazy>>;

class TabErrorBoundary extends ReactComponent<{ resetKey: string; fallback: ReactNode; children: ReactNode }, { failed: boolean }> {
  state = { failed: false };
  static getDerivedStateFromError() { return { failed: true }; }
  componentDidCatch(error: Error, info: ErrorInfo) { if (import.meta.env.DEV) console.error("Settings tab failed to load", error, info); }
  componentDidUpdate(previous: Readonly<{ resetKey: string }>) { if (previous.resetKey !== this.props.resetKey && this.state.failed) this.setState({ failed: false }); }
  render() { return this.state.failed ? this.props.fallback : this.props.children; }
}

export function Component() {
  const { t } = useTranslation();
  const { section = "indexers" } = useParams();
  const navigate = useNavigate();
  const legacy: Record<string, string> = { connections: "indexers", ingest: "stash", automation: "tasks" };
  if (legacy[section]) return <Navigate replace to={`/settings/${legacy[section]}`} />;
  if (!SETTINGS_TABS.includes(section as SettingsTab)) return <Navigate replace to="/settings/indexers" />;
  const active = section as SettingsTab;
  const ActiveTab = tabs[active];
  const loading = <SettingsLoading title={t(`settings.tabs.${active}`)} />;
  const failed = <article className="drawer-card"><div className="drawer-card__head"><h3>{t(`settings.tabs.${active}`)}</h3></div><p className="settings-feedback tone-danger">{t("errors.moduleLoad")}</p><button type="button" onClick={() => window.location.reload()}>{t("common.retry")}</button></article>;

  return <section className="section-band settings-workspace">
    <div className="band-head settings-workspace__head"><div><h2 tabIndex={-1}>{t("settings.title")}</h2></div>
      <nav className="settings-tabs" aria-label={t("settings.title")}>{SETTINGS_TABS.map((tab) => <button key={tab} type="button" aria-current={active === tab ? "page" : undefined} className={`chip ${active === tab ? "is-active" : ""}`} onClick={() => navigate(`/settings/${tab}`)}>{t(`settings.tabs.${tab}`)}</button>)}</nav>
    </div>
    <div className="drawer-stack">
      <Provider value={settingsClient}><SettingsDraftProvider><TabErrorBoundary resetKey={active} fallback={failed}><Suspense fallback={loading}><ActiveTab /></Suspense></TabErrorBoundary></SettingsDraftProvider></Provider>
    </div>
  </section>;
}
