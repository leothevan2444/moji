import { useState, type FormEvent } from "react";
import { createPortal } from "react-dom";
import { useBeforeUnload, useBlocker } from "react-router";
import { useTranslation } from "react-i18next";
import { Drawer } from "../layout/Drawer";
import { SecretInput } from "./SettingsTabCommon";
import { describeQueryError } from "../../services/queryError";
import type { IndexerNetworkProfilePatchInput } from "../../graphql/generated/graphql";

export function IndexerNetworkProfileEditor({ profile, revision, busy, error, onSave, onClose }: {
  profile: { id: string; proxyConfigured: boolean; flareSolverrConfigured: boolean } | null; revision: string; busy: boolean; error: unknown;
  onSave: (id: string, revision: string, create: boolean, patch: IndexerNetworkProfilePatchInput) => Promise<boolean>; onClose: () => void;
}) {
  const { t } = useTranslation();
  const [id, setID] = useState(profile?.id ?? ""); const [proxy, setProxy] = useState(""); const [clear, setClear] = useState(false); const [discard, setDiscard] = useState(false);
  const [solver, setSolver] = useState(""); const [clearSolver, setClearSolver] = useState(false);
  const dirty = solver !== "" || clearSolver || proxy !== "" || clear || id !== (profile?.id ?? "");
  const blocker = useBlocker(dirty || busy);
  useBeforeUnload(event => { if (dirty || busy) { event.preventDefault(); event.returnValue = ""; } });
  const keep = () => { setDiscard(false); if (blocker.state === "blocked") blocker.reset(); };
  const close = () => { if (busy) return; if (dirty) setDiscard(true); else onClose(); };
  const submit = async (event: FormEvent) => {
    event.preventDefault(); if (busy) return;
    const patch: IndexerNetworkProfilePatchInput = clear ? { clearProxy: true } : proxy ? { proxyUrl: proxy } : {};
    if (clearSolver) patch.clearFlareSolverr = true; else if (solver) patch.flareSolverrUrl = solver;
    if (await onSave(id, revision, !profile, patch)) onClose();
  };
  return createPortal(<>
    <Drawer visibleDrawer="settings" closing={false} title={t(profile ? "indexerNetwork.edit" : "indexerNetwork.add")} onClose={close} closeDisabled={busy} footer={<button type="submit" form="indexer-network-profile" className="primary-button" disabled={busy || !dirty || !profile && (!id || (!proxy && !solver))}>{t(busy ? "indexers.saving" : "indexers.save")}</button>}>
      <form id="indexer-network-profile" className="settings-form" onSubmit={event => void submit(event)}>
        {error ? <p role="alert" className="settings-feedback tone-danger">{describeQueryError(error)}</p> : null}
        <label className="settings-field"><span>{t("indexerNetwork.name")}</span><input value={id} disabled={busy || Boolean(profile)} required maxLength={128} pattern={"[A-Za-z0-9][A-Za-z0-9_\\-]{0,127}"} onChange={event => setID(event.target.value)} /></label>
        {!profile ? <p className="settings-note">{t("indexerNetwork.nameHint")}</p> : null}
        <div className="settings-field"><label htmlFor="indexer-proxy-url">{t("indexerNetwork.proxy")}</label><SecretInput id="indexer-proxy-url" value={proxy} disabled={busy || clear} onChange={setProxy} placeholder={profile?.proxyConfigured && !clear ? t("indexers.secretSaved") : ""} /></div>
        <p className="settings-note">{t("indexerNetwork.proxyHint")}</p>
        {profile?.proxyConfigured ? <label className="settings-toggle"><input type="checkbox" checked={clear} disabled={busy} onChange={event => setClear(event.target.checked)} /><span>{t("indexerNetwork.clearProxy")}</span></label> : null}
        <div className="settings-field"><label htmlFor="indexer-solver-url">{t("indexerNetwork.solver")}</label><SecretInput id="indexer-solver-url" value={solver} disabled={busy || clearSolver} onChange={setSolver} placeholder={profile?.flareSolverrConfigured && !clearSolver ? t("indexers.secretSaved") : ""} /></div>
        <p className="settings-note">{t("indexerNetwork.solverHint")}</p>
        {profile?.flareSolverrConfigured ? <label className="settings-toggle"><input type="checkbox" checked={clearSolver} disabled={busy} onChange={event => setClearSolver(event.target.checked)} /><span>{t("indexerNetwork.clearSolver")}</span></label> : null}
      </form>
    </Drawer>
    {discard || blocker.state === "blocked" ? <Drawer visibleDrawer="confirm" closing={false} title={t("indexers.discardTitle")} onClose={keep} footer={<button disabled={busy} onClick={() => { if (blocker.state === "blocked") blocker.proceed(); else onClose(); }}>{t("indexers.discard")}</button>}><p>{t("indexers.unsaved")}</p><button disabled={busy} onClick={keep}>{t("indexers.keepEditing")}</button></Drawer> : null}
  </>, document.body);
}
