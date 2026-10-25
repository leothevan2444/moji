import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { formatDateTime } from "../../utils";
import type { SubscriptionCheckFieldsFragment } from "../../graphql/generated/graphql";

export function SubscriptionCheckDrawer({ run }: { run: SubscriptionCheckFieldsFragment }) {
  const { t, i18n } = useTranslation();
  const reason = (value: string) => i18n.exists(`subscriptionChecks.reasons.${value}`) ? t(`subscriptionChecks.reasons.${value}`) : value;
  return <div className="home-check-detail">
    <p><strong>{t(`subscriptionChecks.triggers.${run.trigger}`)} · {t(`subscriptionChecks.scopes.${run.scope}`)}</strong> · {t(`subscriptionChecks.states.${run.status}`)}</p>
    <dl className="home-check-schedule">
      <div><dt>{t("subscriptionChecks.startedAt")}</dt><dd>{formatDateTime(run.startedAt)}</dd></div>
      {run.finishedAt ? <div><dt>{t("subscriptionChecks.finished")}</dt><dd>{formatDateTime(run.finishedAt)}</dd></div> : null}
    </dl>
    <p role="status">{run.status === "RUNNING" && run.totalCount === 0 ? t("subscriptionChecks.loadingPerformers") : t("subscriptionChecks.progress", { completed: run.completedCount, total: run.totalCount })}</p>
    {run.totalCount > 0 ? <div className="home-progress" role="progressbar" aria-label={t("subscriptionChecks.progressLabel")} aria-valuemin={0} aria-valuemax={run.totalCount} aria-valuenow={run.completedCount}><span style={{ width: `${run.completedCount / run.totalCount * 100}%` }} /></div> : null}
    <p>{t("subscriptionChecks.counts", { succeeded: run.succeededCount, failed: run.failedCount, skipped: run.skippedCount })}</p>
    {run.error ? <p className="tone-danger" role="alert">{reason(run.error)}</p> : null}
    <ul className="home-check-result-list">{run.items.map(item => <li key={item.performerId}>
      <Link to={`/performers/${encodeURIComponent(item.performerId)}`}>{item.name || item.performerId}</Link><span>{t(`subscriptionChecks.itemStates.${item.status}`)}</span>
      {item.error ? <p className={item.status === "FAILED" ? "tone-danger" : "home-muted"}>{reason(item.error)}</p> : null}
    </li>)}</ul>
    <p className="home-muted home-check-id">{t("subscriptionChecks.runId")} <code>{run.id}</code></p>
  </div>;
}
