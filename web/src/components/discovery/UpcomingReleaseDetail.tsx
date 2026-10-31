import { faArrowUpRightFromSquare } from "@fortawesome/free-solid-svg-icons/faArrowUpRightFromSquare";
import { faRotateRight } from "@fortawesome/free-solid-svg-icons/faRotateRight";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { Drawer } from "../layout/Drawer";
import { SceneArtwork } from "../common/SceneArtwork";
import type { UpcomingRelease } from "./UpcomingReleaseCard";

interface Props {
  release: UpcomingRelease;
  status: string | null;
  sourceUrl: string | null;
  refreshing: boolean;
  feedback: string;
  onRefresh: () => void;
  onClose: () => void;
}

export function UpcomingReleaseDetail({ release, status, sourceUrl, refreshing, feedback, onRefresh, onClose }: Props) {
  const { t } = useTranslation();
  return <Drawer visibleDrawer="discovery" variant="media" closing={false} title={release.code || t("upcoming.title")} onClose={onClose}>
    <div className="upcoming-detail">
      <div className="upcoming-player" role="region" aria-label={t("upcoming.trailer")}>
        <SceneArtwork src={release.imageUrl} alt="" loading="eager" />
        <span className="upcoming-player__unavailable">{t("upcoming.noTrailer")}</span>
      </div>
      <div className="upcoming-detail__info">
        <h3>{release.title}</h3>
        <div className="upcoming-detail__toolbar">
          <div className="upcoming-detail__metadata">
            <time dateTime={release.date || undefined}>{t("upcoming.expected", { date: release.date })}</time>
            {status ? <span className="status-chip tone-warn">{status}</span> : null}
          </div>
          <div className="upcoming-detail__actions">
            {sourceUrl ? <a className="profile-action-icon" href={sourceUrl} target="_blank" rel="noreferrer" referrerPolicy="no-referrer" aria-label={t("upcoming.source")} title={t("upcoming.source")}><FontAwesomeIcon icon={faArrowUpRightFromSquare} /></a> : null}
            <button type="button" className="profile-action-icon" disabled={refreshing || release.state === "HANDED_OFF"} aria-label={t(refreshing ? "upcoming.updating" : "upcoming.update")} title={t(refreshing ? "upcoming.updating" : "upcoming.update")} onClick={onRefresh}>
              <FontAwesomeIcon icon={faRotateRight} spin={refreshing} />
            </button>
          </div>
        </div>
        <div className="upcoming-detail__performers">
          {release.performers.map(actor => <Link key={actor.id} to={`/performers/${encodeURIComponent(actor.id)}`} onClick={onClose}>{actor.name}</Link>)}
        </div>
        {release.state === "HANDED_OFF" && release.decision === "QUEUED" ? <Link to="/performers/review" onClick={onClose}>{t("upcoming.review")}</Link> : null}
        {feedback ? <p className="upcoming-detail__feedback" role="status">{feedback}</p> : null}
      </div>
    </div>
  </Drawer>;
}
