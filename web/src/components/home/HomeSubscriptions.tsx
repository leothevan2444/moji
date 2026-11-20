import { useAppTimezone } from "../../hooks/useAppTimezone";
import { createPortal } from "react-dom";
import { useState } from "react";
import { Link } from "react-router";
import { useQuery } from "urql";
import { Drawer } from "../layout/Drawer";
import { SubscriptionCheckHistoryDocument, HomeSubscriptionCheckRunFieldsFragmentDoc } from "../../graphql/generated/graphql";
import { useFragment as readFragment } from "../../graphql/generated/fragment-masking";
import { describeQueryError } from "../../services/queryError";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowRight, faBinoculars, faCheck, faCircleExclamation, faClockRotateLeft } from "@fortawesome/free-solid-svg-icons";
import { useTranslation } from "react-i18next";
import { formatDateTime, formatRelative } from "../../utils";
import type { HomeSubscriptionCheckRunFieldsFragment, HomeSubscriptionChecksQuery, HomeSubscriptionSummaryQuery } from "../../graphql/generated/graphql";

export type HomeCheckRunView = HomeSubscriptionCheckRunFieldsFragment;
export type HomeChecksView = Omit<HomeSubscriptionChecksQuery["homeSubscriptionChecks"], "activeRuns" | "recentRuns" | "latestScheduled" | "latestFinishedScheduled"> & {
  activeRuns: HomeCheckRunView[];
  recentRuns: HomeCheckRunView[];
  latestScheduled: HomeCheckRunView | null;
  latestFinishedScheduled: HomeCheckRunView | null;
};

