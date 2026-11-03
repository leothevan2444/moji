import { Provider } from "urql";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowRight, faRotate } from "@fortawesome/free-solid-svg-icons";
import { settingsClient } from "../../graphql/client";
import { useHomeIndexers } from "../../hooks/useHomeIndexers";
import { describeQueryError } from "../../services/queryError";
import { formatDateTime } from "../../utils";
import { IndexerObservationView } from "../indexers/IndexerObservationView";

export function HomeIndexers() {
  return <Provider value={settingsClient}><Summary /></Provider>;
}

function Summary() {
  const { t } = useTranslation();
  const { snapshot, error, fetching, refresh } = useHomeIndexers();
  return <section className="home-panel home-indexers" aria-labelledby="home-indexers-title">
    <div className="home-panel-heading"><h2 id="home-indexers-title">{t("homeIndexers.title")}</h2><button type="button" className="home-inline-button" disabled={fetching} onClick={refresh} aria-label={t("homeIndexers.refresh")}><FontAwesomeIcon icon={faRotate} aria-hidden="true" /></button></div>
    {error ? <div className="home-module-error" role="status"><p>{snapshot ? t("workbench.refreshFailed") : null} {describeQueryError(error)}</p><button type="button" className="ghost-button" onClick={refresh}>{t("common.retry")}</button></div> : null}
    {!snapshot ? !error && <div className="home-loading" role="status" aria-label={t("common.loading")}><div className="skeleton" /></div> : <>
      <ul className="home-service-list home-indexer-sources">
        {snapshot.builtinEnabled > 0 ? <li><span>{t("indexers.builtin")}</span><span>{t("homeIndexers.enabledSites", { count: snapshot.builtinEnabled })}</span></li> : null}
        {snapshot.jackett ? <li><span>Jackett</span><span>{snapshot.jackett.sites == null ? t("homeIndexers.unknownCount") : t("homeIndexers.enabledSites", { count: snapshot.jackett.sites })}</span></li> : null}
      </ul>
      {!snapshot.builtinEnabled && !snapshot.jackett ? <p className="home-empty home-empty--compact">{t("indexers.empty")}</p> : null}
      {snapshot.builtinUnverified > 0 ? <p className="home-muted home-indexer-note">{t("homeIndexers.unverified", { count: snapshot.builtinUnverified })}</p> : null}
      {snapshot.builtinSearchSucceeded > 0 ? <p className="home-muted home-indexer-note">{t("indexerHealth.searchSucceededCount", { count: snapshot.builtinSearchSucceeded })}</p> : null}
      {snapshot.healthErrorCode ? <p role="status" className="home-warning home-indexer-note">{t("indexerHealth.unavailable")}</p> : null}
      {snapshot.jackett?.stale ? <p className="home-warning home-indexer-note">{t("homeIndexers.stale")}{snapshot.jackett.updatedAt ? <> <time dateTime={snapshot.jackett.updatedAt}>{formatDateTime(snapshot.jackett.updatedAt)}</time></> : null}</p> : null}
      {snapshot.jackett?.errorCode ? <p role="status" className="home-warning home-indexer-note">Jackett · {t(`indexerErrors.${snapshot.jackett.errorCode}`, { defaultValue: t("indexers.catalogFailed") })}</p> : null}
      {snapshot.definitionJob ? <div className="home-indexer-definition-job home-indexer-note home-warning" role="status">
        <strong>{t(`indexers.jobs.${snapshot.definitionJob.state}`)}</strong>
        {snapshot.definitionJob.finishedAt ? <time dateTime={snapshot.definitionJob.finishedAt}>{formatDateTime(snapshot.definitionJob.finishedAt)}</time> : null}
        {snapshot.definitionJob.errorCode && !(snapshot.definitionJob.state === "INTERRUPTED" && snapshot.definitionJob.errorCode === "DEFINITION_UPDATE_INTERRUPTED") ? <span>{t(`indexerErrors.${snapshot.definitionJob.errorCode}`, { defaultValue: t("indexers.actionFailed") })}</span> : null}
        <Link to="/settings/indexers?panel=definitions">{t("homeIndexers.viewDefinitions")}</Link>
        {snapshot.definitionJob.correlationId ? <details><summary>{t("indexers.details")}</summary><code>{snapshot.definitionJob.correlationId}</code></details> : null}
      </div> : snapshot.definitionErrorCode ? <div role="status" className="home-warning home-indexer-note"><span>{t("homeIndexers.definitionUnavailable")}</span> <button type="button" className="home-inline-button" disabled={fetching} onClick={refresh}>{t("common.retry")}</button></div> : null}
      {snapshot.issues.length ? <ul className="home-indexer-issues">{snapshot.issues.map(issue => <li key={issue.id}><strong>{issue.name}</strong>{issue.observation ? <IndexerObservationView operation={issue.operation} observation={issue.observation} /> : <span>{t(`indexerErrors.${issue.errorCode}`, { defaultValue: t("indexers.needsAttention") })}</span>}</li>)}</ul> : null}
      {snapshot.attention > snapshot.issues.length + (snapshot.jackett?.errorCode ? 1 : 0) + (snapshot.definitionJob || snapshot.definitionErrorCode ? 1 : 0) ? <p className="home-warning home-indexer-note">{t("homeIndexers.attention", { count: snapshot.attention })}</p> : null}
    </>}
    <Link className="home-more" to="/settings/indexers">{t("homeIndexers.manage")}<FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></Link>
  </section>;
}
