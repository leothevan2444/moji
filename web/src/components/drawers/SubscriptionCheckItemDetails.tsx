import { useAppTimezone } from "../../hooks/useAppTimezone";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import type { SubscriptionCheckFieldsFragment } from "../../graphql/generated/graphql";
import { formatDateTime } from "../../utils";

type Item = SubscriptionCheckFieldsFragment["items"][number];

export function SubscriptionCheckItemDetails({ item }: { item: Item }) {
  useAppTimezone();
  const { t } = useTranslation();
  const d = item.details;
  const reviews = `/performers/review?performer=${encodeURIComponent(item.performerId)}&status=FAILED`;
  const duration = item.startedAt && item.finishedAt
    ? Math.max(0, Math.round((Date.parse(item.finishedAt) - Date.parse(item.startedAt)) / 1000)) : null;
  if (!d) return item.status === "PENDING" || item.status === "RUNNING" ? null : <p className="home-muted">{t("checkDetails.notRecorded")}</p>;
  const releasesStarted = d.catalogStatus === "SUCCEEDED";
  const retriesStarted = releasesStarted && ["HISTORICAL_RETRIES", "FINISHED"].includes(d.phase);
  return <div className="check-item-details">
    <p className="home-muted">{item.startedAt ? formatDateTime(item.startedAt) : ""}{duration !== null ? ` · ${t("checkDetails.duration", { count: duration })}` : ""}{d.phase !== "FINISHED" ? ` · ${t("checkDetails.phase", { phase: t(`checkDetails.phases.${d.phase}`) })}` : ""}</p>
    <section>
      <h4>{t("checkDetails.catalog")} · {t(`subscriptionChecks.itemStates.${d.catalogStatus}`)}</h4>
      {d.catalogSource ? <p className="check-item-details__source">{d.catalogSource} · {t(`checkDetails.modes.${d.catalogMode}`)}</p> : null}
      <p>{t("checkDetails.catalogCounts", { pages: d.catalogPages, scenes: d.catalogScenes, releases: d.catalogReleases })}</p>
      {d.catalogStopReason ? <p className="home-muted">{t(`checkDetails.stops.${d.catalogStopReason}`, { defaultValue: d.catalogStopReason })}</p> : null}
      {d.catalogErrorCode ? <p className="tone-danger">{t(`checkDetails.errors.${d.catalogErrorCode}`, { defaultValue: d.catalogErrorCode })}</p> : null}
    </section>
    <section>
      <h4>{t("checkDetails.releases")}</h4>
      {releasesStarted ? <><p>{t("checkDetails.releaseCounts", { fresh: d.newCount, known: d.knownCount, queued: d.queuedCount, library: d.inLibraryCount, review: d.reviewCount, skipped: d.skippedCount, upcoming: d.upcomingCount })}</p>
      <p className={d.creationFailedCount ? "tone-danger" : "home-muted"}>{t("checkDetails.creationFailures", { count: d.creationFailedCount })}</p></> : <p>{t("checkDetails.notStarted")}</p>}
    </section>
    <section>
      <h4>{t("checkDetails.retries")}</h4>
      {retriesStarted ? <p>{t("checkDetails.retryCounts", { total: d.retryTotal, attempted: d.retryAttempted, succeeded: d.retrySucceeded, failed: d.retryFailed, deferred: d.retryDeferred })}</p> : <p>{t("checkDetails.notStarted")}</p>}
      {d.retryDeferred ? <p className="home-muted">{t("checkDetails.deferred")}</p> : null}
      {d.retryFailed || d.creationFailedCount ? <Link to={reviews}>{t("checkDetails.openFailures")}</Link> : null}
    </section>
    {d.results.length ? <details open={d.retryFailed > 0 || d.creationFailedCount > 0}>
      <summary>{t("checkDetails.films", { count: d.results.length })}</summary>
      <ul className="check-film-results">{d.results.map((result, index) => <li key={`${result.phase}/${result.key}/${index}`}>
        <strong>{result.code || result.title}</strong>
        {result.title && result.code ? <p className="home-muted">{result.title}</p> : null}
        <p className={result.outcome === "FAILED" ? "tone-danger" : "home-muted"}>{t(`checkDetails.phases.${result.phase}`)} · {t(`checkDetails.outcomes.${result.outcome}`, { defaultValue: result.outcome })}</p>
        {result.reasonCode ? <p className="home-muted">{t(`releaseReview.reasons.${result.reasonCode}`, { defaultValue: result.reasonCode })}</p> : null}
        {result.errorCode ? <p>{t(`checkDetails.errors.${result.errorCode}`, { defaultValue: result.errorCode })}</p> : null}
        {result.error ? <details><summary>{t("checkDetails.technicalError")}</summary><p className="check-item-details__error">{result.error}</p><code>{result.errorCode}</code></details> : null}
        {result.nextRetryAt ? <p>{t("checkDetails.nextRetry", { time: formatDateTime(result.nextRetryAt) })}</p> : null}
        {result.taskId ? <Link to={`/tasks/${encodeURIComponent(result.taskId)}`}>{t("checkDetails.openTask")}</Link> : result.outcome === "FAILED" || result.outcome === "DEFERRED" ? <Link to={`/performers/review?performer=${encodeURIComponent(item.performerId)}&status=ALL&q=${encodeURIComponent(result.code)}`}>{t("checkDetails.openFilm")}</Link> : null}
      </li>)}</ul>
      {d.omittedResults ? <p>{t("checkDetails.omitted", { count: d.omittedResults })}</p> : null}
    </details> : null}
  </div>;
}
