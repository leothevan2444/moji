import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery } from "urql";
import { DownloaderTransferPreferencesDocumentDocument, UpdateDownloaderTransferPreferencesDocumentDocument, type DownloaderTransferPreferencesDocumentQuery } from "../../graphql/generated/graphql";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { FieldLabel, SettingsCard, SettingsError, SettingsLoading, SettingsSaveStatus } from "./SettingsTabCommon";
import { SettingsSwitch } from "./SettingsSwitch";
import DownloaderSwitch from "./DownloaderSwitch";

type Preferences = NonNullable<DownloaderTransferPreferencesDocumentQuery["downloaderTransferPreferences"]>;
const numericFields = ["maxActiveDownloads","maxActiveUploads","maxActiveTorrents","slowTorrentDlRateThreshold","slowTorrentUlRateThreshold","slowTorrentInactiveTimer","maxRatio","maxSeedingTime","maxInactiveSeedingTime"] as const;
type NumericField = typeof numericFields[number];
type Draft = Omit<Preferences, NumericField | "__typename"> & Record<NumericField,string>;

export default function QBTransferSettings() {
 const {t}=useTranslation();
 const [{data,error,fetching},refresh]=useQuery({query:DownloaderTransferPreferencesDocumentDocument,requestPolicy:"network-only"});
 if(error)return <SettingsError title={t("downloader.qbTransfer")} error={error} onRetry={()=>refresh({requestPolicy:"network-only"})}/>;
 if(fetching&&!data)return <SettingsLoading title={t("downloader.qbTransfer")}/>;
 return data?.downloaderTransferPreferences ? <TransferForm prefs={data.downloaderTransferPreferences} refresh={()=>refresh({requestPolicy:"network-only"})}/> : null;
}
function TransferForm({prefs,refresh}:{prefs:Preferences;refresh:()=>void}) {
 const {t}=useTranslation();
 const initial=useMemo(()=>{
  const {__typename:_,...rest}=prefs;
  return {...rest,
 maxActiveDownloads:prefs.maxActiveDownloads==null?"":String(prefs.maxActiveDownloads),
 maxActiveUploads:prefs.maxActiveUploads==null?"":String(prefs.maxActiveUploads),
 maxActiveTorrents:prefs.maxActiveTorrents==null?"":String(prefs.maxActiveTorrents),
 slowTorrentDlRateThreshold:prefs.slowTorrentDlRateThreshold==null?"":String(prefs.slowTorrentDlRateThreshold),
 slowTorrentUlRateThreshold:prefs.slowTorrentUlRateThreshold==null?"":String(prefs.slowTorrentUlRateThreshold),
 slowTorrentInactiveTimer:prefs.slowTorrentInactiveTimer==null?"":String(prefs.slowTorrentInactiveTimer),
 maxRatio:prefs.maxRatio==null?"":String(prefs.maxRatio),
 maxSeedingTime:prefs.maxSeedingTime==null?"":String(prefs.maxSeedingTime),
 maxInactiveSeedingTime:prefs.maxInactiveSeedingTime==null?"":String(prefs.maxInactiveSeedingTime)
} satisfies Draft;
 },[prefs]);
 const [,save]=useMutation(UpdateDownloaderTransferPreferencesDocumentDocument);
 const autosave=useSettingsAutosave("downloader:transfer",initial,value=>{
  const encoded={...value} as Record<string,unknown>;
  for(const key of Object.keys(encoded))if(encoded[key]==null)delete encoded[key];
  for(const key of numericFields) {
   if(prefs[key]==null){delete encoded[key];continue;}
   encoded[key]=value[key].trim()?Number(value[key]):NaN;
  }
  return encoded;
 },{persist:async patch=>{const result=await save({input:patch});if(result.error)throw result.error;},onSaved:refresh});
 const {value:draft,setValue}=autosave;
 const patch=(p:Partial<Draft>)=>setValue(value=>({...value,...p}));
 const number=(key:NumericField,disabled=false,min=0,step="1")=>prefs[key]==null?null:<label className="settings-field"><FieldLabel text={t(`downloader.qbFields.${key}`)}/><input type="number" min={min} max={2147483647} step={step} required disabled={disabled} value={draft[key]} onChange={event=>patch({[key]:event.target.value})}/></label>;
 const toggle=(key:"maxRatioEnabled"|"maxSeedingTimeEnabled"|"maxInactiveSeedingTimeEnabled"|"queueingEnabled"|"dontCountSlowTorrents",disabled=false)=>prefs[key]==null?null:<DownloaderSwitch label={t(`downloader.qbFields.${key}`)} checked={Boolean(draft[key])} disabled={disabled} onChange={value=>{
  const next:Partial<Draft>={[key]:value};
  const field=key==="maxRatioEnabled"?"maxRatio":key==="maxSeedingTimeEnabled"?"maxSeedingTime":key==="maxInactiveSeedingTimeEnabled"?"maxInactiveSeedingTime":null;
  if(value&&field&&Number(draft[field])<0)next[field]=field==="maxRatio"?"1":"1440";
  patch(next);
 }}/>;
 const seedingDisabled=!draft.maxRatioEnabled&&!draft.maxSeedingTimeEnabled&&!draft.maxInactiveSeedingTimeEnabled;
 return <>
  <SettingsCard title={t("downloader.seedingLimits")}><form className="settings-form" onSubmit={event=>event.preventDefault()}>
   <p className="settings-note">{t("downloader.qbSettingsScope")}</p>
   <div className="settings-grid"><div>{toggle("maxRatioEnabled")}{number("maxRatio",!draft.maxRatioEnabled,-1,"0.1")}</div><div>{toggle("maxSeedingTimeEnabled")}{number("maxSeedingTime",!draft.maxSeedingTimeEnabled,-1)}</div><div>{toggle("maxInactiveSeedingTimeEnabled")}{number("maxInactiveSeedingTime",!draft.maxInactiveSeedingTimeEnabled,-1)}</div></div>
   <div className="settings-grid">
    {prefs.shareLimitMode!=null?<label className="settings-field"><FieldLabel text={t("downloader.qbMatch")}/><select disabled={seedingDisabled} value={draft.shareLimitMode??"MatchAny"} onChange={event=>patch({shareLimitMode:event.target.value})}><option value="MatchAny">{t("downloader.qbAny")}</option><option value="MatchAll">{t("downloader.qbAll")}</option></select></label>:null}
    {prefs.maxRatioAction!=null?<label className="settings-field"><FieldLabel text={t("downloader.qbAction")} info={t("downloader.qbActionHint")}/><select disabled={seedingDisabled} value={draft.maxRatioAction??0} onChange={event=>patch({maxRatioAction:Number(event.target.value)})}>{[0,1,3,2].map(action=><option key={action} value={action}>{t(`downloader.qbActions.${action}`)}</option>)}</select></label>:null}
   </div>
   <SettingsSaveStatus {...autosave}/>
  </form></SettingsCard>
  <SettingsCard title={t("downloader.torrentQueueing")} actions={prefs.queueingEnabled==null?undefined:<SettingsSwitch label={t("downloader.qbFields.queueingEnabled")} checked={Boolean(draft.queueingEnabled)} onChange={queueingEnabled=>patch({queueingEnabled})}/>}><form className="settings-form" onSubmit={event=>event.preventDefault()}>
   <div className="settings-grid">{number("maxActiveDownloads",!draft.queueingEnabled,-1)}{number("maxActiveUploads",!draft.queueingEnabled,-1)}{number("maxActiveTorrents",!draft.queueingEnabled,-1)}</div>
   {toggle("dontCountSlowTorrents",!draft.queueingEnabled)}
   <div className="settings-grid">{number("slowTorrentDlRateThreshold",!draft.queueingEnabled||!draft.dontCountSlowTorrents,1)}{number("slowTorrentUlRateThreshold",!draft.queueingEnabled||!draft.dontCountSlowTorrents,1)}{number("slowTorrentInactiveTimer",!draft.queueingEnabled||!draft.dontCountSlowTorrents,1)}</div>
   <SettingsSaveStatus {...autosave}/>
  </form></SettingsCard>
 </>;
}
