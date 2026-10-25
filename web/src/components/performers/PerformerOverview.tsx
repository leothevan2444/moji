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

import type { PerformerSummary, PerformersPageProps, StashPerformerDetail, SubscribedPerformerEntry } from "./types";

export function PerformerOverview({ performerDetail, performerSummary, performerSubscription, page }: { performerDetail: NonNullable<StashPerformerDetail>; performerSummary?: PerformerSummary | null; performerSubscription: SubscribedPerformerEntry | null; page: Pick<PerformersPageProps, "stashBaseURL" | "reviewCounts" | "pendingSubscriptionID" | "onToggle" | "onRefreshOne" | "fetchingPerformerSummary">; }) {
  const { t } = useTranslation();
  const [showCheckError, setShowCheckError] = useState(false);
  const { stashBaseURL, reviewCounts = {}, pendingSubscriptionID, onToggle, onRefreshOne } = page;
  const detailStashURL = stashPerformerURL(performerDetail.performer.id, stashBaseURL);
  const latestRelease = performerSubscription?.recentReleases[0] ?? null;
  return (
    <article className="performer-detail-card">
      <div className="performer-detail-hero">
        <div className="avatar avatar--frame performer-detail-hero__avatar"><span className="avatar__fallback" aria-hidden="true">{performerInitials(performerDetail.performer.name)}</span>{performerImageURL(performerDetail.performer.imagePath, stashBaseURL) ? <img className="avatar__image" src={performerImageURL(performerDetail.performer.imagePath, stashBaseURL)!} alt="" onError={event => { event.currentTarget.style.display = "none"; }} /> : null}</div>

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
                aria-busy={pendingSubscriptionID === performerDetail.performer.id}
                title={t(performerDetail.performer.subscribed ? "performerUi.checkUpdatesHelp" : "performerUi.checkRequiresSubscription")}
                aria-label={t("performerUi.checkName", { name: performerDetail.performer.name })}
                disabled={!performerDetail.performer.subscribed || pendingSubscriptionID === performerDetail.performer.id}
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
              <p title={latestRelease.title}>
                <span>{t("performerUi.latest")}</span>
                {latestRelease.code || latestRelease.title} · {formatRelativeDate(latestRelease.date || latestRelease.seenAt)}
              </p>
            ) : null}
          </div>

          <div className="performer-summary-groups">
            <section className="performer-summary-group" aria-label={t("performerWorkspace.recentCheck")}>
              <h4>{t("performerWorkspace.recentCheck")}</h4>
              <dl className="performer-summary-metrics performer-summary-metrics--check">
                <div>
                  <dt>{t("performerWorkspace.checkTime")}</dt>
                  <dd title={formatDateTime(performerSubscription?.lastCheckedAt)}>{formatRelative(performerSubscription?.lastCheckedAt) ?? t("performerUi.neverChecked")}</dd>
                </div>
                <div className={performerSubscription?.lastError ? "tone-danger" : undefined}>
                  <dt>{t("performerUi.checkStatus")}</dt>
                  <dd className="performer-check-result">
                    <span className={`performer-check-state ${performerSubscription?.lastError ? "is-failed" : performerDetail.performer.subscribed && performerSubscription?.lastCheckedAt ? "is-healthy" : "is-neutral"}`}>{t(performerSubscription?.lastError ? "performerUi.failed" : performerDetail.performer.subscribed ? performerSubscription?.lastCheckedAt ? "performerUi.normal" : "performerUi.neverChecked" : "performerUi.unsubscribed")}</span>
                    {performerSubscription?.lastError ? <button type="button" className="performer-check-reason" aria-haspopup="dialog" title={t("performerWorkspace.errorDetails")} onClick={() => setShowCheckError(true)}>{t("performerWorkspace.errorReason")}</button> : null}
                  </dd>
                </div>
              </dl>
            </section>
            <section className="performer-summary-group" aria-label={t("performerWorkspace.worksAndTasks")}>
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
              {performerSummary && !performerSummary.sceneStatsComplete ? <p className="performer-summary-note">{t("performerWorkspace.partialTasksHelp")}</p> : null}
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
