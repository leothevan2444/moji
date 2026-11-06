import { useEffect, useRef, useState } from "react";
import { useLocation, useSearchParams } from "react-router";
import { useClient, useMutation } from "urql";
import { useTranslation } from "react-i18next";
import { useTorrentSearch } from "./useTorrentSearch";
import { TaskDetailDocumentDocument, ResolveTaskSourcingCandidateDocument, TorrentSearchSort, TorrentSearchState } from "../graphql/generated/graphql";
import { describeQueryError } from "../services/queryError";
import type { DashboardTask } from "../utils";

export interface TaskIndexerSearchOptions {
 task:DashboardTask;disabled:boolean;
 onResolved:(task:Pick<DashboardTask,"id"|"stage"|"stageStatus">)=>void|Promise<void>;
 onNotify?:(tone:"tone-success"|"tone-danger",message:string)=>void;
 onBusyChange?:(busy:boolean)=>void;
}
export function useTaskIndexerSearch({task,disabled,onResolved,onNotify,onBusyChange}:TaskIndexerSearchOptions){
 const client=useClient();const routeLocation=useLocation();
 const {t}=useTranslation();const [params,setParams]=useSearchParams();
 const navigationParams=useRef(params);
 useEffect(()=>{navigationParams.current=params},[routeLocation.search]);
 const scoped=params.get("sourcingTask")===task.id;const runID=scoped?params.get("sourcingRun"):null;
 const page=scoped?Math.max(1,Math.min(100,Number.parseInt(params.get("sourcingPage")??"1",10)||1)):1;
 const rawSort=scoped?params.get("sourcingSort"):null;
 const sort=Object.values(TorrentSearchSort).includes(rawSort as TorrentSearchSort)?rawSort as TorrentSearchSort:TorrentSearchSort.Relevance;
 const site=scoped?params.get("sourcingSite")??"":"";
 const keyword=scoped?(params.get("sourcingKeyword")??"").slice(0,512):"";
 const search=useTorrentSearch(runID,(page-1)*10,10,sort,site,keyword);
 const snapshot=search.snapshot?.taskId===task.id?search.snapshot:null;
 const progress=search.progressSnapshot?.taskId===task.id?search.progressSnapshot:snapshot;
 const running=progress?.state===TorrentSearchState.Queued||progress?.state===TorrentSearchState.Running;
 const wrongTask=!!search.snapshot && !snapshot;
 const [selectionError,setSelectionError]=useState<unknown>(null);const [choosing,setChoosing]=useState<string|null>(null);
 const [,resolve]=useMutation(ResolveTaskSourcingCandidateDocument);
 const life=useRef(false);const activeTask=useRef(task.id);activeTask.current=task.id;const opened=useRef<string|null>(null);const selectionLock=useRef(false);
 useEffect(()=>{life.current=true;return()=>{life.current=false}},[]);
 const location=(run:string,targetPage=1,targetSort=sort,targetSite?:string,targetKeyword?:string)=>{const next=new URLSearchParams(navigationParams.current);next.set("sourcingTask",task.id);next.set("sourcingRun",run);if(targetSite!==undefined){if(targetSite)next.set("sourcingSite",targetSite);else next.delete("sourcingSite");}if(targetKeyword!==undefined){if(targetKeyword)next.set("sourcingKeyword",targetKeyword);else next.delete("sourcingKeyword");}if(targetPage===1)next.delete("sourcingPage");else next.set("sourcingPage",String(targetPage));if(targetSort===TorrentSearchSort.Relevance)next.delete("sourcingSort");else next.set("sourcingSort",targetSort);navigationParams.current=next;setParams(next,{replace:true,state:routeLocation.state});};
 const start=async(refresh:boolean)=>{
  const target=task.id;const id=await search.start({keywords:task.code,taskId:target,refresh});
  if(id&&life.current&&activeTask.current===target)location(id,1,sort,"","");
 };
 // Opening source resolution preserves its existing automatic first search.
 // URL recovery reads only; task-scoped admission deduplicates concurrent opens.
 useEffect(()=>{if(!runID&&opened.current!==task.id){opened.current=task.id;void start(false)}},[runID,task.id,task.code]);
 const total=snapshot?.total??(progress&&(progress.filterInstanceId??"")===site&&(progress.candidateQuery??"")===keyword.trim()?progress.total:0);const pages=Math.max(1,Math.ceil(total/10));
 useEffect(()=>{if(snapshot&&!running&&!search.fetching&&page>pages&&runID)location(runID,pages)},[snapshot?.revision,runID,running,search.fetching,page,pages]);
 const choose=async(candidateId:string)=>{
  if(!runID||wrongTask||selectionLock.current||disabled||!task.revision)return;
  const target=task.id;selectionLock.current=true;setChoosing(candidateId);setSelectionError(null);onBusyChange?.(true);
  try{
   const result=await resolve({input:{taskId:target,searchId:runID,candidateId,expectedRevision:task.revision}});
   if(!life.current||activeTask.current!==target)return;
   if(result.error)throw result.error;
   const resolved=result.data?.resolveTaskSourcingCandidate;if(!resolved?.id)throw new Error(t("taskUi.resolution.noResult"));
   onNotify?.("tone-success",t("taskIndexer.accepted"));await onResolved(resolved);
  }catch(error){if(life.current&&activeTask.current===target){setSelectionError(error);onNotify?.("tone-danger",describeQueryError(error));void client.query(TaskDetailDocumentDocument,{id:target},{requestPolicy:"network-only",fetchOptions:{signal:AbortSignal.timeout(5000)}}).toPromise().catch(()=>undefined)}}
  finally{selectionLock.current=false;if(life.current)setChoosing(null);onBusyChange?.(false)}
 };
 const busy=disabled||!!choosing;const failure=selectionError??search.error;
 return {site,keyword,search,snapshot,progress,running,wrongTask,page,pages,total,sort,runID,start,location,choose,choosing,busy,failure};
}
