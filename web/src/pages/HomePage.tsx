import { useEffect, useRef, useState, type FormEvent } from "react";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowRight, faCheck, faCircleExclamation, faMagnifyingGlass, faRotate, faBookmark, faArrowUpRightFromSquare, faBolt } from "@fortawesome/free-solid-svg-icons";
import { HomeSubscriptionChecks, type HomeChecksView } from "../components/home/HomeSubscriptionChecks";
import { IngestCard, ServiceCard, buildJackettConfig, buildQBittorrentConfig } from "../components/home";
import { TaskPrimaryAction } from "../components/tasks/TaskPrimaryAction";
import { formatDateTime, formatRelative, taskFailureSummary, taskPresentation, taskSummary, type DashboardTask } from "../utils";
import type { GraphQLConnectionStatus } from "../graphql/client";
import type { SettingsTab } from "../types";
import type { HomePageDocumentQuery, HomeServiceStatusQuery, HomeSubscriptionSummaryQuery } from "../graphql/generated/graphql";
import { useGlobalSlashShortcut } from "../hooks/useGlobalSlashShortcut";

export interface HomePageProps {
  checks?: HomeChecksView | null;
  checksError?: string | null;
  onRefreshChecks?: () => void;
  onOpenCheck?: (id: string) => void;
  interactionOpen?: boolean;
  listUpdatePending?: boolean;
  onApplyUpdates?: () => void;
  overview: (Omit<HomePageDocumentQuery["homeOverview"], "attentionTasks" | "activeTasks" | "recentTasks"> & { attentionTasks: DashboardTask[]; activeTasks: DashboardTask[]; recentTasks: DashboardTask[] }) | null;
  runtimeSettings: HomePageDocumentQuery["settings"] | null;
  runtimeStatus: HomeServiceStatusQuery["settingsStatus"] | null;
  subscription: HomeSubscriptionSummaryQuery["homeSubscriptionSummary"] | null;
  overviewError: string | null;
  statusError: string | null;
  subscriptionError: string | null;
  fetching: boolean;
  connectionStatus: GraphQLConnectionStatus;
  pendingTaskRetryId: string | null;
  onRefresh: () => void;
  onRefreshStatus: () => void;
  onRefreshSubscription: () => void;
  onSearch: (query: string, source: "stashbox" | "jackett") => void;
  onOpenTask: (id: string) => void;
  onConfirmFiles: (id: string) => void;
  onRetryTask: (id: string) => void;
  onResolveTask: (id: string) => void;
  onOpenQueue: () => void;
  onOpenSettings: (tab: SettingsTab) => void;
}