function RunRow({ run, onOpen, compact = false }: { run: HomeCheckRunView; onOpen: (id: string) => void; compact?: boolean }) {
  useAppTimezone();
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

export function HomeSubscriptions({ subscription, subscriptionError, onRetrySubscription, snapshot, error, onRetry, onOpenRun }: {
  subscription: HomeSubscriptionSummaryQuery["homeSubscriptionSummary"] | null;
  subscriptionError: string | null;
  onRetrySubscription: () => void;
  snapshot: HomeChecksView | null;
  error: string | null;
  onRetry: () => void;
  onOpenRun: (id: string) => void;
}) {
  useAppTimezone();
  const { t, i18n } = useTranslation();
  const [historyOpen,setHistoryOpen]=useState(false);
  return <section className="home-panel home-subscriptions" id="subscription-checks" aria-labelledby="home-subscriptions-title">
    <div className="home-panel-heading">
      <h2 id="home-subscriptions-title">{t("workbench.subscriptions")}</h2>
      <Link to="/performers" aria-label={t("workbench.manageSubscriptions")}>{t("workbench.manageShort")}</Link>
    </div>
    {subscriptionError ? <div className="home-module-error" role="alert" aria-label={t("workbench.subscriptions")}><p>{subscription ? t("workbench.refreshFailed") : subscriptionError}</p><button type="button" onClick={onRetrySubscription}>{t("common.retry")}</button></div> : null}
    {!subscription ? !subscriptionError && <div className="home-loading" aria-label={t("common.loading")}><span /><span /><span /></div> : <>
      <div className="home-subscription-metrics">
        <div className="home-subscription-total">
          <p className="home-big-number">{subscription.total}</p>
          <span className="home-muted">{t("workbench.subscribedActors")}</span>
        </div>
        <Link className="home-subscription-issues" to="/performers?filter=error" title={t("workbench.subscriptionIssuesHelp")} aria-label={t("workbench.viewSubscriptionIssues", { count: subscription.attention })}>
          <span className={`home-subscription-issue-count${subscription.attention > 0 ? " home-warning" : " home-muted"}`}>{subscription.attention}<FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></span>
          <span className="home-muted">{t("workbench.latestCheckIssues")}</span>
        </Link>
      </div>
      {subscription.unchecked > 0 ? <p className="home-muted">{t("workbench.unchecked", { count: subscription.unchecked })}</p> : null}
      {subscription.total === 0 ? <p className="home-muted">{t("workbench.subscriptionEmpty")}</p> : null}
    </>}
    <div className="home-check-summary">
      <div className="home-check-status-line">
        <div className="home-check-status">
          {error ? <div className="home-module-error" role="alert" aria-label={t("subscriptionChecks.title")}><p>{snapshot ? t("workbench.refreshFailed") : error}</p><button type="button" onClick={onRetry}>{t("common.retry")}</button></div> : null}
          {!snapshot ? !error && <div className="home-loading" aria-label={t("subscriptionChecks.loading")}><span /><span /><span /></div> : snapshot.activeRuns.length > 0 ? <div className="home-check-list" aria-label={t("subscriptionChecks.current")}><RunRow run={snapshot.activeRuns[0]} compact onOpen={onOpenRun} /></div> : <>
            <span className="home-muted"><FontAwesomeIcon icon={faBinoculars} aria-hidden="true" /> {t(snapshot.enabled ? "subscriptionChecks.automaticEnabled" : "subscriptionChecks.automaticUnavailable")}</span>
            {snapshot.activeCount > 0 ? <span className="home-muted">{t("subscriptionChecks.runningCount", { count: snapshot.activeCount })}</span> : snapshot.enabled ? <>
              <span className="home-muted" aria-hidden="true">·</span>
              <span className="home-muted">{snapshot.nextRunAt ? <time dateTime={snapshot.nextRunAt} title={formatDateTime(snapshot.nextRunAt)} aria-label={t("subscriptionChecks.nextCheckIn", { time: formatNextRun(snapshot.nextRunAt, i18n.language) })}>{formatNextRun(snapshot.nextRunAt, i18n.language)}</time> : t("subscriptionChecks.nextCheckUnknown")}</span>
            </> : null}
          </>}
        </div>
        <button type="button" className="home-inline-button home-check-history-button" aria-haspopup="dialog" onClick={() => setHistoryOpen(true)}><FontAwesomeIcon icon={faClockRotateLeft} aria-hidden="true" />{t("performerActivity.history")}</button>
      </div>
    </div>
    {snapshot && snapshot.activeRuns.length > 1 ? <div className="home-check-list" aria-label={t("subscriptionChecks.current")}>{snapshot.activeRuns.slice(1).map(run => <RunRow key={run.id} run={run} compact onOpen={onOpenRun} />)}</div> : null}
    {snapshot && snapshot.activeCount > snapshot.activeRuns.length ? <button type="button" className="home-inline-button home-check-more" onClick={() => setHistoryOpen(true)}>{t("subscriptionChecks.moreActive", { count: snapshot.activeCount - snapshot.activeRuns.length })}</button> : null}
    {historyOpen ? createPortal(<Drawer visibleDrawer="subscription-history" closing={false} title={t("performerActivity.history")} onClose={() => setHistoryOpen(false)}><CheckHistory snapshot={snapshot} onOpen={id => {setHistoryOpen(false);onOpenRun(id);}} /></Drawer>, document.body) : null}
  </section>;
}

function CheckHistory({ onOpen, snapshot }: { onOpen: (id: string) => void; snapshot: HomeChecksView | null }) {
  useAppTimezone();
 const {t}=useTranslation();
 const [{data,error,fetching},refresh]=useQuery({query:SubscriptionCheckHistoryDocument,requestPolicy:"network-only"});
 const statistics = snapshot?.statistics;
 const latest = snapshot?.latestFinishedScheduled;
 const latestWarning = latest && ["FAILED", "PARTIAL", "INTERRUPTED"].includes(latest.status);
 return <div className="home-check-list">
  <div className="home-check-latest">
    <span className="home-muted">{t("subscriptionChecks.lastFinishedScheduled")}</span>
    {latest ? <>
      <button type="button" className={`home-inline-button${latestWarning ? " home-warning" : latest.status === "SUCCEEDED" ? " home-success" : ""}`} onClick={() => onOpen(latest.id)}>
        {latestWarning || latest.status === "SUCCEEDED" ? <FontAwesomeIcon icon={latestWarning ? faCircleExclamation : faCheck} aria-hidden="true" /> : null}{t(`subscriptionChecks.states.${latest.status}`)} <FontAwesomeIcon icon={faArrowRight} aria-hidden="true" />
      </button>
      <time dateTime={latest.finishedAt ?? latest.startedAt}>{formatDateTime(latest.finishedAt ?? latest.startedAt)}</time>
    </> : <span className="home-muted">{t(snapshot ? "subscriptionChecks.historyNone" : "subscriptionChecks.historyUnavailable")}</span>}
  </div>
  {statistics ? <div className="home-check-statistics" title={t("subscriptionChecks.statisticsHelp", { count: statistics.runCount })} tabIndex={0} aria-label={t("subscriptionChecks.statisticsHelp", { count: statistics.runCount })}>
        <span className="home-muted">{t("subscriptionChecks.recentStatistics")}</span>
        <dl>
          <div><dt>{t("subscriptionChecks.statSucceeded")}</dt><dd>{statistics.succeededCount}</dd></div>
          <div><dt>{t("subscriptionChecks.statFailed")}</dt><dd>{statistics.failedCount}</dd></div>
          <div><dt>{t("subscriptionChecks.statSkipped")}</dt><dd>{statistics.skippedCount}</dd></div>
        </dl>
      </div> : null}
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
