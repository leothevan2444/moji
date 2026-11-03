import {useId,useState} from "react";
import {useTranslation} from "react-i18next";
import {useIndexerDefinitionPolicy} from "../../hooks/useIndexerDefinitionPolicy";
import type {IndexerDefinitionPolicyFieldsFragment,IndexerDefinitionPolicyPatchInput} from "../../graphql/generated/graphql";
import {useSettingsAutosave} from "./useSettingsAutosave";
import {SettingsSaveStatus} from "./SettingsTabCommon";
import {Drawer} from "../layout/Drawer";
import {describeQueryError} from "../../services/queryError";
import type {SettingsObject} from "./settingsAutosave";
const draft=(snapshot:IndexerDefinitionPolicyFieldsFragment)=>({checkIntervalHours:String(snapshot.checkIntervalHours),autoUpdate:snapshot.autoUpdate});
const encode=(value:ReturnType<typeof draft>)=>({checkIntervalHours:value.checkIntervalHours.trim()?Number(value.checkIntervalHours):NaN,autoUpdate:value.autoUpdate});
type Policy=ReturnType<typeof useIndexerDefinitionPolicy>;
export function IndexerDefinitionPolicy(){
 const policy=useIndexerDefinitionPolicy();const {t}=useTranslation();const heading=useId();
 return <section className="indexer-definition-policy" aria-labelledby={heading}><h3 id={heading}>{t("indexerPolicy.title")}</h3>
  {policy.error?<p className="settings-feedback tone-danger" role="alert">{describeQueryError(policy.error)} <button type="button" onClick={policy.refresh}>{t("common.retry")}</button></p>:null}
  {policy.snapshot?<PolicyEditor snapshot={policy.snapshot} policy={policy}/>:policy.loading?<p role="status">{t("common.loading")}</p>:null}
 </section>;
}
function PolicyEditor({snapshot,policy}:{snapshot:IndexerDefinitionPolicyFieldsFragment;policy:Policy}){
 const {t}=useTranslation();const inputId=useId();const [reloadOpen,setReloadOpen]=useState(false);const [busy,setBusy]=useState(false);const [error,setError]=useState<unknown>(null);
 const persist=async(patch:SettingsObject,revision?:string)=>{
  if(!revision)throw new Error(t("settings.autosave.noAcknowledgement"));const input:IndexerDefinitionPolicyPatchInput={};
  if("autoUpdate" in patch){if(typeof patch.autoUpdate!=="boolean")throw new Error(t("settings.autosave.invalid"));input.autoUpdate=patch.autoUpdate}
  if("checkIntervalHours" in patch){const value=patch.checkIntervalHours;if(typeof value!=="number"||!Number.isInteger(value)||value<1||value>168)throw new Error(t("settings.autosave.invalid"));input.checkIntervalHours=value}
  return policy.update(input,revision);
 };
 const form=useSettingsAutosave("indexers:definition-policy",draft(snapshot),encode,{persist,sourceRevision:snapshot.revision,onSaved:policy.refresh});
 const reload=async()=>{setBusy(true);setError(null);try{const current=await policy.reload();if(form.discard(draft(current),current.revision))setReloadOpen(false)}catch(failure){setError(failure)}finally{setBusy(false)}};
 return <>
  <div className="settings-form indexer-cache-settings"><label className="settings-field" htmlFor={inputId}><span>{t("indexerPolicy.interval")}</span><input id={inputId} type="number" min={1} max={168} step={1} value={form.value.checkIntervalHours} onChange={event=>{const value=event.target.value;form.setValue(previous=>({...previous,checkIntervalHours:value}))}}/></label>
   <label className="settings-toggle"><input type="checkbox" checked={form.value.autoUpdate} onChange={event=>{const value=event.target.checked;form.setValue(previous=>({...previous,autoUpdate:value}))}}/><span>{t("indexerPolicy.autoUpdate")}</span></label>
  </div><p className="settings-note">{t("indexerPolicy.scope")}</p>
  <SettingsSaveStatus state={form.state} error={form.error} retry={form.retry}/>
  {form.state==="error"?<button type="button" onClick={()=>{setError(null);setReloadOpen(true)}}>{t("indexerCache.reload")}</button>:null}
  {reloadOpen?<Drawer visibleDrawer="confirm" closing={false} title={t("indexerCache.reload")} closeDisabled={busy} onClose={()=>{if(!busy)setReloadOpen(false)}}><p>{t("indexerCache.discard")}</p>{error?<p role="alert">{describeQueryError(error)}</p>:null}<div className="indexer-actions"><button type="button" disabled={busy||form.state==="saving"} onClick={()=>void reload()}>{t("indexerCache.reload")}</button><button type="button" disabled={busy} onClick={()=>setReloadOpen(false)}>{t("common.close")}</button></div></Drawer>:null}
 </>;
}