export function HomePage(props: HomePageProps) {
  const { t } = useTranslation();
  const { overview, runtimeSettings: settings, runtimeStatus: status, subscription } = props;
  const [query, setQuery] = useState(() => readSession("moji:home-search"));
  const [source, setSource] = useState<"stashbox" | "jackett">(() => readSession("moji:home-source") === "jackett" ? "jackett" : "stashbox");
  const input = useRef<HTMLInputElement>(null);
  useGlobalSlashShortcut(input, !props.interactionOpen);
  const metrics = [
    { key: "attention", count: overview?.attention, to: "/tasks?status=failed", tone: "attention" },
    { key: "active", count: overview?.active, to: "/tasks?status=active", tone: "active" },
    { key: "waiting", count: overview?.waiting, to: "/tasks?status=waiting", tone: "waiting" },
    { key: "completed", count: overview?.completedLast24Hours, to: "/tasks?status=completed&completedWithin=24h", tone: "completed" }
  ];
  const problems = status ? (["stash", "jackett", "qbittorrent"] as const).filter(service => !status[service].ready) : [];
  const unconfigured = status ? problems.filter(service => !status[service].configured) : [];
  const needsSetup = Boolean(status && (!status.ingest?.configured || unconfigured.length));
  const primaryProblem = problems[0];
  const serviceNames = { stash: "Stash", jackett: "Jackett", qbittorrent: "qBittorrent" };
  const [connectionInterrupted, setConnectionInterrupted] = useState(() => ["disconnected", "error"].includes(props.connectionStatus));
  useEffect(() => {
    if (props.connectionStatus === "connected") setConnectionInterrupted(false);
    else if (["disconnected", "error"].includes(props.connectionStatus)) setConnectionInterrupted(true);
  }, [props.connectionStatus]);
  const submit = (event: FormEvent) => { event.preventDefault(); if (query.trim()) props.onSearch(query.trim(), source); };
  const error = (message: string | null, retry: () => void, hasData = false) => message ? <div className="home-module-error" role="status">{hasData ? <strong>{t("workbench.refreshFailed")}</strong> : null}<p>{message}</p><button type="button" className="ghost-button" onClick={retry}>{t("common.retry")}</button></div> : null;
  const loading = <div className="home-loading" role="status" aria-label={t("common.loading")}><div className="skeleton" /><div className="skeleton" /></div>;

  return <div className="home-workspace">
    <section className="home-intro section-band">
      {connectionInterrupted ? <aside className="home-alert home-connection-alert" role="status">
        <FontAwesomeIcon icon={faCircleExclamation} aria-hidden="true" />
        <div><strong>{t("workbench.offline")}</strong></div>
        <button type="button" className="ghost-button" disabled={props.fetching} onClick={props.onRefresh}>{t("common.retry")}</button>
      </aside> : null}
      <form className="home-search" onSubmit={submit} role="search" aria-label={t("workbench.searchLabel")}>
        <label className="home-search-mode"><span className="sr-only">{t("discoverUi.searchMode")}</span><select value={source} onChange={event => { setSource(event.target.value as typeof source); writeSession("moji:home-source", event.target.value); }}><option value="stashbox">{t("workbench.findMovie")}</option><option value="jackett">{t("workbench.findResource")}</option></select></label>
        <FontAwesomeIcon icon={faMagnifyingGlass} aria-hidden="true" />
        <input ref={input} value={query} onChange={event => { setQuery(event.target.value); writeSession("moji:home-search", event.target.value); }} aria-label={t("workbench.searchLabel")} placeholder={t("workbench.searchPlaceholder")} />
        <button type="submit" disabled={!query.trim()}>{t("workbench.search")}<FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></button>
      </form>
      <div className="home-metrics" aria-label={t("workbench.overview")}>
        {metrics.map(metric => <Link key={metric.key} to={metric.to} className={`home-metric home-metric--${metric.tone}`}><span>{t(`workbench.metrics.${metric.key}`)}</span><strong>{metric.count ?? "—"}</strong><FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></Link>)}
      </div>
    </section>

    {needsSetup ? <section className="home-alert home-alert--setup" aria-label={t("workbench.setupTitle")}><FontAwesomeIcon icon={faBolt} aria-hidden="true" /><div><strong>{t("workbench.setupTitle")}</strong><p>{t("workbench.setupDetail")}</p><div className="home-setup-steps">{(["stash", "jackett", "qbittorrent", "ingest"] as const).map(service => <button type="button" key={service} onClick={() => props.onOpenSettings(service === "ingest" ? "ingest" : "connections")}><FontAwesomeIcon icon={status?.[service]?.configured ? faCheck : faCircleExclamation} aria-hidden="true" />{service === "ingest" ? t("workbench.delivery") : serviceNames[service]} · {t(status?.[service]?.configured ? "workbench.configured" : "workbench.configure")}</button>)}</div></div></section>
      : primaryProblem ? <section className="home-alert" role="status"><FontAwesomeIcon icon={faCircleExclamation} aria-hidden="true" /><div><strong>{t(`workbench.impacts.${primaryProblem}`)}</strong>{problems.length > 1 ? <p>{t("workbench.moreProblems", { count: problems.length })}</p> : null}</div><button type="button" className="ghost-button" onClick={() => props.onOpenSettings("connections")}>{t("workbench.checkConnection")}</button></section> : null}

    {props.listUpdatePending ? <button type="button" className="ghost-button home-update-notice" onClick={props.onApplyUpdates}>{t("workbench.updates")} <FontAwesomeIcon icon={faRotate} aria-hidden="true" /></button> : null}
    <div className="home-grid">
      <div className="home-main">
      <section className="home-panel home-attention" aria-labelledby="home-attention-title">
        <div className="home-panel-heading"><h2 id="home-attention-title">{t("workbench.attention")} {overview && overview.attention > 0 ? <span className="home-count">{overview.attention}</span> : null}</h2>{overview && overview.attention > 0 ? <button type="button" className="ghost-button" onClick={props.onOpenQueue}>{t("workbench.handleQueue")}<FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></button> : <FontAwesomeIcon icon={faCheck} className="home-success" aria-hidden="true" />}</div>
        {error(props.overviewError, props.onRefresh, Boolean(overview))}
        {!overview ? !props.overviewError && loading : overview.attentionTasks.length ? <><div className="home-task-list">{overview.attentionTasks.map(task => <HomeTaskRow key={task.id} task={task} attention {...props} />)}</div><Link className="home-more" to="/tasks?status=failed">{t("workbench.allAttention")}<FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></Link></> : <p className="home-empty home-empty--compact">{t("workbench.noAttention")}</p>}
      </section>

      <section className="home-panel home-active" aria-labelledby="home-active-title">
        <div className="home-panel-heading"><h2 id="home-active-title">{t("workbench.active")}</h2><Link to="/tasks?status=active">{t("workbench.allTasks")} <FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></Link></div>
        {!overview ? !props.overviewError && loading : overview.activeTasks.length ? <div className="home-task-list">{overview.activeTasks.map(task => <HomeTaskRow key={task.id} task={task} {...props} />)}</div> : <div className="home-empty"><FontAwesomeIcon icon={faMagnifyingGlass} aria-hidden="true" /><strong>{t(overview.waiting ? "workbench.waitingTitle" : "workbench.emptyTitle")}</strong><p>{t(overview.waiting ? "workbench.waitingDetail" : "workbench.emptyDetail")}</p><button type="button" className="ghost-button" onClick={() => input.current?.focus()}>{t("workbench.findMovie")} <FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></button></div>}
      </section>

      <section className="home-panel home-recent" aria-labelledby="home-recent-title">
        <div className="home-panel-heading"><h2 id="home-recent-title">{t("workbench.recent")}</h2><Link to="/tasks?status=completed">{t("workbench.allCompleted")} <FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></Link></div>
        {!overview ? !props.overviewError && loading : overview.recentTasks.length ? <div className="home-task-list">{overview.recentTasks.map(task => {
          const base = settings?.stash.url;
          const sceneId = task.stashSceneIds[0];
          const url = base && /^https?:\/\//i.test(base) && sceneId ? `${base.replace(/\/$/, "")}/scenes/${encodeURIComponent(sceneId)}` : null;
          return <div key={task.id} className="home-task-row"><span className="home-complete-icon"><FontAwesomeIcon icon={faCheck} aria-hidden="true" /></span><div className="home-task-copy"><button type="button" className="home-task-title" data-task-id={task.id} onClick={() => props.onOpenTask(task.id)}>{taskSummary(task)}</button><time className="home-muted" dateTime={task.ingestCompletedAt ?? undefined} title={formatDateTime(task.ingestCompletedAt)}>{t("workbench.completedAt", { time: formatRelative(task.ingestCompletedAt) })}</time></div>{url ? <a className="home-stash-link" href={url} target="_blank" rel="noopener noreferrer">{t("workbench.openStash")}<FontAwesomeIcon icon={faArrowUpRightFromSquare} aria-hidden="true" /></a> : null}</div>;
        })}</div> : <p className="home-empty home-empty--compact">{t("workbench.noRecent")}</p>}
      </section>

      </div>
      <aside className="home-sidebar">
      <section className="home-panel home-subscriptions" aria-labelledby="home-subscriptions-title">
        <div className="home-panel-heading"><h2 id="home-subscriptions-title">{t("workbench.subscriptions")}</h2><FontAwesomeIcon icon={faBookmark} aria-hidden="true" /></div>
        {error(props.subscriptionError, props.onRefreshSubscription, Boolean(subscription))}
        {!subscription ? !props.subscriptionError && loading : <><p className="home-big-number">{subscription.total}<span>{t("workbench.subscribedUnit")}</span></p><p className="home-muted">{t(subscription.total ? "workbench.subscriptionHelp" : "workbench.subscriptionEmpty")}</p>
          {subscription.attention > 0 ? <p className="home-subscription-warning"><FontAwesomeIcon icon={faCircleExclamation} aria-hidden="true" />{t("workbench.subscriptionIssues", { count: subscription.attention })}</p> : null}
          {subscription.unchecked > 0 ? <p className="home-muted">{t("workbench.unchecked", { count: subscription.unchecked })}</p> : null}
          {subscription.lastCheckedAt ? <p className="home-muted"><time dateTime={subscription.lastCheckedAt} title={formatDateTime(subscription.lastCheckedAt)}>{t("workbench.lastChecked", { time: formatRelative(subscription.lastCheckedAt) })}</time></p> : null}
          {subscription.attentionPerformerIds.map((id, index) => <Link className="home-subscription-issue" key={id} to={`/performers/${encodeURIComponent(id)}`}>{t("workbench.reviewSubscription", { number: index + 1 })}<FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></Link>)}
          <Link className="home-more" to="/performers">{t("workbench.manageSubscriptions")} <FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></Link></>}
      </section>

      <HomeSubscriptionChecks snapshot={props.checks ?? null} error={props.checksError ?? null} onRetry={props.onRefreshChecks ?? props.onRefresh} onOpenRun={id => props.onOpenCheck?.(id)} />
      <section className="home-panel home-system" aria-labelledby="home-system-title">
        <div className="home-panel-heading"><h2 id="home-system-title">{t("workbench.system")}</h2><span className={`home-dot ${!status ? "home-dot--unknown" : problems.length || !status.ingest?.configured ? "home-dot--warning" : ""}`} /></div>
        {error(props.statusError, props.onRefreshStatus, Boolean(status))}
        {!status ? !props.statusError && loading : <><ul className="home-service-list">{(["jackett", "qbittorrent", "stash"] as const).map(service => <li key={service}><span>{serviceNames[service]}</span><span className={status[service].ready ? "home-success" : "home-warning"}><FontAwesomeIcon icon={status[service].ready ? faCheck : faCircleExclamation} aria-hidden="true" />{t(!status[service].configured ? "workbench.unconfigured" : status[service].ready ? "workbench.ready" : "workbench.unavailable")}</span></li>)}<li><span>{t("workbench.delivery")}</span><button type="button" className="home-inline-button" onClick={() => props.onOpenSettings("ingest")}>{t(status.ingest?.configured ? "workbench.configured" : "workbench.unconfigured")} <FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></button></li></ul>
          {settings ? <div className="home-automation"><FontAwesomeIcon icon={faBolt} aria-hidden="true" /><span>{t(settings?.automation.taskProcessingEnabled === false ? "workbench.automationPaused" : "workbench.automationEnabled")}</span><button type="button" onClick={() => props.onOpenSettings("automation")} aria-label={t("automationUi.openSettings")}><FontAwesomeIcon icon={faArrowRight} /></button></div> : null}
          {settings ? <details className="home-diagnostics"><summary>{t("workbench.systemDetails")}</summary><div className="home-diagnostics-content"><ServiceCard service="stash" title="Stash" status={status.stash} config={[{ key: "URL", value: settings.stash.url }]} stats={status.stashStats} okAt={status.stashStats?.okAt ?? null} lastError={status.stashStats?.lastError ?? null} cta={{ kind: "open-settings", label: t("home.configure"), tab: "connections" }} onOpenSettings={props.onOpenSettings} /><ServiceCard service="jackett" title="Jackett" status={status.jackett} config={buildJackettConfig(settings)} stats={status.jackettStats ? { ...status.jackettStats, hasIndexerSearch: true } : null} okAt={status.jackettStats?.okAt ?? null} lastError={status.jackettStats?.lastError ?? null} cta={{ kind: "open-settings", label: t("home.configure"), tab: "connections" }} onOpenSettings={props.onOpenSettings} /><ServiceCard service="qbittorrent" title="qBittorrent" status={status.qbittorrent} config={buildQBittorrentConfig(settings)} stats={status.qbittorrentStats ? { ...status.qbittorrentStats, hasIndexerSearch: false } : null} okAt={status.qbittorrentStats?.okAt ?? null} lastError={status.qbittorrentStats?.lastError ?? null} cta={{ kind: "open-settings", label: t("home.configure"), tab: "connections" }} onOpenSettings={props.onOpenSettings} /><IngestCard ingest={settings.ingest} ingestStatus={status.ingest} onOpenSettings={props.onOpenSettings} /></div></details> : null}</>}
      </section>
      </aside>
    </div>
  </div>;
}

