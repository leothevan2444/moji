import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faBinoculars } from "@fortawesome/free-solid-svg-icons/faBinoculars";
import { faDownload } from "@fortawesome/free-solid-svg-icons/faDownload";
import { faRotateLeft } from "@fortawesome/free-solid-svg-icons/faRotateLeft";
import { faWrench } from "@fortawesome/free-solid-svg-icons/faWrench";
import { faXmark } from "@fortawesome/free-solid-svg-icons/faXmark";
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
  return <SettingsCard title={t("indexers.library")} actions={
    <div className="indexer-actions indexer-library-actions">
      {!status.activeRevision ? <button className="primary-button" disabled={busy || active} onClick={() => void run("INSTALL")}><FontAwesomeIcon icon={faDownload} aria-hidden="true" /> {t("indexers.install")}</button> : <>
        <button className="ghost-button" disabled={busy || active} onClick={() => void run("CHECK")}><FontAwesomeIcon icon={faBinoculars} aria-hidden="true" /> {t("indexers.check")}</button>
        {status.availableRevision && status.availableRevision !== status.activeRevision ? <button className="primary-button" disabled={busy || active} onClick={() => void run("UPDATE")}><FontAwesomeIcon icon={faDownload} aria-hidden="true" /> {t("indexers.update")}</button> : null}
        {status.previousRevision ? <button className="ghost-button" disabled={busy || active} onClick={() => setConfirm({ activeRevision: status.activeRevision, previousRevision: status.previousRevision })}><FontAwesomeIcon icon={faRotateLeft} aria-hidden="true" /> {t("indexers.rollback")}</button> : null}
        {!status.available ? <button className="ghost-button" disabled={busy || active} onClick={() => void run("REPAIR")}><FontAwesomeIcon icon={faWrench} aria-hidden="true" /> {t("indexers.repair")}</button> : null}
      </>}
    </div>
  }>
    <div className="settings-form indexer-library-content">
    {library.error ? <p role="alert" className="settings-feedback tone-danger">{describeQueryError(library.error)} <button onClick={library.refresh}>{t("common.retry")}</button></p> : null}
    <div className="indexer-library-summary">
      <p className={`indexer-library-status${status.activeRevision && !status.available ? " tone-danger" : ""}`}>{status.activeRevision ? status.available ? t("indexers.libraryReady", { count: manifest?.definitionCount ?? "—" }) : t("indexers.libraryDamaged") : t("indexers.libraryMissing")}</p>
      {status.lastCheckedAt ? <p className="settings-note">{t("indexers.lastChecked", { time: new Date(status.lastCheckedAt).toLocaleString(i18n.language) })}</p> : null}
    </div>
    {status.checkErrorCode ? <p role="alert" className="settings-feedback tone-danger">{t(`indexerErrors.${status.checkErrorCode}`, { defaultValue: t("indexers.checkFailed") })}</p> : null}
    {job ? <div className="indexer-job" role={active ? "status" : undefined} aria-live="polite">
      <div className="indexer-job__summary"><strong>{t(`indexers.jobs.${job.state}`)}</strong>
        {job.state === "DOWNLOADING" ? <span className="settings-note">{t("indexers.downloaded", { size: (job.downloadedBytes / 1048576).toFixed(1) })}</span> : null}
        {active ? <button type="button" className="ghost-button" disabled={busy || job.state === "ACTIVATING"} onClick={() => void run("CANCEL")}><FontAwesomeIcon icon={faXmark} aria-hidden="true" /> {t("indexers.cancelJob")}</button> : null}
      </div>
      {job.errorCode ? <p className="tone-danger">{t(`indexerErrors.${job.errorCode}`, { defaultValue: t("indexers.actionFailed") })}</p> : null}
      {job.errorCode ? <details><summary>{t("indexers.details")}</summary><code>{job.correlationId}</code></details> : null}
    </div> : null}
    {status.activeRevision ? <details className="indexer-details"><summary>{t("indexers.details")}</summary><dl><dt>{t("indexers.currentVersion")}</dt><dd><code>{status.activeRevision}</code></dd><dt>{t("indexers.schemaVersion")}</dt><dd>{manifest?.schemaVersion ?? "—"}</dd></dl></details> : null}
    <IndexerDefinitionPolicy/>
    </div>
    {confirm ? <Drawer visibleDrawer="confirm" closing={false} title={t("indexers.rollback")} onClose={() => setConfirm(null)} footer={<button className="primary-button" onClick={() => { const reviewed = confirm; setConfirm(null); void run("ROLLBACK", reviewed); }}>{t("indexers.rollback")}</button>}><p>{t("indexers.rollbackHelp")}</p><code>{confirm.previousRevision.slice(0, 12)}</code></Drawer> : null}
  </SettingsCard>;
}
