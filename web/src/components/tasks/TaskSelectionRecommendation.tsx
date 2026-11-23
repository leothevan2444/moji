import { useEffect, useState } from "react";
import { ResourceRuleGroupPicker } from "../discovery/ResourceRuleGroupPicker";
import { useSearchParams } from "react-router";
import { useTranslation } from "react-i18next";
import { useTorrentSelectionPreview } from "../../hooks/useTorrentSelectionPreview";
import { ResourceEvaluationDetails } from "./ResourceEvaluationDetails";
import { describeQueryError } from "../../services/queryError";

export function TaskSelectionRecommendation({taskId,searchId,revision,disabled,onSelect}:{taskId:string;searchId:string;revision:number;disabled:boolean;onSelect:(id:string)=>void}){
 const {t}=useTranslation();const [params,setParams]=useSearchParams();const savedGroup=params.get("sourcingGroup")??undefined;const [groupID,setGroupID]=useState(savedGroup);useEffect(()=>setGroupID(savedGroup),[savedGroup]);
 const preview=useTorrentSelectionPreview();
 const result=preview.value?.searchId===searchId&&preview.value.searchRevision===revision?preview.value:null;
 const candidate=result&&result.evaluations[0]?.status==="ACCEPTABLE"?result.candidates[0]:null;
 const start=async()=>{await preview.preview({searchId,taskId,groupID,limit:1});};
 return <div className="resource-recommendation">
 <ResourceRuleGroupPicker value={groupID} disabled={disabled||preview.busy} onChange={value=>{setGroupID(value);preview.clear();setParams(current=>{const next=new URLSearchParams(current);next.set("sourcingGroup",value);return next;},{replace:true});}}/>
  <button type="button" className="ghost-button" disabled={disabled||preview.busy} onClick={()=>void start()}>{t(preview.busy?"resourceSelection.previewing":"resourceSelection.recommend")}</button>

  {preview.error?<p className="inline-error" role="alert">{describeQueryError(preview.error)}</p>:null}
  {result?.partial?<p>{t("resourceSelection.partial")}</p>:null}
  {candidate?<div><strong>{candidate.title}</strong><ResourceEvaluationDetails evaluation={result!.evaluations[0]}/><button type="button" className="ghost-button" disabled={disabled} onClick={()=>onSelect(candidate.id)}>{t("resourceSelection.chooseRecommended")}</button></div>:result?<p>{t("resourceSelection.ends.NO_ACCEPTABLE_RESOURCE")}</p>:null}
 </div>;
}
