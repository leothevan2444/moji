import { formatDurationSeconds } from "../../utils";
import { SkeletonCardList } from "../common/Skeleton";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowUpRightFromSquare } from "@fortawesome/free-solid-svg-icons/faArrowUpRightFromSquare";
import type { DiscoverScenesDocumentQuery } from "../../graphql/generated/graphql";
import { useTranslation } from "react-i18next";
type DiscoverResult = DiscoverScenesDocumentQuery["discoverScenes"]["items"][number];
interface DiscoveryDrawerProps {
 query:string; searching:boolean;error:Error|null;pendingAddId:string|null;
 discoverItems:DiscoverResult[];hasAnyResults:boolean;usedStashBoxName?:string|null;
 onQueueDiscovered:(result:DiscoverResult)=>void;onTryIndexers:()=>void;
}
export function DiscoveryDrawer({query,searching,error,pendingAddId,discoverItems,hasAnyResults,usedStashBoxName,onQueueDiscovered,onTryIndexers}:DiscoveryDrawerProps){
 const {t}=useTranslation();
 return <>
 {error?<p className="inline-error">{error.message}</p>:null}
 {searching && !hasAnyResults ? <SkeletonCardList count={6}/> : !hasAnyResults ? <div className="empty-card empty-card--wide"><h3>{t(query.trim()?"discoverUi.result.noStash":"discoverUi.result.enter")}</h3><p>{t(query.trim()?"torrentSearch.tryDetail":"discoverUi.result.enterDetail")}</p>{query.trim()?<button type="button" className="ghost-button" onClick={onTryIndexers}>{t("torrentSearch.search")}</button>:null}</div> : (
        <div className="discovery-results">
          {discoverItems.map((result) => (
            <article key={result.key} className="candidate-card candidate-card--stashbox">
              {result.imageUrl ? (
                <div className="candidate-card__poster candidate-card__poster--discovery">
                  <img src={result.imageUrl} alt={result.title} loading="lazy" onError={(event) => { event.currentTarget.style.display = "none"; }} />
                </div>
              ) : null}
              <div className="candidate-card__content">
                <div className="candidate-card__head">
                  <div>
                    <div className="candidate-card__title-row">
                      <h3>
                        <span className="candidate-card__title-text">{result.title}</span>
                        {result.durationSeconds ? (
                          <span className="candidate-card__meta-chip">
                            {formatDurationSeconds(result.durationSeconds)}
                          </span>
                        ) : null}
                      </h3>
                      {usedStashBoxName ? (
                        <span className="status-chip tone-info" title={t("discoverUi.result.source")}>
                          {usedStashBoxName}
                        </span>
                      ) : null}
                    </div>
                    <div className="candidate-card__meta">
                      {result.code ? (
                        <span className="candidate-card__meta-line" title={result.code}>
                          {t("discoverUi.result.code", { code: result.code })}
                        </span>
                      ) : null}
                      {result.studioName ? (
                        <span className="candidate-card__meta-line" title={result.studioName}>
                          {t("discoverUi.result.studio", { studio: result.studioName })}
                        </span>
                      ) : null}
                      {result.performerNames.length > 0 ? (
                        <span className="candidate-card__meta-line" title={result.performerNames.join(" / ")}>
                          {t("discoverUi.result.performers", { performers: result.performerNames.slice(0, 3).join(" / ") })}
                          {result.performerNames.length > 3 ? " …" : ""}
                        </span>
                      ) : null}
                    </div>
                  </div>
                </div>
                <div className="candidate-card__foot">
                  <span>{result.date || t("discoverUi.result.noDate")}</span>
                  <div className="inline-actions">
                    {result.url ? (
                      <a
                        href={result.url}
                        target="_blank"
                        rel="noreferrer"
                        className="icon-button"
                        title={t("discoverUi.result.original")}
                        aria-label={t("discoverUi.result.original")}
                      >
                        <FontAwesomeIcon icon={faArrowUpRightFromSquare} />
                      </a>
                    ) : null}
                    <button
                      type="button"
                      onClick={() => onQueueDiscovered(result)}
                      disabled={pendingAddId === result.key}
                    >
                      {pendingAddId === result.key ? t("discoverUi.result.queueing") : t("discoverUi.result.queue")}
                    </button>
                  </div>
                </div>
              </div>
            </article>
          ))}
        </div>
 )}
 </>;
}
