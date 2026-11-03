import { createPortal } from "react-dom";
import {useState} from "react";
import {useTranslation} from "react-i18next";
import type {TestIndexerInput} from "../../graphql/generated/graphql";
import {useIndexerCheck} from "../../hooks/useIndexerCheck";
import {describeQueryError} from "../../services/queryError";
import {Drawer} from "../layout/Drawer";
export function IndexerTestDialog({target,onClose}:{target:Pick<TestIndexerInput,"instanceId"|"expectedRevision"|"definitionId"|"draft">;onClose:()=>void}){
 const {t}=useTranslation();const [search,setSearch]=useState(false);const [download,setDownload]=useState(false);const [keywords,setKeywords]=useState("");
 const check=useIndexerCheck({...target,search,download,keywords:search?keywords:undefined});
 const close=()=>{check.cancel();onClose()};
 const observedSearch = check.report?.stages.some(stage => stage.name === "SEARCH" && ["PASSED", "EMPTY", "PARTIAL"].includes(stage.state));
 return createPortal(<Drawer visibleDrawer="indexer-test" closing={false} title={t("indexerCheck.title")} onClose={close} footer={<div className="indexer-actions"><button type="button" className="primary-button" disabled={check.busy||(search&&!keywords.trim())} onClick={()=>void check.run()}>{t(check.busy?"indexerCheck.running":"indexerCheck.run")}</button>{check.busy?<button type="button" onClick={check.cancel}>{t("indexerCheck.cancel")}</button>:null}</div>}>
  <p className="settings-note">{t("indexerCheck.scope")}</p><div className="settings-form">
   <label className="settings-toggle"><input type="checkbox" checked={search} onChange={event=>{setSearch(event.target.checked);if(!event.target.checked)setDownload(false)}}/><span>{t("indexerCheck.search")}</span></label>
   {search?<><label className="settings-field"><span>{t("indexerCheck.keywords")}</span><input value={keywords} maxLength={512} onChange={event=>setKeywords(event.target.value)}/></label><label className="settings-toggle"><input type="checkbox" checked={download} onChange={event=>setDownload(event.target.checked)}/><span>{t("indexerCheck.download")}</span></label>{download?<p className="settings-note">{t("indexerCheck.quota")}</p>:null}</>:null}
  </div>
  {check.busy?<p role="status">{t("indexerCheck.running")}</p>:null}{check.error?<p role="alert" className="settings-feedback tone-danger">{describeQueryError(check.error)}</p>:null}
  {check.report?<section className="indexer-check-result" aria-live="polite" aria-label={t("indexerCheck.result")}><dl>{check.report.stages.map(stage=><div key={stage.name}><dt>{t(`indexerCheck.stages.${stage.name}`)}</dt><dd><span className={stage.state==="FAILED"?"tone-danger":undefined}>{t(`indexerCheck.states.${stage.state}`)}</span>{stage.errorCode?<p className="settings-feedback tone-danger">{t(`indexerErrors.${stage.errorCode}`,{defaultValue:t("indexerCheck.failed")})}</p>:null}</dd></div>)}</dl>
   {observedSearch?<p>{t("indexerCheck.count",{count:check.report.results})}{check.report.truncated?` · ${t("torrentSearch.truncated")}`:""}</p>:null}
   {check.report.kind==="JACKETT"?<p className="settings-note">{t("indexerCheck.jackett")}</p>:null}<details><summary>{t("indexers.details")}</summary><dl><dt>{t("indexerCheck.correlation")}</dt><dd><code>{check.report.id}</code></dd>{check.report.ruleRevision?<><dt>{t("indexers.currentVersion")}</dt><dd><code>{check.report.ruleRevision}</code></dd></>:null}</dl></details>
  </section>:null}
 </Drawer>, document.body);
}
