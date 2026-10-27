import { useEffect, useState, useSyncExternalStore } from "react";
import { useSubscription } from "urql";
import { useTranslation } from "react-i18next";
import { DatabasePerformanceDocumentDocument } from "../../graphql/generated/graphql";
import { getGraphQLConnectionSnapshot, subscribeGraphQLConnection } from "../../graphql/client";
import { describeQueryError } from "../../services/queryError";
import { formatBytes, formatDateTime } from "../../utils";
import "../../styles/performance.scss";

export default function PerformanceDrawer() {
  const { t, i18n } = useTranslation();
  const [visible, setVisible] = useState(() => document.visibilityState === "visible");
  useEffect(() => {
    const update = () => setVisible(document.visibilityState === "visible");
    document.addEventListener("visibilitychange", update);
    return () => document.removeEventListener("visibilitychange", update);
  }, []);
  const [{ data, fetching, error }, restart] = useSubscription({ query: DatabasePerformanceDocumentDocument, pause: !visible });
  const connection = useSyncExternalStore(subscribeGraphQLConnection, getGraphQLConnectionSnapshot, getGraphQLConnectionSnapshot);
  const metrics = data?.databasePerformance;
  const [expired, setExpired] = useState(true);
  useEffect(() => {
    if (!metrics?.sampledAt) { setExpired(true); return; }
    const remaining = Date.parse(metrics.sampledAt) + 15000 - Date.now();
    setExpired(remaining <= 0);
    // A local freshness deadline only; this timer never requests any data.
    const deadline = window.setTimeout(() => setExpired(true), Math.max(0, remaining));
    return () => window.clearTimeout(deadline);
  }, [metrics?.sampledAt]);
  const stale = expired || connection.status !== "connected";
  const number = (value: number) => new Intl.NumberFormat(i18n.resolvedLanguage, { maximumFractionDigits: 2 }).format(value);
  const duration = (value: number) => `${number(value)} ms`;
  return <div className="drawer-stack performance-panel">
    {error || metrics?.lastError || stale ? <div className="performance-panel__status" role="status"><span className="status-pill tone-warning">{t("performance.stale")}</span></div> : null}
    {error || metrics?.lastError ? <div className="empty-card" role="alert"><p>{error ? describeQueryError(error) : metrics?.lastError}</p><button type="button" onClick={() => restart()}>{t("common.retry")}</button></div> : null}
    {!metrics && fetching ? <div className="skeleton skeleton-card" aria-label={t("common.loading")} /> : null}
    {metrics ? <p className="performance-panel__note">{t("performance.since", { time: formatDateTime(metrics.startedAt) })}</p> : null}
    {metrics?.databases.map((database) => <article className="drawer-card performance-card" key={database.role}>
      <h3>{t(`performance.roles.${database.role}`)}</h3>
      <div className="performance-card__summary">
        <div><span>{t("performance.writeRate")}</span><strong>{metrics.intervalSeconds > 0 ? number(database.writesPerSecond) : "—"}<small> /s</small></strong></div>
        <div><span>{t("performance.writeSlots")}</span><strong>{database.queued}<small> / {database.queueCapacity}</small></strong></div>
        <div><span>{t("performance.databaseSize")}</span><strong>{formatBytes(Number(database.databaseBytes))}</strong></div>
        <div><span>{t("performance.walSize")}</span><strong>{formatBytes(Number(database.walBytes))}</strong></div>
      </div>
      <dl className="performance-card__details">
        <div><dt>{t("performance.averageWait")}</dt><dd>{metrics.intervalSeconds > 0 && database.writesPerSecond > 0 ? duration(database.averageWaitMilliseconds) : "—"}</dd></div>
        <div><dt>{t("performance.averageExecution")}</dt><dd>{metrics.intervalSeconds > 0 && database.writesPerSecond > 0 ? duration(database.averageExecutionMilliseconds) : "—"}</dd></div>
        <div><dt>{t("performance.totalWait")}</dt><dd>{duration(database.totalWaitMilliseconds)}</dd></div>
        <div><dt>{t("performance.totalExecution")}</dt><dd>{duration(database.totalExecutionMilliseconds)}</dd></div>
        <div><dt>{t("performance.freeSpace")}</dt><dd>{formatBytes(Number(database.freeBytes))}</dd></div>
        <div><dt>{t("performance.writeAttempts")}</dt><dd>{number(Number(database.writeAttempts))}</dd></div>
        <div><dt>{t("performance.writeErrors")}</dt><dd className={Number(database.writeErrors) ? "tone-danger" : undefined}>{number(Number(database.writeErrors))}</dd></div>
        <div><dt>{t("performance.busyRetries")}</dt><dd>{number(Number(database.busyRetries))}</dd></div>
        <div><dt>{t("performance.queueRejected")}</dt><dd>{number(Number(database.queueRejected))}</dd></div>
        <div><dt>{t("performance.queueCanceled")}</dt><dd>{number(Number(database.queueCanceled))}</dd></div>
      </dl>
      <div className="performance-card__table-wrap"><table><caption>{t("performance.pools")}</caption><thead><tr><th scope="col">{t("performance.pool")}</th><th scope="col">{t("performance.connections")}</th><th scope="col">{t("performance.inUse")}</th><th scope="col">{t("performance.poolWaits")}</th><th scope="col">{t("performance.poolWaitTime")}</th></tr></thead><tbody>{(["reader", "writer"] as const).map((role) => <tr key={role}><th scope="row">{t(`performance.${role}`)}</th><td>{database[role].open} / {database[role].limit}</td><td>{database[role].inUse}</td><td>{number(Number(database[role].waitCount))}</td><td>{duration(database[role].waitMilliseconds)}</td></tr>)}</tbody></table></div>
    </article>)}
    {metrics ? <p className="performance-panel__note">{t("performance.scope")}</p> : null}
  </div>;
}
