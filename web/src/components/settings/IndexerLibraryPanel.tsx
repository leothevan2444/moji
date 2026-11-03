import { IndexerDefinitionPolicy } from "./IndexerDefinitionPolicy";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useIndexerLibrary } from "../../hooks/useIndexerManagement";
import { Drawer } from "../layout/Drawer";
import { SettingsCard, SettingsError, SettingsLoading } from "./SettingsTabCommon";
import { describeQueryError } from "../../services/queryError";

export function IndexerLibraryPanel() {
  const { t, i18n } = useTranslation();
  const library = useIndexerLibrary();
  const [confirm, setConfirm] = useState<{ activeRevision: string; previousRevision: string } | null>(null);
  if (library.loading) return <SettingsLoading title={t("indexers.library")} />;
  if (!library.status) return <SettingsError title={t("indexers.library")} error={library.error} onRetry={library.refresh} />;
  const { status, job, active, busy, run } = library;
  const manifest = status.installed.find(item => item.revision === status.activeRevision);
  return <SettingsCard title={t("indexers.library")}>
    {library.error ? <p role="alert" className="settings-feedback tone-danger">{describeQueryError(library.error)} <button onClick={library.refresh}>{t("common.retry")}</button></p> : null}
    <p>{status.activeRevision ? status.available ? t("indexers.libraryReady", { count: manifest?.definitionCount ?? "—" }) : t("indexers.libraryDamaged") : t("indexers.libraryMissing")}</p>
    {status.checkErrorCode ? <p className="settings-feedback tone-danger">{t(`indexerErrors.${status.checkErrorCode}`, { defaultValue: t("indexers.checkFailed") })}</p> : null}
    {job ? <div className="indexer-job" role={active ? "status" : undefined} aria-live="polite">
      <strong>{t(`indexers.jobs.${job.state}`)}</strong>
      {job.state === "DOWNLOADING" ? <span>{t("indexers.downloaded", { size: (job.downloadedBytes / 1048576).toFixed(1) })}</span> : null}
      {job.errorCode ? <p className="tone-danger">{t(`indexerErrors.${job.errorCode}`, { defaultValue: t("indexers.actionFailed") })}</p> : null}
      {job.errorCode ? <details><summary>{t("indexers.details")}</summary><code>{job.correlationId}</code></details> : null}
    </div> : null}
    <div className="indexer-actions">
      {!status.activeRevision ? <button className="primary-button" disabled={busy || active} onClick={() => void run("INSTALL")}>{t("indexers.install")}</button> : <>
        <button disabled={busy || active} onClick={() => void run("CHECK")}>{t("indexers.check")}</button>
        {status.availableRevision && status.availableRevision !== status.activeRevision ? <button className="primary-button" disabled={busy || active} onClick={() => void run("UPDATE")}>{t("indexers.update")}</button> : null}
        {status.previousRevision ? <button disabled={busy || active} onClick={() => setConfirm({ activeRevision: status.activeRevision, previousRevision: status.previousRevision })}>{t("indexers.rollback")}</button> : null}
        {!status.available ? <button disabled={busy || active} onClick={() => void run("REPAIR")}>{t("indexers.repair")}</button> : null}
      </>}
      {active ? <button disabled={busy || job?.state === "ACTIVATING"} onClick={() => void run("CANCEL")}>{t("indexers.cancelJob")}</button> : null}
    </div>
    {status.lastCheckedAt ? <p className="settings-note">{t("indexers.lastChecked", { time: new Date(status.lastCheckedAt).toLocaleString(i18n.language) })}</p> : null}
    {status.activeRevision ? <details className="indexer-details"><summary>{t("indexers.details")}</summary><dl><dt>{t("indexers.currentVersion")}</dt><dd><code>{status.activeRevision}</code></dd><dt>{t("indexers.schemaVersion")}</dt><dd>{manifest?.schemaVersion ?? "—"}</dd></dl></details> : null}
    <IndexerDefinitionPolicy/>
    {confirm ? <Drawer visibleDrawer="confirm" closing={false} title={t("indexers.rollback")} onClose={() => setConfirm(null)} footer={<button className="primary-button" onClick={() => { const reviewed = confirm; setConfirm(null); void run("ROLLBACK", reviewed); }}>{t("indexers.rollback")}</button>}><p>{t("indexers.rollbackHelp")}</p><code>{confirm.previousRevision.slice(0, 12)}</code></Drawer> : null}
  </SettingsCard>;
}
