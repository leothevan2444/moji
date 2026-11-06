import { TorrentSearchProgress } from "./TorrentSearchProgress";
import { PreparedTorrentDownloadActions } from "./PreparedTorrentDownloadActions";
import { useId } from "react";
import { useTranslation } from "react-i18next";
import { CandidateSeeders, TorrentCandidateLinks, candidateURL } from "./TorrentCandidateLinks";
import { CandidatePagination } from "./CandidatePagination";
import { INDEXER_SORT_OPTIONS } from "../../constants";
import { describeQueryError } from "../../services/queryError";
import { formatBytes } from "../../utils";
import { useTaskIndexerSearch, type TaskIndexerSearchOptions } from "../../hooks/useTaskIndexerSearch";
import "../../styles/torrent-search.scss";

export function TaskIndexerSearch(props:TaskIndexerSearchOptions){
 const {t}=useTranslation();const heading=useId();const {task}=props;
 const {site,keyword,search,snapshot,progress,running,wrongTask,page,pages,total,sort,runID,start,location,choose,choosing,busy,failure}=useTaskIndexerSearch(props);
 return <section className="sourcing-resolution__results" aria-labelledby={heading} aria-busy={search.fetching||search.starting}>
  <div className="sourcing-resolution__search"><div className="sourcing-resolution__result-heading"><h3 id={heading}>{t("sourcingManual.candidates")}</h3>{snapshot?<span>{t("taskUi.resolution.found",{count:total})}</span>:null}</div>
   <button type="button" className="ghost-button" disabled={busy||search.starting} onClick={()=>void start(true)}>{t("taskUi.resolution.searchAgain")}</button>
  </div>
  {progress?<TorrentSearchProgress key={progress.id} snapshot={progress}>{running?<button type="button" className="ghost-button" disabled={busy||!!progress.cancelRequested||search.cancelling} onClick={()=>void search.cancel()}>{t(progress.cancelRequested?"torrentSearch.cancelling":"torrentSearch.cancel")}</button>:null}</TorrentSearchProgress>:search.starting||search.fetching?<p role="status">{t("sourcingManual.searchingFor",{code:task.code})}</p>:null}
  {failure?<p className="sourcing-resolution__error" role="alert">{describeQueryError(failure)}</p>:null}
  {wrongTask?<p className="inline-error">{t("indexerErrors.TORRENT_SEARCH_TASK_SCOPE_INVALID")}</p>:null}
  {snapshot?.errorCode?<p className="inline-error sourcing-resolution__note">{t(`indexerErrors.${snapshot.errorCode}`,{defaultValue:t("indexerErrors.INDEXER_SEARCH_INCOMPLETE")})}</p>:null}
  {snapshot?.truncated?<p>{t("torrentSearch.truncated")}</p>:null}
  {runID?<div className="candidate-filters">
   <label><span>{t("candidateFilters.site")}</span><select value={site} disabled={busy} onChange={event=>location(runID,1,sort,event.target.value)}><option value="">{t("candidateFilters.allSites")}{progress?` (${progress.sites.reduce((sum,item)=>sum+item.candidateCount,0)})`:""}</option>{progress?.sites.map(item=><option value={item.instanceId} key={item.instanceId}>{item.name} ({item.candidateCount})</option>)}</select></label>
   <label className="candidate-filters__query"><span>{t("candidateFilters.keyword")}</span><input type="search" value={keyword} maxLength={512} disabled={busy} placeholder={t("candidateFilters.placeholder")} onChange={event=>location(runID,1,sort,undefined,event.target.value)}/></label>
   <label><span>{t("discoverUi.toolbar.sort")}</span><select value={sort} disabled={busy} onChange={event=>location(runID,1,event.target.value as typeof sort)}>{INDEXER_SORT_OPTIONS.map(option=><option key={option.value} value={option.value}>{t(option.label)}</option>)}</select></label>
  </div>:null}
  {snapshot&&!running&&total===0?<p className="sourcing-resolution__empty">{t("taskUi.resolution.none")}</p>:null}
  {snapshot?.candidates.length?<ul className="sourcing-resolution__candidates">{snapshot.candidates.map((candidate,index)=><li className="sourcing-candidate" key={candidate.id}>
   <span className="sourcing-candidate__rank" aria-hidden="true">{String((page-1)*10+index+1).padStart(2,"0")}</span>
   <div className="sourcing-candidate__copy"><strong title={candidate.title}>{candidate.title}</strong><div className="sourcing-candidate__meta"><span className="sourcing-candidate__tracker">{candidate.instanceName}</span><span>{candidate.size!=null?formatBytes(Number(candidate.size)):t("torrentSearch.unknownSize")}</span><CandidateSeeders seeders={candidate.seeders}/></div><div className="sourcing-candidate__links">{candidateURL(candidate.detailsUrl)?<TorrentCandidateLinks candidate={{details:candidate.detailsUrl??""}}/>:null}{runID?<PreparedTorrentDownloadActions compact searchId={runID} candidateId={candidate.id} disabled={busy}/>:null}</div></div>
   <button type="button" className="ghost-button task-primary-action" disabled={busy||!task.revision} aria-busy={choosing===candidate.id} aria-label={t("sourcingManual.selectCandidate",{title:candidate.title})} onClick={()=>void choose(candidate.id)}>{t(choosing===candidate.id?"taskUi.resolution.selecting":"taskUi.resolution.select")}</button>
  </li>)}</ul>:null}
  {runID?<CandidatePagination page={page} pages={pages} disabled={busy||search.fetching} onChange={value=>location(runID,value)}/>:null}
 </section>;
}