function HomeTaskRow({ task, attention = false, onOpenTask, onConfirmFiles, onRetryTask, onResolveTask, pendingTaskRetryId }: Pick<HomePageProps, "onOpenTask" | "onConfirmFiles" | "onRetryTask" | "onResolveTask" | "pendingTaskRetryId"> & { task: DashboardTask; attention?: boolean }) {
  const { t, i18n } = useTranslation();
  const issueKey = `workbench.issues.${task.stageErrorCode ?? ""}`;
  const failure = i18n.exists(issueKey) ? t(issueKey) : taskFailureSummary(task).title;
  const presentation = taskPresentation(task);
  return <div className={`home-task-row ${attention ? "home-task-row--attention" : ""}`}>
    <span className={`home-task-marker ${attention ? "home-warning" : ""}`}><FontAwesomeIcon icon={attention ? faCircleExclamation : faBolt} aria-hidden="true" /></span>
    <div className="home-task-copy"><button type="button" data-task-id={task.id} className="home-task-title" onClick={() => onOpenTask(task.id)}>{taskSummary(task)}</button><span className="home-muted">{attention ? failure : presentation.label}</span>{!attention && presentation.showDownloadProgress ? <div className="home-progress" role="progressbar" aria-label={t("taskModel.states.downloading")} aria-valuemin={0} aria-valuemax={100} aria-valuenow={presentation.progressPercent}><span style={{ width: `${presentation.progressPercent}%` }} /></div> : null}</div>
    {attention ? <TaskPrimaryAction task={task} pendingRetry={pendingTaskRetryId === task.id} onResolve={onResolveTask} onFiles={onConfirmFiles} onRetry={onRetryTask} /> : presentation.showDownloadProgress ? <span className="home-progress-value">{presentation.progressPercent}%</span> : <button className="home-inline-button" type="button" aria-label={t("taskUi.card.aria", { task: taskSummary(task), status: presentation.label })} onClick={() => onOpenTask(task.id)}><FontAwesomeIcon icon={faArrowRight} /></button>}
  </div>;
}

function readSession(key: string) {
  try { return sessionStorage.getItem(key) ?? ""; } catch { return ""; }
}
function writeSession(key: string, value: string) {
  try { sessionStorage.setItem(key, value); } catch { /* Search remains usable when storage is unavailable. */ }
}
