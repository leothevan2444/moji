import { IndexerLoginDialog } from "./IndexerLoginDialog";
import { IndexerTestDialog } from "./IndexerTestDialog";
import { useIndexerReorder } from "../../hooks/useIndexerReorder";
import { useState } from "react";
import { useSearchParams } from "react-router";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faGripVertical } from "@fortawesome/free-solid-svg-icons/faGripVertical";
import { useIndexerManagement, type IndexerSite } from "../../hooks/useIndexerManagement";
import { IndexerSourceKind } from "../../graphql/generated/graphql";
import { IndexerCatalog } from "./IndexerCatalog";
import { IndexerSiteEditor, type SiteEditorDefinition } from "./IndexerSiteEditor";
import { SettingsCard, SettingsError, SettingsLoading } from "./SettingsTabCommon";
import { Drawer } from "../layout/Drawer";
import { describeQueryError } from "../../services/queryError";
import { IndexerObservationView } from "../indexers/IndexerObservationView";

export function IndexerSitesPanel() {
  const { t } = useTranslation();
  const [testing,setTesting]=useState<IndexerSite|null>(null); const management = useIndexerManagement(); const [, setSearch] = useSearchParams();
  const [loginId,setLoginId]=useState<string|null>(null);
  const [adding, setAdding] = useState(false);
  const [editor, setEditor] = useState<{ site?: IndexerSite; definition: SiteEditorDefinition; revision: string } | null>(null);
  const [removing, setRemoving] = useState<IndexerSite | null>(null);
  const {sites,busy}=management;
  const move=(from:number,to:number)=>{if(busy||from<0||from===to||to<0||to>=sites.length)return;const ids=sites.map(site=>site.id);const [id]=ids.splice(from,1);ids.splice(to,0,id);void management.reorder(ids);};
  const reorder=useIndexerReorder<HTMLDivElement>(sites.map(site=>site.id),move,busy);
  if (management.loading) return <SettingsLoading title={t("indexers.sites")} />;
  if (!management.snapshot) return <SettingsError title={t("indexers.sites")} error={management.error} onRetry={management.refresh} />;
  const { snapshot } = management;
  const add = () => { if (snapshot.ruleRevision) setAdding(true); else setSearch(current => { const next = new URLSearchParams(current); next.set("panel", "definitions"); return next; }); };
  return <SettingsCard title={t("indexers.sites")}>
    {management.error ? <p role="alert" className="settings-feedback tone-danger">{describeQueryError(management.error)} <button onClick={management.refresh}>{t("common.retry")}</button></p> : null}
    <div className="indexer-actions"><button className="primary-button" disabled={busy} onClick={add}>{t(snapshot.ruleRevision ? "indexers.addSite" : "indexers.installFirst")}</button>{snapshot.jackettEnabled ? <button disabled={busy} onClick={() => void management.refreshJackett()}>{t("indexers.refreshExternal")}</button> : null}</div>
    {snapshot.catalogErrorCode ? <p className="settings-feedback">{t(`indexerErrors.${snapshot.catalogErrorCode}`, { defaultValue: t("indexers.catalogFailed") })}</p> : null}
    {snapshot.healthErrorCode ? <p role="status" className="settings-feedback">{t("indexerHealth.unavailable")}</p> : null}
    <div ref={reorder.list} className="indexer-list" aria-busy={busy}>
      {!sites.length ? <p>{t("indexers.empty")}</p> : null}
      {sites.map((site, index) => <div className={`indexer-row${reorder.dragged===site.id?" is-dragging":""}${reorder.over===site.id?" is-drop-target":""}`} key={site.id} data-indexer-id={site.id} {...reorder.row(site.id)}>
        <button className="ghost-button indexer-drag" disabled={busy} aria-label={t("indexers.drag", { name: site.name })} {...reorder.handle(site.id)}><FontAwesomeIcon icon={faGripVertical} aria-hidden="true" /></button>
        <div className="indexer-row__content"><strong>{site.name}</strong><span className="settings-note">{t(site.kind === IndexerSourceKind.Builtin ? "indexers.builtin" : "indexers.external")}{site.language ? ` · ${site.language}` : ""}{site.privacy ? ` · ${t(`indexers.privacyKinds.${site.privacy}`, { defaultValue: site.privacy })}` : ""}</span>{/^https?:\/\//i.test(site.siteUrl) ? <a dir="ltr" href={site.siteUrl} target="_blank" rel="noreferrer" draggable={false}>{site.siteUrl}</a> : null}{site.errorCode ? <span className="tone-danger">{t(`indexerErrors.${site.errorCode}`, { defaultValue: t("indexers.needsAttention") })}</span> : null}{site.health.search ? <IndexerObservationView operation="SEARCH" observation={site.health.search} /> : null}{site.health.fetch ? <IndexerObservationView operation="FETCH" observation={site.health.fetch} /> : null}</div>
        <label className="settings-toggle"><input type="checkbox" checked={site.enabled} disabled={busy} aria-label={t("indexers.enableSite", { name: site.name })} onChange={event => void management.setEnabled(site.id, event.target.checked)} /><span>{t("indexers.enable")}</span></label>
        <div className="indexer-row__actions"><button disabled={busy || index === 0} aria-label={t("indexers.moveUp", { name: site.name })} onClick={() => move(index, index - 1)}>↑</button><button disabled={busy || index === sites.length - 1} aria-label={t("indexers.moveDown", { name: site.name })} onClick={() => move(index, index + 1)}>↓</button>
          <button disabled={busy} onClick={()=>setTesting(site)}>{t("indexerCheck.title")}</button>
          {site.kind === IndexerSourceKind.Builtin ? <><button disabled={busy} onClick={()=>setLoginId(site.id)}>{t("indexerLogin.title")}</button><button disabled={busy} onClick={() => setEditor({ site, definition: { id: site.definitionId, name: site.name, links: [site.siteUrl], settings: site.settings }, revision: snapshot.revision })}>{t("indexers.edit")}</button><button disabled={busy} onClick={() => setRemoving(site)}>{t("indexers.remove")}</button></> : null}
        </div>
      </div>)}
    </div>
    {loginId && sites.some(site=>site.id===loginId)?<IndexerLoginDialog input={{instanceId:loginId,expectedRevision:snapshot.revision}} name={sites.find(site=>site.id===loginId)!.name} onClose={()=>setLoginId(null)}/>:null}
    {testing?<IndexerTestDialog target={{instanceId:testing.id,expectedRevision:snapshot.revision}} onClose={()=>setTesting(null)}/>:null}
    {adding ? <IndexerCatalog onClose={() => setAdding(false)} onSelect={definition => { setAdding(false); setEditor({ definition, revision: snapshot.revision }); }} /> : null}
    {editor ? <IndexerSiteEditor key={editor.site?.id ?? editor.definition.id} {...editor} profiles={snapshot.networkProfiles} busy={busy} error={management.error} onSave={management.save} onClose={() => setEditor(null)} /> : null}
    {removing ? <Drawer visibleDrawer="confirm" closing={false} title={t("indexers.remove")} onClose={() => { if (!busy) setRemoving(null); }} footer={<button disabled={busy} onClick={async () => { if (await management.remove(removing.id)) setRemoving(null); }}>{t("indexers.remove")}</button>}><p>{t("indexers.removeHelp", { name: removing.name })}</p>{management.error ? <p role="alert">{describeQueryError(management.error)}</p> : null}</Drawer> : null}
  </SettingsCard>;
}
