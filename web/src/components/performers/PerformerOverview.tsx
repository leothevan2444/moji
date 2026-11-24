import { useAppTimezone } from "../../hooks/useAppTimezone";
import { ActivityLevelIcons } from "./ActivityLevelIcons";
import { activityDisplayStatus, activityRankForDisplay } from "./performerActivityStatus";
import { faRotateRight } from "@fortawesome/free-solid-svg-icons/faRotateRight";
import { faCircleInfo } from "@fortawesome/free-solid-svg-icons/faCircleInfo";
import { faBookmark } from "@fortawesome/free-solid-svg-icons/faBookmark";
import { faClipboardQuestion } from "@fortawesome/free-solid-svg-icons/faClipboardQuestion";
import { faHeart } from "@fortawesome/free-solid-svg-icons/faHeart";
import { faPlayCircle } from "@fortawesome/free-solid-svg-icons/faPlayCircle";
import { faBinoculars } from "@fortawesome/free-solid-svg-icons/faBinoculars";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { Drawer } from "../layout/Drawer";

import {
  formatDateTime,
  formatRelative,
  formatRelativeDate,
  performerImageURL,
  performerInitials,
  stashPerformerURL
} from "../../utils";

import { performerCheckRun } from "../../hooks/useSubscriptionChecks";
import type { PerformerSummary, PerformersPageProps, StashPerformerDetail, SubscribedPerformerEntry } from "./types";

