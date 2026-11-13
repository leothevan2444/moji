import { createPortal } from "react-dom";
import { useState } from "react";
import { useQuery } from "urql";
import { Drawer } from "../layout/Drawer";
import { SubscriptionCheckHistoryDocument, HomeSubscriptionCheckRunFieldsFragmentDoc } from "../../graphql/generated/graphql";
import { useFragment as readFragment } from "../../graphql/generated/fragment-masking";
import { describeQueryError } from "../../services/queryError";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowRight, faBinoculars, faCheck, faCircleExclamation } from "@fortawesome/free-solid-svg-icons";
import { useTranslation } from "react-i18next";
import { formatDateTime, formatRelative } from "../../utils";
import type { HomeSubscriptionCheckRunFieldsFragment, HomeSubscriptionChecksQuery } from "../../graphql/generated/graphql";

export type HomeCheckRunView = HomeSubscriptionCheckRunFieldsFragment;
export type HomeChecksView = Omit<HomeSubscriptionChecksQuery["homeSubscriptionChecks"], "activeRuns" | "recentRuns" | "latestScheduled" | "latestFinishedScheduled"> & {
  activeRuns: HomeCheckRunView[];
  recentRuns: HomeCheckRunView[];
  latestScheduled: HomeCheckRunView | null;
  latestFinishedScheduled: HomeCheckRunView | null;
};

function RunRow({ run, onOpen, compact = false }: { run: HomeCheckRunView; onOpen: (id: string) => void; compact?: boolean }) {
  const { t } = useTranslation();
  const running = run.status === "RUNNING";
  const warning = ["FAILED", "PARTIAL", "INTERRUPTED"].includes(run.status);
  const progress = run.totalCount ? Math.round(run.completedCount / run.totalCount * 100) : 0;
  if (compact && running) return <div className="home-check-row home-check-row--compact">
    <button type="button" className="home-check-open" onClick={() => onOpen(run.id)} aria-label={t("subscriptionChecks.openRun", { trigger: t(`subscriptionChecks.triggers.${run.trigger}`), time: formatDateTime(run.startedAt) })}>
      <strong>{run.currentPerformers.length ? t("subscriptionChecks.currentPerformers", { names: run.currentPerformers.map(actor => actor.name || actor.performerId).join(", ") }) : t(`subscriptionChecks.triggers.${run.trigger}`)}</strong>
      <span className="home-muted">{run.totalCount ? t("subscriptionChecks.progress", { completed: run.completedCount, total: run.totalCount }) : t("subscriptionChecks.loadingPerformers")} <FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></span>
    </button>
    {run.totalCount > 0 ? <div className="home-progress" role="progressbar" aria-label={t("subscriptionChecks.progressLabel")} aria-valuemin={0} aria-valuemax={run.totalCount} aria-valuenow={run.completedCount}><span style={{ width: `${progress}%` }} /></div> : null}
  </div>;
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
  const { t, i18n } = useTranslation();
  const latest = snapshot?.latestFinishedScheduled;
  const historyWarning = latest && ["FAILED", "PARTIAL", "INTERRUPTED"].includes(latest.status);
  const historySuccess = latest?.status === "SUCCEEDED";
  const historyDescription = latest ? t("subscriptionChecks.historyLatest", { status: t(`subscriptionChecks.states.${latest.status}`), time: formatDateTime(latest.finishedAt ?? latest.startedAt) }) : t(snapshot ? "subscriptionChecks.historyNone" : "subscriptionChecks.historyUnavailable");
  const [historyOpen,setHistoryOpen]=useState(false);
  return <section className="home-panel home-checks" id="subscription-checks" aria-labelledby="home-checks-title">
    <div className="home-panel-heading"><h2 id="home-checks-title">{t("subscriptionChecks.title")}{snapshot?.activeCount ? <span className="home-count">{snapshot.activeCount}</span> : null}</h2><div className="home-check-heading-actions"><button type="button" className={`home-inline-button home-check-history-button${historyWarning ? " home-warning" : historySuccess ? " home-success" : ""}`} aria-haspopup="dialog" aria-label={`${t("performerActivity.history")} · ${historyDescription}`} title={historyDescription} onClick={() => setHistoryOpen(true)}>{historyWarning || historySuccess ? <FontAwesomeIcon icon={historyWarning ? faCircleExclamation : faCheck} aria-hidden="true" /> : null}{t("performerActivity.history")}</button><FontAwesomeIcon icon={faBinoculars} aria-hidden="true" /></div></div>
    {error ? <div className="home-module-error" role="alert"><p>{snapshot ? t("workbench.refreshFailed") : error}</p><button type="button" onClick={onRetry}>{t("common.retry")}</button></div> : null}
    {!snapshot ? !error && <div className="home-loading" aria-label={t("subscriptionChecks.loading")}><span /><span /><span /></div> : <>
      <div className="home-check-summary">
        <div className="home-check-status-line"><span className="home-muted">{t(snapshot.enabled ? "subscriptionChecks.automaticEnabled" : "subscriptionChecks.automaticDisabled")}</span><span className="home-muted">{t(snapshot.activeCount ? "subscriptionChecks.runningCount" : "subscriptionChecks.idle", { count: snapshot.activeCount })}</span></div>
        <div className="home-check-next"><span className="home-muted">{t("subscriptionChecks.nextScheduled")}</span><span>{snapshot.enabled && snapshot.nextRunAt ? <time dateTime={snapshot.nextRunAt} title={formatDateTime(snapshot.nextRunAt)}>{formatNextRun(snapshot.nextRunAt, i18n.language)}</time> : "—"}</span></div>

      </div>
      <div className="home-check-statistics" title={t("subscriptionChecks.statisticsHelp", { count: snapshot.statistics.runCount })} tabIndex={0} aria-label={t("subscriptionChecks.statisticsHelp", { count: snapshot.statistics.runCount })}>
        <span className="home-muted">{t("subscriptionChecks.recentStatistics")}</span>
        <dl>
          <div><dt>{t("subscriptionChecks.statSucceeded")}</dt><dd>{snapshot.statistics.succeededCount}</dd></div>
          <div><dt>{t("subscriptionChecks.statFailed")}</dt><dd>{snapshot.statistics.failedCount}</dd></div>
          <div><dt>{t("subscriptionChecks.statSkipped")}</dt><dd>{snapshot.statistics.skippedCount}</dd></div>
        </dl>
      </div>
      {snapshot.activeRuns.length > 0 ? <div className="home-check-list" aria-label={t("subscriptionChecks.current")}>{snapshot.activeRuns.map(run => <RunRow key={run.id} run={run} compact onOpen={onOpenRun} />)}</div> : null}
      {snapshot.activeCount > snapshot.activeRuns.length ? <button type="button" className="home-inline-button home-check-more" onClick={() => setHistoryOpen(true)}>{t("subscriptionChecks.moreActive", { count: snapshot.activeCount - snapshot.activeRuns.length })}</button> : null}
    </>}
    {historyOpen ? createPortal(<Drawer visibleDrawer="subscription-history" closing={false} title={t("performerActivity.history")} onClose={() => setHistoryOpen(false)}><CheckHistory onOpen={id => {setHistoryOpen(false);onOpenRun(id);}} /></Drawer>, document.body) : null}
  </section>;
}

