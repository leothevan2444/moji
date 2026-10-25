import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowRight, faBinoculars } from "@fortawesome/free-solid-svg-icons";
import { useTranslation } from "react-i18next";
import { formatDateTime, formatRelative } from "../../utils";
import type { HomeSubscriptionCheckRunFieldsFragment, HomeSubscriptionChecksQuery } from "../../graphql/generated/graphql";

export type HomeCheckRunView = HomeSubscriptionCheckRunFieldsFragment;
export type HomeChecksView = Omit<HomeSubscriptionChecksQuery["homeSubscriptionChecks"], "activeRuns" | "recentRuns" | "latestScheduled"> & {
  activeRuns: HomeCheckRunView[];
  recentRuns: HomeCheckRunView[];
  latestScheduled: HomeCheckRunView | null;
};

function RunRow({ run, onOpen }: { run: HomeCheckRunView; onOpen: (id: string) => void }) {
  const { t } = useTranslation();
  const running = run.status === "RUNNING";
  const warning = ["FAILED", "PARTIAL", "INTERRUPTED"].includes(run.status);
  const progress = run.totalCount ? Math.round(run.completedCount / run.totalCount * 100) : 0;
  return <div className="home-check-row">
    <button type="button" className="home-check-open" onClick={() => onOpen(run.id)} aria-label={t("subscriptionChecks.openRun", { trigger: t(`subscriptionChecks.triggers.${run.trigger}`), time: formatDateTime(run.startedAt) })}>
      <span><strong>{t(`subscriptionChecks.triggers.${run.trigger}`)} · {t(`subscriptionChecks.scopes.${run.scope}`)}</strong><time dateTime={run.startedAt} title={formatDateTime(run.startedAt)}>{formatRelative(run.startedAt)}</time></span>
      <span className={warning ? "home-warning" : "home-muted"}>{t(`subscriptionChecks.states.${run.status}`)} <FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></span>
    </button>
    {running ? <>
      <p className="home-muted">{run.totalCount ? t("subscriptionChecks.progress", { completed: run.completedCount, total: run.totalCount }) : t("subscriptionChecks.loadingPerformers")}</p>
      {run.totalCount > 0 ? <div className="home-progress" role="progressbar" aria-label={t("subscriptionChecks.progressLabel")} aria-valuemin={0} aria-valuemax={run.totalCount} aria-valuenow={run.completedCount}><span style={{ width: `${progress}%` }} /></div> : null}
      {run.currentPerformers.length > 0 ? <p className="home-muted">{t("subscriptionChecks.currentPerformers", { names: run.currentPerformers.map(actor => actor.name || actor.performerId).join(", ") })}</p> : null}
    </> : <p className={warning ? "home-warning" : "home-muted"}>{t("subscriptionChecks.counts", { succeeded: run.succeededCount, failed: run.failedCount, skipped: run.skippedCount })}</p>}
  </div>;
}

export function HomeSubscriptionChecks({ snapshot, error, onRetry, onOpenRun }: { snapshot: HomeChecksView | null; error: string | null; onRetry: () => void; onOpenRun: (id: string) => void }) {
  const { t } = useTranslation();
  const latest = snapshot?.latestScheduled;
  return <section className="home-panel home-checks" id="subscription-checks" aria-labelledby="home-checks-title">
    <div className="home-panel-heading"><h2 id="home-checks-title">{t("subscriptionChecks.title")}{snapshot?.activeCount ? <span className="home-count">{snapshot.activeCount}</span> : null}</h2><FontAwesomeIcon icon={faBinoculars} aria-hidden="true" /></div>
    {error ? <div className="home-module-error" role="alert"><p>{snapshot ? t("workbench.refreshFailed") : error}</p><button type="button" onClick={onRetry}>{t("common.retry")}</button></div> : null}
    {!snapshot ? !error && <div className="home-loading" aria-label={t("subscriptionChecks.loading")}><span /><span /><span /></div> : <>
      <dl className="home-check-schedule">
        <div><dt>{t("subscriptionChecks.current")}</dt><dd>{t(snapshot.activeCount ? "subscriptionChecks.runningCount" : "subscriptionChecks.idle", { count: snapshot.activeCount })}</dd></div>
        <div><dt>{t("subscriptionChecks.schedule")}</dt><dd>{t(snapshot.enabled ? "subscriptionChecks.enabled" : "subscriptionChecks.disabled")}</dd></div>
        <div><dt>{t("subscriptionChecks.lastScheduled")}</dt><dd>{latest ? <button type="button" className="home-inline-button" onClick={() => onOpenRun(latest.id)}><time dateTime={latest.startedAt} title={formatDateTime(latest.startedAt)}>{formatRelative(latest.startedAt)}</time> · {t(`subscriptionChecks.states.${latest.status}`)} <FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></button> : t("subscriptionChecks.never")}</dd></div>
        <div><dt>{t("subscriptionChecks.nextScheduled")}</dt><dd>{snapshot.enabled && snapshot.nextRunAt ? <time dateTime={snapshot.nextRunAt} title={formatDateTime(snapshot.nextRunAt)}>{formatDateTime(snapshot.nextRunAt)}</time> : "—"}</dd></div>
      </dl>
      {snapshot.activeRuns.length > 0 ? <div className="home-check-list" aria-label={t("subscriptionChecks.current")}>{snapshot.activeRuns.map(run => <RunRow key={run.id} run={run} onOpen={onOpenRun} />)}</div> : null}
      {snapshot.activeCount > snapshot.activeRuns.length ? <p className="home-muted home-check-more">{t("subscriptionChecks.moreActive", { count: snapshot.activeCount - snapshot.activeRuns.length })}</p> : null}
      <details className="home-diagnostics home-check-history"><summary>{t("subscriptionChecks.history", { count: snapshot.recentRuns.length })}</summary>
        {snapshot.recentRuns.length ? <div className="home-check-list">{snapshot.recentRuns.map(run => <RunRow key={run.id} run={run} onOpen={onOpenRun} />)}</div> : <p className="home-empty home-empty--compact">{t("subscriptionChecks.noHistory")}</p>}
      </details>
    </>}
  </section>;
}
