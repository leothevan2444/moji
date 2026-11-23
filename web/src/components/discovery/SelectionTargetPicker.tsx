import { useEffect, useRef, useState } from "react";
import { useClient } from "urql";
import { useTranslation } from "react-i18next";
import { TaskFilmCandidatesDocument, ConfirmedTaskFilmDocument, TaskFilmFieldsFragmentDoc, type TaskFilmFieldsFragment } from "../../graphql/generated/graphql";
import { useFragment } from "../../graphql/generated";
import { describeQueryError } from "../../services/queryError";
export function SelectionTargetPicker({keywords,value,onChange}:{keywords:string;value:TaskFilmFieldsFragment|null;onChange:(value:TaskFilmFieldsFragment|null)=>void}){
 const {t}=useTranslation();const client=useClient();const [items,setItems]=useState<TaskFilmFieldsFragment[]>([]);const [error,setError]=useState<string|null>(null);const [busy,setBusy]=useState(false);const ticket=useRef(0);
 useEffect(()=>()=>{ticket.current++;},[]);
 const search=async()=>{const current=++ticket.current;setBusy(true);setError(null);try{const response=await client.query(TaskFilmCandidatesDocument,{input:{keywords}},{requestPolicy:"network-only"}).toPromise();if(response.error)throw response.error;if(current===ticket.current)setItems((response.data?.taskFilmCandidates??[]).map(v=>useFragment(TaskFilmFieldsFragmentDoc,v)));}catch(e){if(current===ticket.current)setError(describeQueryError(e));}finally{if(current===ticket.current)setBusy(false);}};
 const choose=async(endpoint:string,sceneId:string)=>{const current=++ticket.current;setBusy(true);setError(null);try{const response=await client.query(ConfirmedTaskFilmDocument,{endpoint,sceneId},{requestPolicy:"network-only"}).toPromise();if(response.error)throw response.error;if(current===ticket.current&&response.data){onChange(useFragment(TaskFilmFieldsFragmentDoc,response.data.taskFilm));setItems([]);}}catch(e){if(current===ticket.current)setError(describeQueryError(e));}finally{if(current===ticket.current)setBusy(false);}};
 return <div className="resource-selection-target">{value?<><span>{t("resourceSelection.target",{code:value.code})}</span><button type="button" className="ghost-button" onClick={()=>onChange(null)}>{t("resourceSelection.changeTarget")}</button></>:<button type="button" className="ghost-button" disabled={busy||!keywords} onClick={()=>void search()}>{t("resourceSelection.chooseTarget")}</button>}
 {error?<p role="alert">{error}</p>:null}{items.length?<ul>{items.map(v=><li key={`${v.endpoint}/${v.sceneId}`}><button className="ghost-button" type="button" disabled={busy} onClick={()=>void choose(v.endpoint,v.sceneId)}>{v.code} · {v.title}</button></li>)}</ul>:null}
 </div>;
}
