import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { SceneArtwork } from "../common/SceneArtwork";
import type { UpcomingReleasesQuery } from "../../graphql/generated/graphql";

export type UpcomingRelease = UpcomingReleasesQuery["upcomingReleases"]["items"][number];

interface Props {
  release: UpcomingRelease;
  status: string | null;
  onOpen: () => void;
}

export function UpcomingReleaseCard({ release, status, onOpen }: Props) {
  const { t } = useTranslation();
  const name = release.code || t("upcoming.unknownCode");
  return <article className="candidate-card performer-scene-card has-media upcoming-card">
    <button type="button" className="upcoming-card__open" onClick={onOpen} aria-label={t("upcoming.open", { name })}>
      <div className="performer-scene-card__media">
        <SceneArtwork src={release.imageUrl} alt="" />
      </div>
      <div className="performer-scene-card__content">
        <div className="performer-scene-card__status-row">
          <h3 title={name}>{name}</h3>
          <time className="performer-scene-card__date" dateTime={release.date || undefined} aria-label={t("upcoming.expected", { date: release.date })}>
            {release.date}
          </time>
        </div>
        <p className="performer-scene-card__title" title={release.title}>{release.title}</p>
      </div>
    </button>
    <div className="upcoming-card__footer">
      <div className="upcoming-card__performers">
        {release.performers.map(actor => <Link key={actor.id} to={`/performers/${encodeURIComponent(actor.id)}`}>{actor.name}</Link>)}
      </div>
      {status ? <span className="status-chip tone-warn upcoming-card__state">{status}</span> : null}
    </div>
  </article>;
}
