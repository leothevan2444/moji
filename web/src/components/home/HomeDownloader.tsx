import { useEffect } from "react";
import { Provider, useMutation, useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowRight, faCircleInfo, faRotate } from "@fortawesome/free-solid-svg-icons";
import { settingsClient } from "../../graphql/client";
import { DownloaderSummaryDocumentDocument, RefreshDownloaderDocumentDocument } from "../../graphql/generated/graphql";
import { formatBytes, formatBytesRate } from "../../utils";
import { describeQueryError } from "../../services/queryError";

export function HomeDownloader({ onOpenList }: { onOpenList: () => void }) {
  return <Provider value={settingsClient}><Summary onOpenList={onOpenList} /></Provider>;
}
function Summary({ onOpenList }: { onOpenList: () => void }) {
  const { t } = useTranslation();
  const [{ data, error }, requery] = useQuery({ query: DownloaderSummaryDocumentDocument, requestPolicy: "network-only" });
  const [{ fetching: refreshing, error: refreshError }, refresh] = useMutation(RefreshDownloaderDocumentDocument);
  useEffect(() => {
    const query = () => { if (!document.hidden) requery({ requestPolicy: "network-only" }); };
    const timer = window.setInterval(query, 3000);
    document.addEventListener("visibilitychange", query);
    return () => { window.clearInterval(timer); document.removeEventListener("visibilitychange", query); };
  }, [requery]);
  const snap = data?.downloader;
  const metrics = snap ? [
    ["download", formatBytesRate(snap.downloadSpeed)], ["upload", formatBytesRate(snap.uploadSpeed)],
    ["freeSpace", snap.freeSpace == null ? t("downloader.unknown") : formatBytes(snap.freeSpace)],
    ["completedBytes", formatBytes(snap.completedBytes)], ["remainingBytes", formatBytes(snap.remainingBytes)],
    ["connection", t(`downloader.connections.${snap.connectionStatus || "unknown"}`, { defaultValue: t("downloader.unknown") })]
  ] : [];
  return <section className="home-downloader" aria-labelledby="home-downloader-title">
    <div className="home-panel-heading"><h3 id="home-downloader-title">{t("downloader.title")}</h3><div className="profile-card__icons">
      <button type="button" className="home-inline-button" title={t("downloader.refresh")} aria-label={t("downloader.refresh")} aria-busy={refreshing} disabled={refreshing} onClick={() => void refresh({}).then(() => requery({ requestPolicy: "network-only" }))}><FontAwesomeIcon icon={faRotate} aria-hidden="true" /></button>
    </div></div>
    {error || refreshError ? <div className="home-module-error" role="status"><p>{describeQueryError(error || refreshError)}</p></div> : null}
    {!snap ? <div className="home-loading" role="status" aria-label={t("common.loading")}><div className="skeleton" /><div className="skeleton" /></div> : <>
      {snap.stale ? <p className="home-warning home-downloader-status" role="status">{t("downloader.stale")}</p> : null}
      <dl className="home-downloader-metrics">{metrics.map(([key, value]) => <div key={key}><dt>{t(`downloader.${key}`)}{["freeSpace", "completedBytes"].includes(key) ? <button type="button" className="home-inline-button home-downloader-info" title={t("downloader.spaceHint")} aria-label={`${t(`downloader.${key}`)} · ${t("downloader.details")}`}><FontAwesomeIcon icon={faCircleInfo} aria-hidden="true" /></button> : null}</dt><dd>{value}</dd></div>)}</dl>
      <div className="home-downloader-footer"><div className="home-downloader-status"><span>{t("downloader.total", { count: snap.total })}</span>{snap.attentionCount ? <span className="home-warning">{t("downloader.issuesCount", { count: snap.attentionCount })}</span> : null}</div><button type="button" className="home-more home-inline-button" onClick={onOpenList}>{t("downloader.viewTorrents")}<FontAwesomeIcon icon={faArrowRight} aria-hidden="true" /></button></div>

    </>}
  </section>;
}
