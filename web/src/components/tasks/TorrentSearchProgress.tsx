import type { ReactNode } from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faChevronDown, faCircleExclamation, faClock, faMagnifyingGlass, faMinus } from "@fortawesome/free-solid-svg-icons";
import { useTranslation } from "react-i18next";
import type { TorrentSearchFieldsFragment } from "../../graphql/generated/graphql";

type Site = TorrentSearchFieldsFragment["sites"][number];
const active = (site: Site) => site.state === "RUNNING" || site.state === "QUEUED";
const issue = (site: Site) => site.state === "FAILED" || site.state === "INTERRUPTED" || site.partial || Boolean(site.errorCode);

export function TorrentSearchProgress({ snapshot, children }: { snapshot: Pick<TorrentSearchFieldsFragment, "state" | "sites">; children?: ReactNode }) {
  const { t, i18n } = useTranslation();
  const { sites, state } = snapshot;
  const searching = sites.filter(site => site.state === "RUNNING").length;
  const queued = sites.filter(site => site.state === "QUEUED").length;
  const issues = sites.filter(issue).length;
  const status = <span className="torrent-search-progress__state" role="status">{t(`torrentSearch.states.${state === "PARTIAL" ? "COMPLETED" : state}`)}</span>;
  return <div className={`torrent-search-progress${children ? " torrent-search-progress--with-actions" : ""}`}>
    {sites.length ? <details className="torrent-search-progress__details">
      <summary>
        {status}
        <span className="torrent-search-progress__counts">
          <span>{t("torrentSearch.progress.finished", { finished: sites.filter(site => !active(site)).length, total: sites.length })}</span>
          {searching ? <span>{t("torrentSearch.progress.searching", { count: searching })}</span> : null}
          {queued ? <span>{t("torrentSearch.progress.waiting", { count: queued })}</span> : null}
          {issues ? <span className="torrent-search-progress__issue"><FontAwesomeIcon icon={faCircleExclamation} aria-hidden="true" /> {t("torrentSearch.progress.issues", { count: issues })}</span> : null}
        </span>
        <span className="torrent-search-progress__toggle">{t("torrentSearch.progress.details")} <FontAwesomeIcon icon={faChevronDown} aria-hidden="true" /></span>
      </summary>
      <ul className="torrent-search-progress__list" aria-label={t("torrentSearch.sites")} tabIndex={0}>
        {sites.map(site => <li key={site.instanceId}>
          <div className="torrent-search-progress__site-heading"><strong>{site.name}</strong>{site.kind ? <span className="torrent-search__kind">{t(site.kind === "BUILTIN" ? "torrentSearch.builtin" : "indexers.external")}</span> : null}</div>
          <div className={`torrent-search-progress__site-meta${issue(site) ? " torrent-search-progress__issue" : ""}`}>
            <span><FontAwesomeIcon icon={issue(site) ? faCircleExclamation : site.state === "COMPLETED" ? faCheck : site.state === "RUNNING" ? faMagnifyingGlass : site.state === "QUEUED" ? faClock : faMinus} aria-hidden="true" /> {t(`torrentSearch.siteStates.${site.state}`)}</span>
            {site.state === "COMPLETED" || site.results > 0 ? <span>{t("torrentSearch.progress.results", { count: site.results })}</span> : null}
            {site.cacheHit ? <span>{t("torrentSearch.cached")}</span> : null}
          </div>
          {site.errorCode ? <p className="torrent-search-progress__error">{i18n.exists(`indexerErrors.${site.errorCode}`) ? t(`indexerErrors.${site.errorCode}`) : t("indexerErrors.INDEXER_REQUEST_FAILED")}</p> : null}
        </li>)}
      </ul>
    </details> : status}
    {children ? <div className="torrent-search-progress__actions">{children}</div> : null}
  </div>;
}