export function PerformerOverview({ performerDetail, performerSummary, performerSubscription, page }: { performerDetail: NonNullable<StashPerformerDetail>; performerSummary?: PerformerSummary | null; performerSubscription: SubscribedPerformerEntry | null; page: Pick<PerformersPageProps, "checkActivity" | "checksLoading" | "checksError" | "checkStarting" | "stashBaseURL" | "reviewCounts" | "pendingSubscriptionID" | "onToggle" | "onRefreshOne" | "fetchingPerformerSummary" | "performerActivity" | "performerActivityError" | "onRefreshActivity" | "onOpenActivityHelp">; }) {
  useAppTimezone();
  const { t } = useTranslation();
  const checkRun = performerCheckRun(page.checkActivity, performerDetail.performer.id);
  const checkItem = checkRun;
  const activity = page.performerActivity;
  const activityError = page.performerActivityError;
  const reloadActivity = page.onRefreshActivity;
  const ratingStatus = activityDisplayStatus(activity, Boolean(activityError), checkItem?.status);
  const displayedRank = activityRankForDisplay(activity);
  const unknownReason = t(`performerActivity.details.${ratingStatus}`);
  const rankDescription = displayedRank ? t("performerActivity.rankLabel", {rank:displayedRank}) : unknownReason;
  const valueDescription = [displayedRank ? t(activity?.lowerBound ? "performerActivity.lowerBound" : "performerActivity.valueHelp") : unknownReason, activityError ? t("performerActivity.loadFailed") : null].filter(Boolean).join(" ");
  const [showCheckError, setShowCheckError] = useState(false);
  const { stashBaseURL, reviewCounts = {}, pendingSubscriptionID, onToggle, onRefreshOne } = page;
  const detailStashURL = stashPerformerURL(performerDetail.performer.id, stashBaseURL);
  const latestRelease = performerSummary?.latestRelease ?? null;
  return (
    <article className="performer-detail-card">
      <div className="performer-detail-hero">
        <div className="avatar avatar--frame performer-detail-hero__avatar"><span className="avatar__fallback" aria-hidden="true">{performerInitials(performerDetail.performer.name)}</span>{performerImageURL(performerDetail.performer.imagePath, stashBaseURL) ? <img data-nsfw className="avatar__image" src={performerImageURL(performerDetail.performer.imagePath, stashBaseURL)!} alt="" onError={event => { event.currentTarget.style.display = "none"; }} /> : null}</div>

        <div className="performer-detail-hero__copy">
          <div className="performer-detail-hero__topline">
            <div className="performer-detail-hero__identity">
              <strong className="performer-detail-hero__name">{performerDetail.performer.name}</strong>
              <span
                className={`performer-detail-hero__state ${performerDetail.performer.favorite ? "is-favorite" : ""}`}
                title={t(performerDetail.performer.favorite ? "performerUi.favorite" : "performerUi.notFavorite")}
              >
                <FontAwesomeIcon icon={faHeart} />
                {t(performerDetail.performer.favorite ? "performerUi.favorite" : "performerUi.notFavorite")}
              </span>
              <button
                type="button"
                className={`performer-detail-hero__state performer-detail-hero__subscription ${performerDetail.performer.subscribed ? "is-subscribed" : ""}`}
                title={t(performerDetail.performer.subscribed ? "performerUi.removeSubscription" : "performerUi.addSubscription")}
                aria-label={t(performerDetail.performer.subscribed ? "performerUi.unsubscribeName" : "performerUi.subscribeName", { name: performerDetail.performer.name })}
                disabled={pendingSubscriptionID === performerDetail.performer.id}
                onClick={() => onToggle(performerDetail.performer)}
              >
                <FontAwesomeIcon icon={faBookmark} />
                Moji {t(performerDetail.performer.subscribed ? "performerUi.subscribed" : "performerUi.unsubscribed")}
              </button>
              <span className="performer-detail-hero__stashbox" title={performerSummary?.matchedStashBox?.name ?? t("performerUi.noPreferredBox")}>
                <span>{t("performerUi.preferredBox")}</span>
                <strong>{performerSummary?.matchedStashBox?.name ?? (performerSummary ? t("performerUi.unmatched") : page.fetchingPerformerSummary ? "…" : "—")}</strong>
              </span>
            </div>

            <div className="performer-detail-card__icons">
              <Link
                className="profile-action-icon profile-action-icon--count"
                title={t("releaseReview.actorEntry", { count: reviewCounts[performerDetail.performer.id] ?? performerSubscription?.reviewReleaseCount ?? 0 })}
                aria-label={t("releaseReview.actorEntry", { count: reviewCounts[performerDetail.performer.id] ?? performerSubscription?.reviewReleaseCount ?? 0 })}
                to={`/performers/review?performer=${encodeURIComponent(performerDetail.performer.id)}`}
              >
                <FontAwesomeIcon icon={faClipboardQuestion} />
                <span aria-hidden="true">{reviewCounts[performerDetail.performer.id] ?? performerSubscription?.reviewReleaseCount ?? 0}</span>
              </Link>
              <button
                type="button"
                className="profile-action-icon"
                aria-busy={Boolean(checkRun || page.checkStarting)}
                title={t(checkRun ? "subscriptionChecks.inRun" : page.checkActivity?.allRunId ? "subscriptionChecks.allBusy" : "performerUi.checkUpdatesHelp")}
                aria-label={t("performerUi.checkName", { name: performerDetail.performer.name })}
                disabled={pendingSubscriptionID === performerDetail.performer.id || Boolean(checkRun || page.checkActivity?.allRunId || page.checkStarting || page.checksLoading || page.checksError)}
                onClick={() => onRefreshOne(performerDetail.performer)}
              >
                <FontAwesomeIcon icon={faBinoculars} />
              </button>
              {detailStashURL ? (
                <a
                  className="profile-action-icon"
                  href={detailStashURL}
                  target="_blank"
                  rel="noreferrer"
                  title={t("performerUi.stashHome")}
                  aria-label={t("performerUi.stashHomeName", { name: performerDetail.performer.name })}
                >
                  <FontAwesomeIcon icon={faPlayCircle} />
                </a>
              ) : null}
            </div>
          </div>

          <div className="performer-detail-hero__description">
            <p title={performerDetail.performer.aliasList.join(" / ")}>
              <span>{t("performerUi.aliases")}</span>
              {performerDetail.performer.aliasList.length > 0 ? performerDetail.performer.aliasList.join(" / ") : t("performerUi.none")}
            </p>
            <p title={performerDetail.disambiguation ?? ""}>
              <span>{t("performerUi.description")}</span>
              {performerDetail.disambiguation || t("performerUi.noDescription")}
            </p>
            <p>
              <span>{t("performerUi.profile")}</span>
              {[performerDetail.birthdate, performerDetail.country, performerDetail.heightCm ? `${performerDetail.heightCm} cm` : null]
                .filter(Boolean)
                .join(" · ") || t("performerUi.none")}
            </p>
            {latestRelease ? (
              <p title={[latestRelease.title, t("performerActivity.latestSourceHelp")].join(" · ")}>
                <span>{t("performerUi.latest")}</span>
                {latestRelease.code || latestRelease.title} · {formatRelativeDate(latestRelease.date)}
              </p>
            ) : null}
          </div>

          <div className="performer-summary-groups">
            <section className="performer-summary-group performer-summary-group--activity" aria-label={t("performerActivity.rank")}>
              <h4>{t("performerActivity.rank")}</h4>
              <button type="button" className="performer-activity-help" title={t("performerActivity.help")} onClick={page.onOpenActivityHelp}>{t("performerActivity.help")}</button>
              <dl className="performer-summary-metrics performer-summary-metrics--activity">
                <div>
                  <dt>{t("performerActivity.rank")}</dt>
                  <dd title={rankDescription}>
                    <ActivityLevelIcons rank={displayedRank} label={displayedRank ? rankDescription : t(`performerActivity.states.${ratingStatus}`)} title={rankDescription} />
                  </dd>
                </div>
                <div>
                  <dt>{t("performerActivity.value")}</dt>
                  <dd className="performer-activity-value" title={valueDescription}>
                    <span>{activity?.value != null ? `${activity.lowerBound ? "≥ " : ""}${activity.value.toFixed(2)}` : t(`performerActivity.states.${ratingStatus}`)}</span>
                    <span className="performer-activity-retry">{activityError ? <button type="button" className="performer-check-reason" title={`${t("performerActivity.loadFailed")} ${t("common.retry")}`} aria-label={`${t("performerActivity.loadFailed")} ${t("common.retry")}`} onClick={reloadActivity}><FontAwesomeIcon icon={faRotateRight} /></button> : null}</span>
                  </dd>
                </div>
              </dl>
            </section>
            <section className="performer-summary-group" aria-label={t("performerWorkspace.checks")}>
              <h4>{t("performerWorkspace.checks")}</h4>
              <dl className="performer-summary-metrics performer-summary-metrics--check">
                <div>
                  <dt>{t("performerActivity.nextCheck")}</dt>
                  <dd title={performerDetail.performer.subscribed && activity?.nextCheckAt ? formatDateTime(activity.nextCheckAt) : undefined}>{checkRun ? t("subscriptionChecks.itemStates.RUNNING") : performerDetail.performer.subscribed && activity?.nextCheckAt ? formatDateTime(activity.nextCheckAt, "compact") : "—"}</dd>
                </div>
                <div>
                  <dt>{t("performerWorkspace.checkTime")}</dt>
                  <dd title={formatDateTime(performerSubscription?.lastCheckedAt)}>{formatRelative(performerSubscription?.lastCheckedAt) ?? t("performerUi.neverChecked")}</dd>
                </div>
                <div className={performerSubscription?.lastError ? "tone-danger" : undefined}>
                  <dt>{t("performerUi.checkStatus")}</dt>
                  <dd className="performer-check-result">
                    <span className={`performer-check-state ${performerSubscription?.lastError ? "is-failed" : performerDetail.performer.subscribed && performerSubscription?.lastCheckedAt ? "is-healthy" : "is-neutral"}`}>{checkRun ? t(`subscriptionChecks.itemStates.${checkItem?.status ?? "PENDING"}`) : t(performerSubscription?.lastError ? "performerUi.failed" : performerDetail.performer.subscribed ? performerSubscription?.lastCheckedAt ? "performerUi.normal" : "performerUi.neverChecked" : "performerUi.unsubscribed")}</span>
                    {performerSubscription?.lastError ? <button type="button" className="performer-check-reason" aria-label={t("performerWorkspace.errorReason")} aria-haspopup="dialog" title={t("performerWorkspace.errorDetails")} onClick={() => setShowCheckError(true)}><span className="performer-check-reason__label">{t("performerWorkspace.errorReason")}</span><FontAwesomeIcon className="performer-check-reason__icon" icon={faCircleInfo} aria-hidden="true" /></button> : null}
                  </dd>
                </div>
              </dl>
            </section>
            <section className="performer-summary-group performer-summary-group--works" aria-label={t("performerWorkspace.worksAndTasks")}>
              <h4>{t("performerWorkspace.worksAndTasks")}</h4>
              <dl className="performer-summary-metrics performer-summary-metrics--works">
                <div>
                  <dt>{t("performerWorkspace.libraryWorks")}</dt>
                  <dd>{performerSummary?.stashSceneCount ?? performerDetail.performer.sceneCount}</dd>
                </div>
                <div>
                  <dt title={performerSummary?.matchedStashBox?.name ?? ""}>{t("performerWorkspace.sourceWorks")}</dt>
                  <dd>{performerSummary?.stashBoxSceneCount ?? "—"}</dd>
                </div>
                <div>
                  <dt title={t("performerWorkspace.activeTasksHelp")}>{t("performerWorkspace.activeTasks")}</dt>
                  <dd title={performerSummary ? t(performerSummary.sceneStatsComplete ? "performerWorkspace.activeTasksHelp" : "performerWorkspace.partialTasksHelp") : undefined}>{performerSummary ? performerSummary.sceneStatsComplete ? performerSummary.activeTaskCount : `≥ ${performerSummary.activeTaskCount}` : "—"}</dd>
                </div>
              </dl>
            </section>
          </div>
        </div>
      </div>
      {showCheckError && performerSubscription?.lastError ? createPortal(
        <Drawer visibleDrawer="confirm" closing={false} title={t("performerUi.failed")} onClose={() => setShowCheckError(false)}>
          <h3>{performerDetail.performer.name}</h3>
          <p className="performer-card-error-message">{performerSubscription.lastError}</p>
        </Drawer>, document.body,
      ) : null}
    </article>
  );
}
