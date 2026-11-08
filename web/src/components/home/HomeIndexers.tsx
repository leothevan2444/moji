import { useState } from "react";
import { Provider } from "urql";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowRight, faRotate } from "@fortawesome/free-solid-svg-icons";
import { settingsClient } from "../../graphql/client";
import { useHomeIndexers } from "../../hooks/useHomeIndexers";
import { describeQueryError } from "../../services/queryError";
import { IndexerDiagnosticsDrawer } from "../indexers/IndexerDiagnosticsDrawer";
import { formatDateTime } from "../../utils";

export function HomeIndexers() {
  return <Provider value={settingsClient}><Summary /></Provider>;
}

function Summary() {
  const { t, i18n } = useTranslation();
  const [hours, setHours] = useState(72);
  const { snapshot, error, fetching, refresh } = useHomeIndexers(hours);
  const [history, setHistory] = useState(false);
  const stats = snapshot?.hours === hours ? snapshot.statistics : null;
  const savedRecords = snapshot?.statistics?.records;
  const number = (value: number) => new Intl.NumberFormat(i18n.language, { maximumFractionDigits: 1 }).format(value);
  const enabled = snapshot ? number(snapshot.total) : "—";
  const metrics = [
    ["enabled", enabled],
    ["searches", stats ? number(stats.searches) : "—"],
    ["successRate", stats?.successRate == null ? "—" : `${number(stats.successRate)}%`],
    ["averageElapsed", !stats?.averageElapsedMs ? "-" : `${number(stats.averageElapsedMs / 1000)} ${t("indexerOverview.seconds")}`]
  ];
  const window = stats ? t("indexerOverview.window", { from: formatDateTime(stats.windowStartedAt), to: formatDateTime(stats.windowEndedAt) }) : undefined;
  return <section className="home-panel home-indexers" aria-labelledby="home-indexers-title">
    <div className="home-panel-heading">
      <h2 id="home-indexers-title">{t("homeIndexers.title")}</h2>
      <div className="profile-card__icons home-indexer-actions">
        <select className="home-indexer-period" aria-label={t("indexerOverview.period")} value={hours} onChange={event => setHours(Number(event.target.value))}>
          <option value={24}>24h</option><option value={72}>72h</option><option value={336}>{t("indexerOverview.twoWeeks")}</option>
        </select>
        <Link className="home-inline-button" to="/settings/indexers">{t("homeIndexers.manage")}</Link>
        <button type="button" className="home-inline-button" disabled={fetching} onClick={refresh} aria-label={t("homeIndexers.refresh")}><FontAwesomeIcon icon={faRotate} aria-hidden="true" /></button>
      </div>
    </div>
    {error ? <div className="home-module-error" role="status"><p>{snapshot ? t("workbench.refreshFailed") : null} {describeQueryError(error)}</p><button className="ghost-button" onClick={refresh}>{t("common.retry")}</button></div> : null}
    {!snapshot ? !error && <div className="home-loading" role="status" aria-label={t("common.loading")}><div className="skeleton" /></div> : <>
      <dl className="home-downloader-metrics" aria-busy={fetching}>{metrics.map(([key, value]) => <div key={key} title={key === "enabled" ? t("indexerOverview.enabledHint") : window}><dt>{t(`indexerOverview.${key}`)}</dt><dd>{value}</dd></div>)}</dl>
      {snapshot.statisticsErrorCode ? <p className="home-indexer-note" role="status">{t("indexerOverview.unavailable")}</p>
        : stats?.searches === 0 ? <p className="home-indexer-note home-muted" role="status">{t("indexerOverview.noSamples")}</p> : null}
      <div className="home-downloader-footer"><div className="home-downloader-status"><span>{savedRecords == null ? "—" : t("indexerOverview.records", { count: savedRecords })}</span></div><button type="button" className="home-more home-inline-button" onClick={() => setHistory(true)}>{t("indexerOverview.viewRecords")}<FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></button></div>
    </>}
    {history ? <IndexerDiagnosticsDrawer onClose={() => setHistory(false)} /> : null}
  </section>;
}
