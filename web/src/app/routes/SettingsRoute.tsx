import { Component as ReactComponent, lazy, Suspense, useEffect, useRef, type ErrorInfo, type ReactNode } from "react";
import { Navigate, useNavigate, useParams, useSearchParams } from "react-router";
import { useTranslation } from "react-i18next";
import { Provider } from "urql";
import { settingsClient } from "../../graphql/client";
import { SettingsLoading } from "../../components/settings/SettingsTabCommon";
import { SettingsAutosaveProvider } from "../../components/settings/SettingsAutosaveProvider";
import { SETTINGS_TABS } from "../../constants";
import type { SettingsTab } from "../../types";
import "../../styles/settings-shell.scss";
import { SETTINGS_PANELS, SettingsPanelContext } from "../../components/settings/SettingsPanels";

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
  const [search, setSearch] = useSearchParams();
  const sidebar = useRef<HTMLElement>(null);
  useEffect(() => {
    sidebar.current?.querySelector<HTMLElement>('[aria-current="page"]')?.scrollIntoView?.({block:"nearest",inline:"nearest"});
    if (window.scrollY > 80) window.scrollTo({top:0});
  }, [section]);
  const legacy: Record<string, string> = { connections: "indexers", ingest: "stash", automation: "tasks" };
  if (legacy[section]) return <Navigate replace to={`/settings/${legacy[section]}`} />;
  if (!SETTINGS_TABS.includes(section as SettingsTab)) return <Navigate replace to="/settings/indexers" />;
  const active = section as SettingsTab;
  const ActiveTab = tabs[active];
  const loading = <SettingsLoading title={t(`settings.tabs.${active}`)} />;
  const failed = <article className="drawer-card"><div className="drawer-card__head"><h3>{t(`settings.tabs.${active}`)}</h3></div><p className="settings-feedback tone-danger">{t("errors.moduleLoad")}</p><button type="button" onClick={() => window.location.reload()}>{t("common.retry")}</button></article>;

  const panels: readonly string[] = SETTINGS_PANELS[active];
  const requested = active === "indexers" && search.get("panel") === "queryOrder" ? "sites" : search.get("panel");
  const panel = requested && panels.includes(requested) ? requested : panels[0] ?? null;
  const selectPanel = (next: string) => setSearch(current => { const value = new URLSearchParams(current); value.set("panel", next); return value; });

  return <section className="section-band settings-workspace">
    <aside className="settings-sidebar">
      <p className="settings-sidebar__label">{t("settings.title")}</p>
      <nav ref={sidebar} className="settings-tabs" aria-label={t("settings.title")}>
        {SETTINGS_TABS.map(tab => <button key={tab} type="button" aria-current={active === tab ? "page" : undefined}
          className={`settings-sidebar__item${active === tab ? " is-active" : ""}`} onClick={() => navigate(`/settings/${tab}`)}>{t(`settings.tabs.${tab}`)}</button>)}
      </nav>
    </aside>
    <div className="settings-main">
      <header className="settings-main__head"><h2>{t(`settings.tabs.${active}`)}</h2></header>
      {panels.length ? <div className="settings-subtabs" role="tablist" aria-label={t(`settings.tabs.${active}`)}>
        {panels.map((item, index) => <button key={item} type="button" role="tab" id={`settings-tab-${item}`} aria-selected={panel === item}
          aria-controls="settings-panel" tabIndex={panel === item ? 0 : -1} onClick={() => selectPanel(item)}
          onKeyDown={event => {
            const rtl = document.documentElement.dir === "rtl";
            const direction = event.key === "ArrowRight" ? (rtl ? -1 : 1) : event.key === "ArrowLeft" ? (rtl ? 1 : -1) : 0;
            const target = event.key === "Home" ? 0 : event.key === "End" ? panels.length - 1 : direction ? (index + direction + panels.length) % panels.length : null;
            if (target === null) return;
            event.preventDefault(); selectPanel(panels[target]);
            document.getElementById(`settings-tab-${panels[target]}`)?.focus();
          }}>{t(active === "indexers" && item === "connection" ? "indexers.external" : `settingsLayout.panels.${item}`)}</button>)}
      </div> : null}
      <div id="settings-panel" className="settings-main__panel" role={panels.length ? "tabpanel" : undefined} aria-labelledby={panel ? `settings-tab-${panel}` : undefined}>
        <Provider value={settingsClient}><SettingsAutosaveProvider><SettingsPanelContext.Provider value={panel}>
          <TabErrorBoundary resetKey={active} fallback={failed}><Suspense fallback={loading}><ActiveTab /></Suspense></TabErrorBoundary>
        </SettingsPanelContext.Provider></SettingsAutosaveProvider></Provider>
      </div>
    </div>
  </section>;
}
