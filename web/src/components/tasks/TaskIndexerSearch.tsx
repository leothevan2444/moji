import { PreparedTorrentDownloadActions } from "./PreparedTorrentDownloadActions";
import { useId } from "react";
import { useTranslation } from "react-i18next";
import { CandidateSeeders, TorrentCandidateLinks, candidateURL } from "./TorrentCandidateLinks";
import { SortAndPagination } from "../drawers/SortAndPagination";
import { INDEXER_SORT_OPTIONS } from "../../constants";
import { describeQueryError } from "../../services/queryError";
import { formatBytes } from "../../utils";
import { useTaskIndexerSearch, type TaskIndexerSearchOptions } from "../../hooks/useTaskIndexerSearch";
import "../../styles/torrent-search.scss";
import "../../styles/discovery-toolbar.scss";

export function TaskIndexerSearch(props:TaskIndexerSearchOptions){
 const {t}=useTranslation();const heading=useId();const {task}=props;
 const {search,snapshot,running,wrongTask,page,pages,total,sort,runID,start,location,choose,choosing,busy,failure}=useTaskIndexerSearch(props);
 return <section className="sourcing-resolution__results" aria-labelledby={heading} aria-busy={search.fetching||search.starting}>
  <div className="sourcing-resolution__search"><div className="sourcing-resolution__result-heading"><h3 id={heading}>{t("sourcingManual.candidates")}</h3>{snapshot?<span>{t("taskUi.resolution.found",{count:total})}</span>:null}</div>
   <button type="button" className="ghost-button" disabled={busy||search.starting} onClick={()=>void start(true)}>{t("taskUi.resolution.searchAgain")}</button>
  </div>
  {snapshot?<p role="status">{t(`torrentSearch.states.${snapshot.state}`)}</p>:search.starting||search.fetching?<p role="status">{t("sourcingManual.searchingFor",{code:task.code})}</p>:null}
  {failure?<p className="sourcing-resolution__error" role="alert">{describeQueryError(failure)}</p>:null}
  {wrongTask?<p className="inline-error">{t("indexerErrors.TORRENT_SEARCH_TASK_SCOPE_INVALID")}</p>:null}
  {running?<button type="button" className="ghost-button" disabled={busy||!!snapshot?.cancelRequested||search.cancelling} onClick={()=>void search.cancel()}>{t(snapshot?.cancelRequested?"torrentSearch.cancelling":"torrentSearch.cancel")}</button>:null}
  {snapshot?.errorCode?<p className="inline-error">{t(`indexerErrors.${snapshot.errorCode}`,{defaultValue:t("indexerErrors.INDEXER_SEARCH_INCOMPLETE")})}</p>:null}
  {snapshot?.truncated?<p>{t("torrentSearch.truncated")}</p>:null}
  {snapshot?.sites.length?<details className="torrent-search__sites"><summary>{t("torrentSearch.sites")}</summary><ul>{snapshot.sites.map(site=><li key={site.instanceId}>{site.name} · {t(`torrentSearch.siteStates.${site.state}`)}{site.errorCode?` · ${t(`indexerErrors.${site.errorCode}`,{defaultValue:t("indexerErrors.INDEXER_REQUEST_FAILED")})}`:""}</li>)}</ul></details>:null}
  {snapshot?<SortAndPagination sortValue={sort} sortOptions={INDEXER_SORT_OPTIONS} onSortChange={value=>{if(runID&&!busy)location(runID,1,value)}} page={page} totalPages={pages} total={total} onPrevPage={()=>{if(runID&&!busy)location(runID,Math.max(1,page-1))}} onNextPage={()=>{if(runID&&!busy)location(runID,Math.min(pages,page+1))}}/>:null}
  {snapshot&&!running&&total===0?<p className="sourcing-resolution__empty">{t("taskUi.resolution.none")}</p>:null}
  {snapshot?.candidates.length?<ul className="sourcing-resolution__candidates">{snapshot.candidates.map((candidate,index)=><li className="sourcing-candidate" key={candidate.id}>
   <span className="sourcing-candidate__rank" aria-hidden="true">{String((page-1)*10+index+1).padStart(2,"0")}</span>
   <div className="sourcing-candidate__copy"><strong title={candidate.title}>{candidate.title}</strong><div className="sourcing-candidate__meta"><span className="sourcing-candidate__tracker">{candidate.instanceName}</span><span>{candidate.size!=null?formatBytes(Number(candidate.size)):t("torrentSearch.unknownSize")}</span><CandidateSeeders seeders={candidate.seeders}/></div>{candidateURL(candidate.detailsUrl)?<TorrentCandidateLinks candidate={{details:candidate.detailsUrl??""}}/>:null}{runID?<PreparedTorrentDownloadActions searchId={runID} candidateId={candidate.id} disabled={busy}/>:null}</div>
   <button type="button" className="ghost-button task-primary-action" disabled={busy||!task.revision} aria-busy={choosing===candidate.id} aria-label={t("sourcingManual.selectCandidate",{title:candidate.title})} onClick={()=>void choose(candidate.id)}>{t(choosing===candidate.id?"taskUi.resolution.selecting":"taskUi.resolution.select")}</button>
  </li>)}</ul>:null}
 </section>;
}
