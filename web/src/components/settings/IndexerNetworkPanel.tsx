import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faPlus, faGear, faTrashCan, faRotate } from "@fortawesome/free-solid-svg-icons";
import { useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { useIndexerNetwork } from "../../hooks/useIndexerNetwork";
import { useIndexerManagement } from "../../hooks/useIndexerManagement";
import { SettingsCard, SettingsError, SettingsLoading } from "./SettingsTabCommon";
import { IndexerNetworkProfileEditor } from "./IndexerNetworkProfileEditor";
import { IndexerNetworkTimeouts } from "./IndexerNetworkTimeouts";
import { Drawer } from "../layout/Drawer";
import { describeQueryError } from "../../services/queryError";

export function IndexerNetworkPanel() {
  const { t } = useTranslation(); const management = useIndexerManagement(); const network = useIndexerNetwork(management.refresh);
  const [editing, setEditing] = useState<{ revision: string; profile: { id: string; proxyConfigured: boolean; flareSolverrConfigured: boolean } | null } | null>(null);
  const [removing, setRemoving] = useState<{ id: string; revision: string } | null>(null);
  if (!network.snapshot) return network.error ? <SettingsError title={t("indexerNetwork.title")} error={network.error} onRetry={network.refresh} /> : <SettingsLoading title={t("indexerNetwork.title")} />;
  const snapshot = network.snapshot;
  return <SettingsCard title={t("indexerNetwork.title")}>
    <IndexerNetworkTimeouts snapshot={snapshot} onSaved={() => { network.refresh(); management.refresh(); }} />
    {network.error && !editing && !removing ? <p role="alert" className="settings-feedback tone-danger">{describeQueryError(network.error)} <button onClick={network.refresh}>{t("common.retry")}</button></p> : null}
    <div className="indexer-actions"><button className="icon-button" aria-label={t("indexerNetwork.add")} title={t("indexerNetwork.add")} disabled={network.busy} onClick={() => { network.resetError(); setEditing({ revision: snapshot.revision, profile: null }); }}><FontAwesomeIcon icon={faPlus} aria-hidden="true" /></button><button className="icon-button" aria-label={t("common.refresh")} title={t("common.refresh")} onClick={network.refresh}><FontAwesomeIcon icon={faRotate} aria-hidden="true" /></button></div>
    <ul className="indexer-network-profiles">{snapshot.profiles.map(profile => <li key={profile.id}><div><strong>{profile.id === "direct" ? t("indexers.direct") : profile.id}</strong><span>{t(profile.proxyConfigured ? "indexerNetwork.proxyConfigured" : "indexerNetwork.noProxy")}{profile.flareSolverrConfigured ? ` · ${t("indexerNetwork.solverConfigured")}` : ""} · {t("indexerNetwork.usedBy", { count: profile.usedBy })}</span></div>{profile.id !== "direct" ? <div className="indexer-actions"><button className="icon-button" title={t("indexerNetwork.editNamed", { name: profile.id })} disabled={network.busy} aria-label={t("indexerNetwork.editNamed", { name: profile.id })} onClick={() => { network.resetError(); setEditing({ revision: snapshot.revision, profile }); }}><FontAwesomeIcon icon={faGear} aria-hidden="true" /></button><button className="icon-button" disabled={network.busy || profile.usedBy > 0} title={profile.usedBy > 0 ? t("indexerNetwork.inUse") : t("indexerNetwork.removeNamed", { name: profile.id })} aria-label={t("indexerNetwork.removeNamed", { name: profile.id })} onClick={() => { network.resetError(); setRemoving({ id: profile.id, revision: snapshot.revision }); }}><FontAwesomeIcon icon={faTrashCan} aria-hidden="true" /></button></div> : null}</li>)}</ul>
    {editing ? <IndexerNetworkProfileEditor profile={editing.profile} revision={editing.revision} busy={network.busy} error={network.error} onSave={network.save} onClose={() => setEditing(null)} /> : null}
    {removing ? createPortal(<Drawer visibleDrawer="confirm" closing={false} title={t("indexerNetwork.remove")} onClose={() => { if (!network.busy) setRemoving(null); }} closeDisabled={network.busy}><p>{t("indexerNetwork.removeHelp", { name: removing.id })}</p>{network.error ? <p role="alert">{describeQueryError(network.error)}</p> : null}<div className="indexer-actions"><button disabled={network.busy} onClick={() => void network.remove(removing.id, removing.revision).then(ok => { if (ok) setRemoving(null); })}>{t("indexerNetwork.remove")}</button><button disabled={network.busy} onClick={() => setRemoving(null)}>{t("common.close")}</button></div></Drawer>, document.body) : null}
  </SettingsCard>;
}