function CheckHistory({onOpen}:{onOpen:(id:string)=>void}) {
 const {t}=useTranslation();
 const [{data,error,fetching},refresh]=useQuery({query:SubscriptionCheckHistoryDocument,requestPolicy:"network-only"});
 return <div className="home-check-list">
  {error ? <div role="alert"><p>{describeQueryError(error)}</p><button type="button" onClick={()=>refresh({requestPolicy:"network-only"})}>{t("common.retry")}</button></div> : null}
  {fetching && !data ? <p role="status">{t("subscriptionChecks.loading")}</p> : null}
  {data?.subscriptionCheckHistory.map(item=>{const run=readFragment(HomeSubscriptionCheckRunFieldsFragmentDoc,item);return <RunRow key={run.id} run={run} onOpen={onOpen}/>;})}
  {data?.subscriptionCheckHistory.length===0 ? <p>{t("subscriptionChecks.noHistory")}</p> : null}
 </div>;
}

function formatNextRun(value: string, language: string): string {
  const seconds = Math.max(0, (Date.parse(value) - Date.now()) / 1000);
  if (!Number.isFinite(seconds)) return formatDateTime(value);
  const [amount, unit] = seconds >= 86400 ? [seconds / 86400, "day"] as const : seconds >= 3600 ? [seconds / 3600, "hour"] as const : [seconds / 60, "minute"] as const;
  return new Intl.RelativeTimeFormat(language.startsWith("zh") ? "zh-CN" : "en", { numeric: "auto" }).format(Math.ceil(amount), unit);
}
