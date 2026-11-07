import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useIndexerNetwork } from "../../hooks/useIndexerNetwork";
import { useIndexerManagement } from "../../hooks/useIndexerManagement";
import { FieldLabel, SettingsCard, SettingsError, SettingsLoading, SettingsSaveStatus, SecretInput } from "./SettingsTabCommon";
import { SettingsSwitch } from "./SettingsSwitch";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { Drawer } from "../layout/Drawer";
import { describeQueryError } from "../../services/queryError";
import type { IndexerNetworkSettingsFieldsFragment, IndexerNetworkPatchInput } from "../../graphql/generated/graphql";
import type { SettingsObject } from "./settingsAutosave";

const draft = (s: IndexerNetworkSettingsFieldsFragment) => ({
 proxyEnabled:s.proxyEnabled,proxyUrl:s.proxyUrl,proxyUsername:s.proxyUsername,proxyPassword:"",clearProxyPassword:false,
 flareSolverrMaxTimeoutMs:String(s.flareSolverrMaxTimeoutMs),flareSolverrEnabled:s.flareSolverrEnabled,flareSolverrUrl:s.flareSolverrUrl,
 requestTimeoutSeconds:String(s.requestTimeoutSeconds),siteTimeoutSeconds:String(s.siteTimeoutSeconds)
});
const numeric=(value:string)=>value.trim()?Number(value):NaN;
const encode=(value:ReturnType<typeof draft>):SettingsObject=>({
 proxyEnabled:value.proxyEnabled,proxyUrl:value.proxyUrl,proxyUsername:value.proxyUsername,
 flareSolverrMaxTimeoutMs:numeric(value.flareSolverrMaxTimeoutMs),flareSolverrEnabled:value.flareSolverrEnabled,flareSolverrUrl:value.flareSolverrUrl,
 requestTimeoutSeconds:numeric(value.requestTimeoutSeconds),siteTimeoutSeconds:numeric(value.siteTimeoutSeconds),
 ...(value.clearProxyPassword?{clearProxyPassword:true}:value.proxyPassword?{proxyPassword:value.proxyPassword}:{})
});
function NetworkForm({ snapshot, network, onSaved }: { snapshot: IndexerNetworkSettingsFieldsFragment; network: ReturnType<typeof useIndexerNetwork>; onSaved:()=>void }) {
 const {t}=useTranslation();const [reloadOpen,setReloadOpen]=useState(false);const [reloading,setReload]=useState(false);const [reloadError,setReloadError]=useState<unknown>(null);
 const persist=async(patch:SettingsObject,revision?:string)=>{
  if(!revision)throw new Error(t("settings.autosave.noAcknowledgement"));
  const input:IndexerNetworkPatchInput={};
  for(const key of ["proxyEnabled","flareSolverrEnabled","clearProxyPassword"] as const)if(key in patch){if(typeof patch[key]!=="boolean")throw new Error(t("settings.autosave.invalid"));input[key]=patch[key];}
  for(const key of ["proxyUrl","proxyUsername","proxyPassword","flareSolverrUrl"] as const)if(key in patch){if(typeof patch[key]!=="string")throw new Error(t("settings.autosave.invalid"));input[key]=patch[key];}
  for(const key of ["flareSolverrMaxTimeoutMs","requestTimeoutSeconds","siteTimeoutSeconds"] as const)if(key in patch){if(typeof patch[key]!=="number"||!Number.isInteger(patch[key]))throw new Error(t("settings.autosave.invalid"));input[key]=patch[key];}
  return (await network.save(revision,input)).revision;
 };
 const form=useSettingsAutosave("indexers:shared-network",draft(snapshot),encode,{persist,sourceRevision:snapshot.revision,onSaved:()=>{network.refresh();onSaved();}});
 const value=form.value;
 const change=<K extends keyof typeof value>(key:K,next:typeof value[K])=>form.setValue(previous=>({...previous,[key]:next}));
 const reload=async()=>{setReload(true);setReloadError(null);try{const latest=await network.load();if(form.discard(draft(latest),latest.revision))setReloadOpen(false);}catch(error){setReloadError(error);}finally{setReload(false);}};
 return <>
  <div className="settings-form indexer-global-network-form">
   <SettingsCard title={t("indexerNetwork.proxy")} actions={<SettingsSwitch checked={value.proxyEnabled} label={t("indexerNetwork.enableProxy")} onChange={next=>change("proxyEnabled",next)}/> }>
    <div className="indexer-global-network-fields">
     <label className="settings-field"><span>{t("indexerNetwork.proxyUrl")}</span><input type="url" value={value.proxyUrl} placeholder="http://127.0.0.1:7890" onChange={e=>change("proxyUrl",e.target.value)}/></label>
     <label className="settings-field"><FieldLabel text={t("indexerNetwork.proxyUsername")} info={t("indexerNetwork.usernameInfo")}/><input autoComplete="off" value={value.proxyUsername} placeholder={t("indexerNetwork.usernamePlaceholder")} onChange={e=>change("proxyUsername",e.target.value)}/></label>
     <label className="settings-field"><FieldLabel text={t("indexerNetwork.proxyPassword")} info={t("indexerNetwork.passwordInfo")}/><SecretInput disabled={value.clearProxyPassword} value={value.proxyPassword} placeholder={t(value.clearProxyPassword?"indexerNetwork.passwordClearing":snapshot.proxyPasswordConfigured?"indexerNetwork.passwordSavedPlaceholder":"indexerNetwork.passwordPlaceholder")} onChange={next=>change("proxyPassword",next)}/></label>
     {snapshot.proxyPasswordConfigured?<button type="button" className="ghost-button" disabled={form.state==="saving"} onClick={()=>form.setValue(v=>({...v,clearProxyPassword:!v.clearProxyPassword,proxyPassword:""}))}>{t(value.clearProxyPassword?"indexerNetwork.keepPassword":"indexerNetwork.clearPassword")}</button>:null}
    </div>
   </SettingsCard>
   <SettingsCard title="FlareSolverr" actions={<SettingsSwitch checked={value.flareSolverrEnabled} label={t("indexerNetwork.enableSolver")} onChange={next=>change("flareSolverrEnabled",next)}/> }>
    <div className="settings-form">
    <label className="settings-field"><span>{t("indexerNetwork.solverUrl")}</span><input type="url" value={value.flareSolverrUrl} placeholder="http://flaresolverr:8191" onChange={e=>change("flareSolverrUrl",e.target.value)}/></label>
    <label className="settings-field"><FieldLabel text={t("indexerNetwork.solverMaxTimeout")} info={t("indexerNetwork.solverMaxTimeoutInfo")}/><input type="number" min={1000} max={300000} step={1000} value={value.flareSolverrMaxTimeoutMs} onChange={e=>change("flareSolverrMaxTimeoutMs",e.target.value)}/></label>
    </div>
   </SettingsCard>
   <SettingsCard title={t("indexerNetwork.timeouts")}>
    <div className="settings-form indexer-timeout-fields">{([["requestTimeoutSeconds","requestTimeout",1,120],["siteTimeoutSeconds","siteTimeout",10,600]] as const).map(([key,label,min,max])=><label className="settings-field" key={key}><FieldLabel text={t(`indexerNetwork.${label}`)} info={t(`indexerNetwork.${label}Info`)}/><input type="number" min={min} max={max} step={1} value={value[key]} onChange={e=>change(key,e.target.value)}/></label>)}</div>
   </SettingsCard>
   <SettingsSaveStatus state={form.state} error={form.error} retry={form.retry}/>
   {network.error?<p role="alert" className="settings-feedback tone-danger">{describeQueryError(network.error)} <button type="button" onClick={network.refresh}>{t("common.retry")}</button></p>:null}
   {form.state==="error"?<button type="button" className="ghost-button" onClick={()=>{setReloadError(null);setReloadOpen(true);}}>{t("indexerCache.reload")}</button>:null}
  </div>
  {reloadOpen?<Drawer visibleDrawer="confirm" closing={false} title={t("indexerCache.reload")} closeDisabled={reloading} onClose={()=>{if(!reloading)setReloadOpen(false);}} footer={<div className="indexer-actions"><button className="primary-button" disabled={reloading||form.state==="saving"} onClick={()=>void reload()}>{t("indexerCache.reload")}</button><button disabled={reloading} onClick={()=>setReloadOpen(false)}>{t("indexers.keepEditing")}</button></div>}><p>{t("indexerCache.discard")}</p>{reloadError?<p role="alert">{describeQueryError(reloadError)}</p>:null}</Drawer>:null}
 </>;
}
export function IndexerNetworkPanel(){
 const {t}=useTranslation();const management=useIndexerManagement();const network=useIndexerNetwork();
 if(!network.snapshot)return network.error?<SettingsError title={t("indexerNetwork.title")} error={network.error} onRetry={network.refresh}/>:<SettingsLoading title={t("indexerNetwork.title")}/>;
 return <NetworkForm snapshot={network.snapshot} network={network} onSaved={management.refresh}/>;
}
