import { useAppTimezone } from "../../hooks/useAppTimezone";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowUpRightFromSquare, faRotate } from "@fortawesome/free-solid-svg-icons";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import type { HomeServiceStatusQuery } from "../../graphql/generated/graphql";
import { formatDateTime } from "../../utils";

interface Props {
  status: HomeServiceStatusQuery["settingsStatus"] | null;
  url: string;
  error: string | null;
  refreshing?: boolean;
  onRefresh: () => void;
}

function stashLink(value: string) {
  try {
    const url = new URL(value);
    if (!["http:", "https:"].includes(url.protocol)) return null;
    url.pathname = url.pathname.replace(/\/graphql\/?$/, "/");
    return url.href;
  } catch { return null; }
}

export function HomeStash({ status, url, error, refreshing = false, onRefresh }: Props) {
  useAppTimezone();
  const { t, i18n } = useTranslation();
  const state = !status || error ? "unknown" : !status.stash.configured ? "unconfigured"
    : status.stash.ready ? "connected" : status.stashStats?.lastError?.trim() ? "unavailable" : "unknown";
  const stats = status?.stashStats;
  const known = Boolean(stats?.okAt);
  const number = (value: number | null | undefined) => value == null ? "—" : new Intl.NumberFormat(i18n.language).format(value);
  const metrics = [
    ["connection", t(`homeStash.states.${state}`)],
    ["version", known && stats?.version ? stats.version : "—"],
    ["scenes", known ? number(stats?.sceneCount) : "—"],
    ["pendingScans", known ? number(stats?.pendingMojiScanCount) : "—"]
  ];
  const href = stashLink(url);

  return <section className="home-panel home-stash" aria-labelledby="home-stash-title">
    <div className="home-panel-heading">
      <h2 id="home-stash-title">Stash</h2>
      <div className="profile-card__icons">
        <Link className="home-inline-button" to="/settings/stash?panel=connection" aria-label={t("homeStash.settings")}>{t("home.configure")}</Link>
        <button type="button" className="home-inline-button" onClick={onRefresh} disabled={refreshing} aria-busy={refreshing} aria-label={t("homeStash.refresh")}><FontAwesomeIcon icon={faRotate} aria-hidden="true" /></button>
      </div>
    </div>
    {error ? <div className="home-module-error" role="status"><p>{status ? t("workbench.refreshFailed") : null} {error}</p><button type="button" className="ghost-button" onClick={onRefresh}>{t("common.retry")}</button></div> : null}
    {!status && !error ? <div className="home-loading" role="status" aria-label={t("common.loading")}><div className="skeleton" /></div> : <>
      <dl className="home-downloader-metrics" aria-busy={refreshing}>{metrics.map(([key, value]) => <div key={key}>
        <dt>{t(`homeStash.${key}`)}</dt><dd className={key === "connection" ? state === "connected" ? "home-success" : "home-warning" : undefined}>{value}</dd>
      </div>)}</dl>
      {state !== "connected" && !error ? <p className="home-stash-note home-warning" role="status">{t(`homeStash.impacts.${state}`)}</p> : null}
      {href || (state !== "connected" && stats?.okAt) ? <div className="home-downloader-footer">
        <div className="home-downloader-status">{state !== "connected" && stats?.okAt ? <span>{t("homeStash.lastSuccess", { time: formatDateTime(stats.okAt) })}</span> : null}</div>
        {href ? <a className="home-more home-inline-button" href={href} target="_blank" rel="noopener noreferrer">{t("homeStash.open")}<FontAwesomeIcon icon={faArrowUpRightFromSquare} aria-hidden="true" /></a> : null}
      </div> : null}
    </>}
  </section>;
}
