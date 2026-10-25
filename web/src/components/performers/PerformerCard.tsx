import { faBookmark, faClipboardCheck, faHeart, faPlayCircle, faRotate } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useState } from "react";
import { createPortal } from "react-dom";
import { Drawer } from "../layout/Drawer";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { formatDateTime, formatRelative, performerImageURL, performerInitials, stashPerformerURL } from "../../utils";
import type { PerformersPageProps, StashPerformerEntry, SubscribedPerformerEntry } from "./types";

type CardPage = Pick<PerformersPageProps, "stashBaseURL" | "performerSelectionMode" | "selectedPerformerIds" | "performerBatchPending" | "pendingSubscriptionID" | "onTogglePerformerSelection" | "onOpenPerformer" | "onToggle" | "onRefreshOne">;

export function PerformerCard({ performer, subscription, reviewCount, page }: { performer: StashPerformerEntry; subscription: SubscribedPerformerEntry | null; reviewCount: number; page: CardPage; }) {
  const { t } = useTranslation();
  const [showCheckError, setShowCheckError] = useState(false);
  const { stashBaseURL, performerSelectionMode, selectedPerformerIds, performerBatchPending, pendingSubscriptionID, onTogglePerformerSelection, onOpenPerformer, onToggle, onRefreshOne } = page;
  const selected = selectedPerformerIds.includes(performer.id);
  const pending = pendingSubscriptionID === performer.id;
  const imageURL = performerImageURL(performer.imagePath, stashBaseURL);
  const stashURL = stashPerformerURL(performer.id, stashBaseURL);
  const aliases = performer.aliasList.join(" / ");
  return (
    <>
      <article className={`profile-card${performerSelectionMode ? " profile-card--selectable" : ""}${selected ? " is-selected" : ""}`} onClick={event => {
        if ((event.target as Element).closest("a, button, input, label, details")) return;
        if (performerSelectionMode) { if (!performerBatchPending) onTogglePerformerSelection(performer.id); }
        else onOpenPerformer(performer.id);
      }}>
        <div className="avatar avatar--frame">
          <span className="avatar__fallback" aria-hidden="true">{performerInitials(performer.name)}</span>
          {imageURL ? <img className="avatar__image" src={imageURL} alt="" loading="lazy" onError={event => { event.currentTarget.style.display = "none"; }} /> : null}
        </div>
        <div className="profile-card__body">
          <div className="profile-card__head">
            <div className="performer-card-identity">
              <h3 title={performer.name}>{performerSelectionMode ? performer.name : <Link to={`/performers/${encodeURIComponent(performer.id)}`} onClick={event => {
                if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
                event.preventDefault(); onOpenPerformer(performer.id);
              }}>{performer.name}</Link>}</h3>
              {aliases ? <p className="performer-card-aliases" title={aliases}>{aliases}</p> : null}
            </div>
            <div className="performer-card-flags">
              <span
                className={`profile-icon profile-icon--favorite${performer.favorite ? " is-active" : ""}`}
                role="img"
                title={t(performer.favorite ? "performerUi.favorite" : "performerUi.notFavorite")}
                aria-label={t(performer.favorite ? "performerUi.favorite" : "performerUi.notFavorite")}
              ><FontAwesomeIcon icon={faHeart} /></span>
              {performerSelectionMode ? (
                <span className={`profile-icon profile-icon--subscribe${performer.subscribed ? " is-active" : ""}`} role="img" title={t(performer.subscribed ? "performerUi.subscribed" : "performerUi.unsubscribed")} aria-label={t(performer.subscribed ? "performerUi.subscribed" : "performerUi.unsubscribed")}><FontAwesomeIcon icon={faBookmark} /></span>
              ) : (
                <button
                  type="button"
                  className={`profile-icon profile-icon--subscribe${performer.subscribed ? " is-active" : ""}`}
                  disabled={pending}
                  aria-pressed={performer.subscribed}
                  aria-label={t(performer.subscribed ? "performerUi.unsubscribeName" : "performerUi.subscribeName", { name: performer.name })}
                  title={t(performer.subscribed ? "performerUi.removeSubscription" : "performerUi.addSubscription")}
                  onClick={() => onToggle(performer)}
                ><FontAwesomeIcon icon={faBookmark} /></button>
              )}
              {performerSelectionMode ? <input type="checkbox" className="performer-card-selector" checked={selected} disabled={performerBatchPending} onChange={() => onTogglePerformerSelection(performer.id)} aria-label={t(selected ? "performerUi.deselectName" : "performerUi.selectName", { name: performer.name })} /> : null}
            </div>
            <div className="performer-card-check-row">
              {performer.subscribed ? <p className="profile-check-meta" title={formatDateTime(subscription?.lastCheckedAt)}>{subscription?.lastCheckedAt ? t("performerUi.lastChecked", { time: formatRelative(subscription.lastCheckedAt) }) : t("performerUi.neverChecked")}</p> : null}
              {subscription?.lastError ? <button type="button" className="performer-card-error-button" aria-haspopup="dialog" title={t("performerWorkspace.errorDetails")} onClick={() => setShowCheckError(true)}>{t("performerUi.failed")}</button> : null}
            </div>
          </div>
          <div className="profile-card__footer">
            {reviewCount > 0 || !performerSelectionMode ? <div className="profile-actions">
              {reviewCount > 0 ? <Link className="profile-action-icon profile-action-icon--count" aria-label={t("releaseReview.actorEntry", { count: reviewCount })} title={t("releaseReview.actorEntry", { count: reviewCount })} to={`/performers/review?performer=${encodeURIComponent(performer.id)}`}><FontAwesomeIcon icon={faClipboardCheck} /><span aria-hidden="true">{reviewCount}</span></Link> : null}
              {!performerSelectionMode ? <>
                {performer.subscribed ? <button type="button" className="profile-action-icon" title={t("performerUi.checkNow")} aria-label={t("performerUi.checkName", { name: performer.name })} disabled={pending} onClick={() => onRefreshOne(performer)}><FontAwesomeIcon icon={faRotate} className={pending ? "is-spinning" : undefined} /></button> : null}
                {stashURL ? <a className="profile-action-icon profile-action-icon--count" href={stashURL} target="_blank" rel="noreferrer" title={t("performerWorkspace.stashLibrary", { name: performer.name, count: performer.sceneCount })} aria-label={t("performerWorkspace.stashLibrary", { name: performer.name, count: performer.sceneCount })}><FontAwesomeIcon icon={faPlayCircle} /><span aria-hidden="true">{performer.sceneCount}</span></a> : null}
              </> : null}
            </div> : null}
          </div>
        </div>
      </article>
      {showCheckError && subscription?.lastError ? createPortal(
        <Drawer visibleDrawer="confirm" closing={false} title={t("performerUi.failed")} onClose={() => setShowCheckError(false)}>
          <h3>{performer.name}</h3>
          <p className="performer-card-error-message">{subscription.lastError}</p>
        </Drawer>, document.body,
      ) : null}
    </>
  );
}
