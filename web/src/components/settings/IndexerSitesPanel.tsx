import { IndexerLoginDialog } from "./IndexerLoginDialog";
import { useIndexerConnections } from "../../hooks/useIndexerConnections";
import { IndexerConnectionButton } from "./IndexerConnectionButton";
import { useIndexerReorder } from "../../hooks/useIndexerReorder";
import { useState } from "react";
import { createPortal } from "react-dom";
import { useSearchParams } from "react-router";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faGripVertical, faPlus, faWrench, faTrashCan, faRightToBracket, faRotate } from "@fortawesome/free-solid-svg-icons";
import { SettingsSwitch } from "./SettingsSwitch";
import { IndexerPrivacyBadge } from "../indexers/IndexerPrivacyBadge";
import { useIndexerManagement, type IndexerSite } from "../../hooks/useIndexerManagement";
import { IndexerSourceKind } from "../../graphql/generated/graphql";
import { IndexerCatalog } from "./IndexerCatalog";
import { IndexerSiteEditor, type SiteEditorDefinition } from "./IndexerSiteEditor";
import { SettingsCard, SettingsError, SettingsLoading } from "./SettingsTabCommon";
import { Drawer } from "../layout/Drawer";
import { describeQueryError } from "../../services/queryError";

export function IndexerSitesPanel() {
  const { t } = useTranslation();
  const management = useIndexerManagement(); const [, setSearch] = useSearchParams();
  const [loginId,setLoginId]=useState<string|null>(null);
  const [adding, setAdding] = useState(false);
  const [editor, setEditor] = useState<{ site?: IndexerSite; definition: SiteEditorDefinition; revision: string } | null>(null);
  const [removing, setRemoving] = useState<IndexerSite | null>(null);
  const { sites } = management;
  const checks = useIndexerConnections(management.snapshot?.revision ?? "", "sites", Object.fromEntries(sites.map(site => [site.id, site.connectionRevision])));
  const busy = management.busy;
  const move=(from:number,to:number)=>{if(busy||from<0||from===to||to<0||to>=sites.length)return;const ids=sites.map(site=>site.id);const [id]=ids.splice(from,1);ids.splice(to,0,id);void management.reorder(ids);};
  const reorder=useIndexerReorder<HTMLDivElement>(sites.map(site=>site.id),move,busy);
  if (management.loading) return <SettingsLoading title={t("indexers.sites")} />;
  if (!management.snapshot) return <SettingsError title={t("indexers.sites")} error={management.error} onRetry={management.refresh} />;
  const { snapshot } = management;
  const add = () => { if (snapshot.ruleRevision) setAdding(true); else setSearch(current => { const next = new URLSearchParams(current); next.set("panel", "definitions"); return next; }); };
  return <SettingsCard title={t("indexers.sites")} actions={<>
      <button className="ghost-button" disabled={busy} onClick={add}><FontAwesomeIcon icon={faPlus} aria-hidden="true" /> {t(snapshot.ruleRevision ? "indexers.addSite" : "indexers.installFirst")}</button>
      {snapshot.jackettEnabled ? <button className="ghost-button" disabled={busy} onClick={() => void management.refreshJackett()}><FontAwesomeIcon icon={faRotate} aria-hidden="true" /> {t("indexers.refreshExternal")}</button> : null}
      <IndexerConnectionButton result={checks.all} all disabled={busy || Boolean(checks.all) || !sites.length} onClick={() => void checks.run(sites.map(site => site.id), true)} />
    </>}>
    {management.error ? <p role="alert" className="settings-feedback tone-danger">{describeQueryError(management.error)} <button onClick={management.refresh}>{t("common.retry")}</button></p> : null}

    {snapshot.catalogErrorCode ? <p className="settings-feedback">{t(`indexerErrors.${snapshot.catalogErrorCode}`, { defaultValue: t("indexers.catalogFailed") })}</p> : null}
    {snapshot.healthErrorCode ? <p role="status" className="settings-feedback">{t("indexerHealth.unavailable")}</p> : null}
    <div ref={reorder.list} className="indexer-list" aria-busy={busy}>
      {!sites.length ? <p>{t("indexers.empty")}</p> : null}
      {sites.map(site => <div className={`indexer-row${reorder.dragged===site.id?" is-dragging":""}${reorder.over===site.id?" is-drop-target":""}`} key={site.id} data-indexer-id={site.id} {...reorder.row(site.id)}>
        <button className="ghost-button indexer-drag" aria-disabled={busy} aria-keyshortcuts="ArrowUp ArrowDown" aria-description={t("indexers.dragHint")} title={t("indexers.dragHint")} aria-label={t("indexers.drag", { name: site.name })} {...reorder.handle(site.id)}><FontAwesomeIcon icon={faGripVertical} aria-hidden="true" /></button>
        <div className="indexer-row__content">
          <div className="indexer-row__identity">
            <strong>{/^https?:\/\//i.test(site.siteUrl) ? <a href={site.siteUrl} target="_blank" rel="noreferrer" draggable={false}>{site.name}</a> : site.name}</strong>
            <span className="indexer-language">{site.language}</span><IndexerPrivacyBadge privacy={site.privacy} />
          </div>
          {site.description ? <p className="indexer-description" title={site.description}>{site.description}</p> : null}

        </div>
        <div className="indexer-row__controls">
          <span className="indexer-source">{t(site.kind === IndexerSourceKind.Builtin ? "indexers.builtin" : "indexers.external")}</span>
          <div className="indexer-row__actions">
          <IndexerConnectionButton result={checks.results[site.id]} disabled={busy || checks.isTesting(site.id)} jackett={site.kind === IndexerSourceKind.Jackett} onClick={() => void checks.run([site.id])} />
          {site.kind === IndexerSourceKind.Builtin ? <>{site.loginSupported ? <button className="icon-button" title={t("indexerLogin.title")} aria-label={t("indexerLogin.title")} disabled={busy} onClick={()=>setLoginId(site.id)}><FontAwesomeIcon icon={faRightToBracket} aria-hidden="true" /></button> : null}<button className="icon-button" title={t("indexers.configure")} aria-label={t("indexers.configure")} disabled={busy} onClick={() => setEditor({ site, definition: { id: site.definitionId, name: site.name, links: [site.siteUrl], settings: site.settings }, revision: snapshot.revision })}><FontAwesomeIcon icon={faWrench} aria-hidden="true" /></button><button className="icon-button" title={t("indexers.remove")} aria-label={t("indexers.remove")} disabled={busy} onClick={() => setRemoving(site)}><FontAwesomeIcon icon={faTrashCan} aria-hidden="true" /></button></> : null}
          <SettingsSwitch checked={site.enabled} disabled={busy} label={t("indexers.enableSite", { name: site.name })} onChange={enabled => void management.setEnabled(site.id, enabled)} />
          </div>
        </div>
      </div>)}
    </div>
    {loginId && sites.some(site=>site.id===loginId)?<IndexerLoginDialog input={{instanceId:loginId,expectedRevision:snapshot.revision}} name={sites.find(site=>site.id===loginId)!.name} onClose={()=>setLoginId(null)}/>:null}
    {adding ? <IndexerCatalog sites={sites} busy={busy} mutationError={management.error} onQuickAdd={definition => management.save(null, snapshot.revision, { definitionId: definition.id, name: definition.name, enabled: true })} onClose={() => setAdding(false)} onSelect={definition => { setAdding(false); setEditor({ definition, revision: snapshot.revision }); }} /> : null}
    {editor ? <IndexerSiteEditor connectionPending={Boolean(editor.site && checks.isTesting(editor.site.id))} key={editor.site?.id ?? editor.definition.id} {...editor} profiles={snapshot.networkProfiles} busy={busy} error={management.error} onSave={management.save} onClose={() => setEditor(null)} /> : null}
    {removing ? createPortal(<Drawer visibleDrawer="confirm" closing={false} title={t("indexers.remove")} onClose={() => { if (!busy) setRemoving(null); }} footer={<button disabled={busy} onClick={async () => { if (await management.remove(removing.id)) setRemoving(null); }}>{t("indexers.remove")}</button>}><p>{t("indexers.removeHelp", { name: removing.name })}</p>{management.error ? <p role="alert">{describeQueryError(management.error)}</p> : null}</Drawer>, document.body) : null}
  </SettingsCard>;
}
