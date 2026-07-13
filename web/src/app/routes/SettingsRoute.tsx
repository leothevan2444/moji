import { Component as ReactComponent, lazy, Suspense, type ErrorInfo, type ReactNode } from "react";
import { Navigate, useNavigate, useParams } from "react-router";
import { useTranslation } from "react-i18next";
import { LocaleSelect } from "../../components/common/LocaleSelect";
import { SettingsDraftProvider } from "../../components/settings/SettingsDraftStore";
import { SETTINGS_TABS } from "../../constants";
import type { SettingsTab } from "../../types";
import "../../styles/settings-shell.scss";

const tabs = {
  connections: lazy(() => import("../../components/settings/ConnectionsSettingsTab")),
  ingest: lazy(() => import("../../components/settings/IngestSettingsTab")),
  automation: lazy(() => import("../../components/settings/AutomationSettingsTab")),
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
  const { section = "connections" } = useParams();
  const navigate = useNavigate();
  if (!SETTINGS_TABS.includes(section as SettingsTab)) return <Navigate replace to="/settings/connections" />;
  const active = section as SettingsTab;
  const ActiveTab = tabs[active];
  const loading = <div className="skeleton skeleton-card" aria-label={t("common.loading")} />;
  const failed = <article className="drawer-card"><div className="drawer-card__head"><h3>{t(`settings.tabs.${active}`)}</h3></div><p className="settings-feedback tone-danger">{t("errors.moduleLoad")}</p><button type="button" onClick={() => window.location.reload()}>{t("common.retry")}</button></article>;

  return <section className="section-band">
    <div className="band-head"><div><h2 tabIndex={-1}>{t("settings.title")}</h2></div></div>
    <div className="drawer-stack">
      <LocaleSelect />
      <div className="settings-tabs">{SETTINGS_TABS.map((tab) => <button key={tab} type="button" className={`chip ${active === tab ? "is-active" : ""}`} onClick={() => navigate(`/settings/${tab}`)}>{t(`settings.tabs.${tab}`)}</button>)}</div>
      <SettingsDraftProvider><TabErrorBoundary resetKey={active} fallback={failed}><Suspense fallback={loading}><ActiveTab /></Suspense></TabErrorBoundary></SettingsDraftProvider>
    </div>
  </section>;
}
