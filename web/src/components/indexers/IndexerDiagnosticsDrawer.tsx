import { useAppTimezone } from "../../hooks/useAppTimezone";
import { useCallback, useId, useState } from "react";
import { createPortal } from "react-dom";
import { useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faSearch, faCopy, faCheck } from "@fortawesome/free-solid-svg-icons";
import { Drawer } from "../layout/Drawer";
import { IndexerDiagnosticsDocument, IndexerEventKind, type IndexerDiagnosticFilterInput, type IndexerDiagnosticsQuery } from "../../graphql/generated/graphql";
import { useIndexerEvents } from "../../hooks/useIndexerEvents";
import { describeQueryError } from "../../services/queryError";
import { formatDateTime } from "../../utils";
import "../../styles/indexer-observation.scss";

type Report = IndexerDiagnosticsQuery["indexerDiagnostics"]["items"][number];
function Identifier({label,value}:{label:string;value:string}) {
 const {t}=useTranslation();const [copied,setCopied]=useState(false);const [failed,setFailed]=useState(false);
 return <span className="indexer-record-id"><span>{label}</span><button type="button" className="home-inline-button" title={value} aria-label={copied?t("indexerRecordUi.copiedIdentifier",{label}):t("indexerRecordUi.copyIdentifier",{label,value})} onClick={async()=>{try{await navigator.clipboard.writeText(value);setCopied(true);setFailed(false);}catch{setFailed(true);}}}><code>{value.length>18?`${value.slice(0,10)}…${value.slice(-6)}`:value}</code><FontAwesomeIcon icon={copied?faCheck:faCopy} aria-hidden="true"/></button>{failed?<span role="status">{t("indexerRecordUi.copyFailed")}</span>:null}</span>;
}
function Record({record}:{record:Report}){
  useAppTimezone();
 const {t}=useTranslation();const cancelled=record.errorCode==="INDEXER_CANCELLED";const failed=!cancelled&&Boolean(record.errorCode||record.partial);
 const outcome=cancelled?"cancelled":failed?"failed":"succeeded";
 const ids=[["task",record.taskId],["searchId",record.searchId],["resource",record.resourceId],["correlation",record.correlationId],["config",record.configRevision],["rules",record.bundleRevision]].filter(([,value])=>value);
 return <li className={`indexer-record${failed?" indexer-record--failed":""}`}>
  <div className="indexer-record__heading"><strong>{record.name}</strong><span>{t(record.operation==="FETCH"?"indexerDiagnostics.fetch":"indexerDiagnostics.search")}</span><span className={`indexer-record__outcome${failed?" indexer-record__outcome--failed":""}`}>{t(`indexerRecordUi.${outcome}`)}</span><time dateTime={record.finishedAt} title={t("indexerRecordUi.started",{time:formatDateTime(record.startedAt)})}>{formatDateTime(record.finishedAt)}</time></div>
  <div className="indexer-record__facts">
   {!record.executed&&!record.cacheHit?<span>{t("indexerRecordUi.notExecuted")}</span>:null}
   {record.code?<strong>{record.code}</strong>:null}
   <span>{t(`indexerDiagnostics.triggers.${record.trigger}`,{defaultValue:t("indexerDiagnostics.triggers.OTHER")})}</span>
   <span>{t(record.elapsedKnown?"indexerRecordUi.elapsed":"indexerRecordUi.legacyElapsed",{seconds:Number((record.elapsedMs/1000).toFixed(2))})}</span>
   {record.operation==="SEARCH"?<><span>{t("indexerRecordUi.candidates",{count:record.results})}</span><span>{t("indexerRecordUi.query",{mode:record.mode,limit:record.limit,offset:record.offset})}{record.categories.length?` · ${record.categories.join(", ")}`:""}</span>{record.issueCount?<span>{t("indexerRecordUi.issues",{count:record.issueCount})}</span>:null}</>:null}
   {record.cacheHit?<span>{t("indexerDiagnostics.cached")}</span>:null}{record.truncated?<span>{t("indexerDiagnostics.truncated")}</span>:null}{record.httpStatus?<span>HTTP {record.httpStatus}</span>:null}
  </div>
  {record.errorCode||record.partial?<p className="indexer-record__reason">{t(`indexerErrors.${record.errorCode||"INDEXER_SEARCH_INCOMPLETE"}`,{defaultValue:t("indexers.needsAttention")})}</p>:null}
  {record.retryAt?<p className="indexer-record__retry">{t("indexerHealth.retryAt",{time:formatDateTime(record.retryAt)})}</p>:null}
  <div className="indexer-record__identifiers">{ids.map(([key,value])=><Identifier key={key} label={t(`indexerDiagnostics.${key}`)} value={value}/>)}</div>
 </li>;
}
export function IndexerDiagnosticsDrawer({instanceId,name,onClose}:{instanceId?:string;name?:string;onClose:()=>void}){
 const {t}=useTranslation();const searchId=useId();
 const [filter,setFilter]=useState<IndexerDiagnosticFilterInput>({instanceId,limit:20});
 const [range,setRange]=useState(0);const [search,setSearch]=useState("");
 const [pages,setPages]=useState<Array<string|undefined>>([undefined]);
 const [query,requery]=useQuery({query:IndexerDiagnosticsDocument,variables:{input:{...filter,before:pages.at(-1)}},requestPolicy:"network-only"});
 const refresh=useCallback(()=>requery({requestPolicy:"network-only"}),[requery]);useIndexerEvents(refresh,[IndexerEventKind.Site]);
 const update=(patch:Partial<IndexerDiagnosticFilterInput>)=>{setPages([undefined]);setFilter(current=>({...current,...patch}));};
 const page=query.data?.indexerDiagnostics;
 const selectedName=page?.sites.find(site=>site.id===filter.instanceId)?.name||(filter.instanceId===instanceId?name:undefined);
 return createPortal(<Drawer visibleDrawer="indexer-diagnostics" closing={false} className="indexer-history-drawer" title={selectedName?t("indexerDiagnostics.siteTitle",{name:selectedName}):t("indexerDiagnostics.title")} onClose={onClose} footer={<>
  <span className="indexer-history-total">{page?t("indexerRecordUi.total",{count:page.total}):""}</span>
  <button className="ghost-button" disabled={query.fetching||pages.length===1} onClick={()=>setPages(current=>current.slice(0,-1))}>{t("indexerDiagnostics.previous")}</button>
  <button className="ghost-button" disabled={query.fetching||!page?.hasMore} onClick={()=>setPages(current=>[...current,page?.items.at(-1)?.id])}>{t("indexerDiagnostics.next")}</button>
 </>}>
  <p className="indexer-history-retention">{t("indexerDiagnostics.retention")}</p>
  <form className="indexer-history-filters" onSubmit={event=>{event.preventDefault();update({query:search.trim()||null});}}>
   <label><span>{t("indexerRecordUi.site")}</span><select value={filter.instanceId??""} onChange={event=>update({instanceId:event.target.value||null})}><option value="">{t("indexerRecordUi.allSites")}</option>{filter.instanceId&&!page?.sites.some(site=>site.id===filter.instanceId)?<option value={filter.instanceId}>{selectedName||filter.instanceId}</option>:null}{page?.sites.map(site=><option key={site.id} value={site.id}>{site.name}</option>)}</select></label>
   <label><span>{t("indexerRecordUi.result")}</span><select value={filter.outcome??""} onChange={event=>update({outcome:event.target.value||null})}><option value="">{t("indexerRecordUi.allResults")}</option>{["SUCCEEDED","FAILED","CANCELLED"].map(value=><option key={value} value={value}>{t(`indexerRecordUi.${value.toLowerCase()}`)}</option>)}</select></label>
   <label><span>{t("indexerRecordUi.operation")}</span><select value={filter.operation??""} onChange={event=>update({operation:event.target.value||null})}><option value="">{t("indexerRecordUi.allOperations")}</option><option value="SEARCH">{t("indexerDiagnostics.search")}</option><option value="FETCH">{t("indexerDiagnostics.fetch")}</option></select></label>
   <label><span>{t("indexerRecordUi.time")}</span><select value={range} onChange={event=>{const hours=Number(event.target.value);setRange(hours);update({since:hours?new Date(Date.now()-hours*3600000).toISOString():null});}}>{[0,1,24,168].map(hours=><option key={hours} value={hours}>{t(hours?"indexerRecordUi.hours":"indexerRecordUi.allTime",{count:hours})}</option>)}</select></label>
   <div className="indexer-history-search"><label htmlFor={searchId}>{t("indexerRecordUi.search")}</label><div><input id={searchId} type="search" value={search} maxLength={256} placeholder={t("indexerRecordUi.searchPlaceholder")} onChange={event=>setSearch(event.target.value)}/><button className="ghost-button" type="submit" aria-label={t("indexerRecordUi.applySearch")}><FontAwesomeIcon icon={faSearch} aria-hidden="true"/></button></div></div>
  </form>
  {query.error?<p role="alert">{describeQueryError(query.error)} <button onClick={refresh}>{t("common.retry")}</button></p>:null}
  {query.fetching&&!page?<p role="status">{t("common.loading")}</p>:null}
  {page?.items.length===0?<p>{t("indexerDiagnostics.empty")}</p>:null}
  <ul className="indexer-diagnostics" aria-busy={query.fetching}>{page?.items.map(record=><Record key={record.id} record={record}/>)}</ul>
 </Drawer>,document.body);
}
