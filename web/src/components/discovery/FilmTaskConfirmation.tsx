import { useEffect, useRef, useState } from "react";
import { useClient } from "urql";
import { useTranslation } from "react-i18next";
import { Drawer } from "../layout/Drawer";
import { SceneArtwork } from "../common/SceneArtwork";
import { useFragment } from "../../graphql/generated/fragment-masking";
import { TaskFilmSourcesDocument, TaskFilmCandidatesDocument, ConfirmedTaskFilmDocument, CreateFilmTaskDocument, TaskFilmFieldsFragmentDoc, type TaskFilmFieldsFragment } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";

export type ResourceChoice = { searchId?: string; candidateId?: string; title?: string; sourceName?: string; resourceUrl?: string };
export function FilmTaskConfirmation({ resource, onClose, onCreated }: { resource: ResourceChoice; onClose: () => void; onCreated: (id: string) => void }) {
 const {t}=useTranslation();const client=useClient();const [keywords,setKeywords]=useState("");const [endpoint,setEndpoint]=useState("");
 const [sources,setSources]=useState<Array<{name:string;endpoint:string}>>([]);
 const [resourceUrl,setResourceUrl]=useState(resource.resourceUrl??"");const [items,setItems]=useState<TaskFilmFieldsFragment[]>([]);const [selected,setSelected]=useState<TaskFilmFieldsFragment|null>(null);
 const [busy,setBusy]=useState(false);const [error,setError]=useState<string|null>(null);const [acknowledged,setAcknowledged]=useState(false);const generation=useRef(0);const submitting=useRef(false);
 useEffect(()=>{let active=true;void client.query(TaskFilmSourcesDocument,{}, {requestPolicy:"network-only"}).toPromise().then(response=>{if(active&&response.data)setSources(response.data.taskFilmSources)});return()=>{active=false;generation.current++}},[client]);
 const search=async(automatic=false)=>{
  const version=++generation.current;setBusy(true);setError(null);setSelected(null);setAcknowledged(false);
  try {
   const response=await client.query(TaskFilmCandidatesDocument,{input:{keywords:automatic?undefined:keywords,endpoint:endpoint||undefined,searchId:resource.searchId,candidateId:resource.candidateId,resourceUrl:resource.searchId?undefined:resourceUrl}},{requestPolicy:"network-only"}).toPromise();
   if(response.error)throw response.error;if(version!==generation.current)return;
   const found=(response.data?.taskFilmCandidates??[]).map(item=>useFragment(TaskFilmFieldsFragmentDoc,item));setItems(found);
  }catch(e){if(version===generation.current)setError(describeQueryError(e));}finally{if(version===generation.current)setBusy(false)}
 };
 useEffect(()=>{if(resource.searchId||resource.resourceUrl)void search(true)},[]);
 const choose=async(item:TaskFilmFieldsFragment)=>{
  const version=++generation.current;setBusy(true);setError(null);setSelected(null);setAcknowledged(false);
  try{const response=await client.query(ConfirmedTaskFilmDocument,{endpoint:item.endpoint,sceneId:item.sceneId},{requestPolicy:"network-only"}).toPromise();if(response.error)throw response.error;if(version!==generation.current)return;if(response.data)setSelected(useFragment(TaskFilmFieldsFragmentDoc,response.data.taskFilm));}
  catch(e){if(version===generation.current)setError(describeQueryError(e));}finally{if(version===generation.current)setBusy(false)}
 };
 const confirm=async()=>{
  if(!selected||busy||submitting.current||!acknowledged)return;submitting.current=true;const version=generation.current;setBusy(true);setError(null);
  try{const response=await client.mutation(CreateFilmTaskDocument,{input:{endpoint:selected.endpoint,sceneId:selected.sceneId,confirmationDigest:selected.confirmationDigest,searchId:resource.searchId,candidateId:resource.candidateId,resourceUrl:resource.searchId?undefined:resourceUrl||undefined,resourceConfirmed:true}}).toPromise();if(response.error)throw response.error;if(version===generation.current&&response.data?.createFilmTask.id)onCreated(response.data.createFilmTask.id);}
  catch(e){if(version===generation.current)setError(describeQueryError(e));}finally{if(version===generation.current)setBusy(false);submitting.current=false}
 };
 const isResource=!!resource.searchId||!!resourceUrl;
 return <Drawer visibleDrawer="confirm" closing={false} title={t("filmCreation.title")} onClose={()=>{if(!submitting.current)onClose()}} closeDisabled={submitting.current}>
  <div className="film-confirmation">
   {!selected&&resource.title?<p className="film-confirmation__resource">{resource.title}</p>:null}
   {!selected&&!resource.searchId?<label>{t("filmCreation.resourceUrl")}<input value={resourceUrl} disabled={busy} onChange={e=>{setResourceUrl(e.target.value);setSelected(null);setAcknowledged(false)}}/><button type="button" className="ghost-button" disabled={busy||!resourceUrl.trim()} onClick={()=>void search(true)}>{t("filmCreation.match")}</button></label>:null}
   {!selected?<form onSubmit={e=>{e.preventDefault();void search()}}>
    <label>{t("filmCreation.keywords")}<input value={keywords} disabled={busy} onChange={e=>setKeywords(e.target.value)}/></label>
    <label>{t("filmCreation.source")}<select value={endpoint} disabled={busy} onChange={e=>{setEndpoint(e.target.value);setSelected(null);setAcknowledged(false)}}><option value="">{t("filmCreation.preferredSource")}</option>{sources.map(source=><option key={source.endpoint} value={source.endpoint}>{source.name||source.endpoint}</option>)}</select></label>
    <button type="submit" disabled={busy||!keywords.trim()}>{t("filmCreation.search")}</button>
   </form>:null}
   {busy?<p role="status">{t("filmCreation.loading")}</p>:null}
   {error?<p className="inline-error" role="alert">{error}</p>:null}
   {!busy&&!error&&!selected&&!items.length?<p>{t("filmCreation.noResults")}</p>:null}
   {!selected?items.map(item=><button type="button" className="ghost-button film-confirmation__candidate" key={`${item.endpoint}/${item.sceneId}`} disabled={busy||!item.code.trim()} onClick={()=>void choose(item)}><strong>{item.code} · {item.title}</strong><span>{item.date} · {item.studio} · {item.performers.join(" / ")}</span><small>{item.endpoint}</small></button>):<article className="film-confirmation__film">
    {selected.imageUrl?<SceneArtwork src={selected.imageUrl} alt={selected.title}/>:null}<div><h3>{selected.code} · {selected.title}</h3><p>{selected.date} · {selected.studio}</p><p>{selected.performers.join(" / ")}</p><p>{selected.endpoint}</p></div>
    {isResource?<p className="film-confirmation__resource"><strong>{t("filmCreation.resource")}</strong><br/>{resource.title||t("filmCreation.providedLink")}{resource.sourceName?` · ${resource.sourceName}`:""}</p>:null}
    {isResource&&resource.title&&!resource.title.toUpperCase().includes(selected.code.toUpperCase())?<p className="inline-error">{t("filmCreation.resourceMismatch")}</p>:null}
    <label><input type="checkbox" checked={acknowledged} disabled={busy} onChange={e=>setAcknowledged(e.target.checked)}/>{t(isResource?"filmCreation.confirmResource":"filmCreation.confirmFilm")}</label>
    <div className="inline-actions"><button type="button" disabled={busy||!acknowledged} onClick={()=>void confirm()}>{t("filmCreation.create")}</button><button type="button" className="ghost-button" disabled={busy} onClick={()=>{setSelected(null);setAcknowledged(false)}}>{t("filmCreation.chooseAnother")}</button></div>
   </article>}
  </div>
 </Drawer>
}
