import { useEffect, useState } from "react";
import { Link } from "react-router";
import { Provider, useMutation, useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faRotate } from "@fortawesome/free-solid-svg-icons/faRotate";
import { settingsClient } from "../../graphql/client";
import { ControlTorrentDocumentDocument, DownloaderDocumentDocument, RefreshDownloaderDocumentDocument, TorrentAction, type DownloaderDocumentQuery } from "../../graphql/generated/graphql";
import { formatBytesRate, formatDurationSeconds } from "../../utils/formatters";
import { describeQueryError } from "../../services/queryError";
import { TorrentTable } from "./TorrentTable";
import { TorrentDetails, TorrentNotice } from "./TorrentDetails";
import "../../styles/downloader.scss";

type Overview = DownloaderDocumentQuery["downloader"];
type Torrent = Overview["torrents"][number];

interface DownloadManagerProps { taskId?: string; onCountChange?: (count: number) => void }
export function DownloadManager({ taskId, onCountChange }: DownloadManagerProps) {
  return <Provider value={settingsClient}><Manager taskId={taskId} onCountChange={onCountChange} /></Provider>;
}

function Manager({ taskId, onCountChange }: DownloadManagerProps) {
  const { t } = useTranslation();
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [attentionOnly, setAttentionOnly] = useState(false);
  const [pending, setPending] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [{ data, error, fetching }, requery] = useQuery({ query: DownloaderDocumentDocument, variables: { offset, limit: 30, search, taskId, attentionOnly }, requestPolicy: "network-only" });
  const [, refreshServer] = useMutation(RefreshDownloaderDocumentDocument);
  const [, control] = useMutation(ControlTorrentDocumentDocument);
  useEffect(() => {
    const timer = window.setInterval(() => { if (!document.hidden) requery({ requestPolicy: "network-only" }); }, 3000);
    const onVisible = () => { if (!document.hidden) requery({ requestPolicy: "network-only" }); };
    document.addEventListener("visibilitychange", onVisible);
    return () => { window.clearInterval(timer); document.removeEventListener("visibilitychange", onVisible); };
  }, [requery]);
  const snapshot = data?.downloader;
  const torrents = snapshot?.torrents.filter(torrent => taskId || torrent.present) ?? [];
  useEffect(() => { if (!fetching && snapshot && offset > 0 && offset >= snapshot.total) setOffset(Math.max(0, Math.floor((snapshot.total - 1) / 30) * 30)); }, [fetching, snapshot, offset]);
  useEffect(() => { if (!taskId && snapshot) onCountChange?.(snapshot.total); }, [onCountChange, snapshot?.total, taskId]);
  const disabled = Boolean(pending) || !snapshot || snapshot.stale;
  const refresh = async () => {
    setPending("refresh"); setFeedback(null);
    try {
      const result = await refreshServer({});
      if (result.error) setFeedback(describeQueryError(result.error));
      requery({ requestPolicy: "network-only" });
    } finally { setPending(null); }
  };
  const action = async (hash: string, next: TorrentAction) => {
    setPending(hash); setFeedback(null);
    try {
      const result = await control({ hash, action: next });
      if (result.error) setFeedback(describeQueryError(result.error));
      requery({ requestPolicy: "network-only" });
    } finally { setPending(null); }
  };
  const refreshButton = <button type="button" className="icon-button" title={t("downloader.refresh")} aria-label={t("downloader.refresh")} aria-busy={pending === "refresh"} disabled={Boolean(pending) || fetching} onClick={() => void refresh()}><FontAwesomeIcon icon={faRotate} aria-hidden="true" /></button>;
  return <section className="download-manager" aria-label={t("downloader.title")}>
    {taskId ? <div className="download-manager__toolbar"><h3>{t("downloader.taskTitle")}</h3>{refreshButton}</div> : <div className="download-manager__toolbar download-manager__filters">
      <input type="search" aria-label={t("downloader.search")} placeholder={t("downloader.search")} value={search} onChange={event => { setSearch(event.target.value); setOffset(0); }} />
      <label><input type="checkbox" checked={attentionOnly} onChange={event => { setAttentionOnly(event.target.checked); setOffset(0); }} />{t("downloader.attention", { count: snapshot?.attentionCount ?? 0 })}</label>
      {refreshButton}
    </div>}
    {error ? <p role="alert" className="tone-danger">{describeQueryError(error)}</p> : null}
    {feedback ? <p role="alert" className="tone-danger">{feedback}</p> : null}
    {snapshot?.stale ? <p className="tone-danger" role="status">{t("downloader.stale")}</p> : null}
    {!snapshot ? <p role="status">{t("common.loading")}</p> : <>
      <div className="download-manager__list">
        {torrents.length ? taskId ? torrents.map(torrent => <TorrentCard key={torrent.hash} torrent={torrent} disabled={disabled} pending={pending === torrent.hash} onAction={action} />) : <TorrentTable torrents={torrents} disabled={disabled} pending={pending} onAction={action} /> : <p>{t("downloader.empty")}</p>}
      </div>
      {snapshot.total > 30 ? <div className="download-manager__toolbar">
        <button type="button" disabled={fetching || offset === 0} onClick={() => setOffset(Math.max(0, offset - 30))}>{t("downloader.previous")}</button>
        <button type="button" disabled={fetching || offset + 30 >= snapshot.total} onClick={() => setOffset(offset + 30)}>{t("downloader.next")}</button>
      </div> : null}
    </>}
  </section>;
}
function Metric({ label, value }: { label: string; value: string }) { return <div><span className="muted">{label}</span><strong>{value}</strong></div>; }
function TorrentCard({ torrent: torrent, disabled, pending, onAction }: { torrent: Torrent; disabled: boolean; pending: boolean; onAction: (hash: string, action: TorrentAction) => Promise<void> }) {
  const { t } = useTranslation();
  const [confirm, setConfirm] = useState<TorrentAction | null>(null);
  const [details, setDetails] = useState(false);
  const paused = /^(paused|stopped)/u.test(torrent.state);
  const problem = torrent.needsAttention;
  const run = (next: TorrentAction) => { setConfirm(null); void onAction(torrent.hash, next); };
  return <article className={`drawer-card download-torrent ${problem ? "download-torrent--attention" : ""}`}>
    <div className="drawer-card__head download-manager__toolbar"><h3>{torrent.name || torrent.hash}</h3><span className={`status-chip ${problem ? "tone-danger" : "tone-neutral"}`}>{t(`downloader.states.${torrent.state}`, { defaultValue: torrent.state })}</span></div>
    <p>{torrent.taskIds.length ? torrent.taskIds.map((id, index) => <Link key={id} to={`/tasks/${encodeURIComponent(id)}`}>{torrent.taskCodes[index] || id} </Link>) : t("downloader.unmanaged")}</p>
    <TorrentNotice torrent={torrent} />
    <div className="home-progress" role="progressbar" aria-label={t("downloader.progress")} aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(torrent.progress * 100)}><span style={{ width: `${torrent.progress * 100}%` }} /></div>
    <div className="download-metrics">
      <Metric label={t("downloader.progress")} value={`${(torrent.progress * 100).toFixed(1)}%`} />
      <Metric label={t("downloader.download")} value={formatBytesRate(torrent.downloadSpeed)} />
      <Metric label={t("downloader.upload")} value={formatBytesRate(torrent.uploadSpeed)} />
      <Metric label={t("downloader.ratio")} value={torrent.ratio.toFixed(2)} />
      <Metric label={t("downloader.seedTime")} value={t("downloader.hours", { value: (torrent.seedingSeconds / 3600).toFixed(1) })} />
      <Metric label={t("downloader.eta")} value={torrent.eta > 0 && torrent.eta < 8640000 ? formatDurationSeconds(torrent.eta) : t("downloader.unknown")} />
    </div>
    <div className="download-manager__toolbar">
      <button type="button" className="ghost-button" disabled={disabled || !torrent.present} onClick={() => run(paused ? TorrentAction.Resume : TorrentAction.Pause)}>{t(paused ? "downloader.resume" : "downloader.pause")}</button>
      <button type="button" className="ghost-button" disabled={disabled || !torrent.present} onClick={() => run(TorrentAction.Reannounce)}>{t("downloader.reannounce")}</button>
      <button type="button" className="ghost-button" disabled={disabled || !torrent.present} onClick={() => setConfirm(TorrentAction.Recheck)}>{t("downloader.recheck")}</button>
      <button type="button" className="ghost-button" disabled={disabled || !torrent.present || (torrent.taskIds.length > 0 && !torrent.allCompleted)} onClick={() => setConfirm(TorrentAction.Remove)}>{t("downloader.remove")}</button>
      <button type="button" className="ghost-button" disabled={disabled || !torrent.present || !torrent.canDeleteFiles} onClick={() => setConfirm(TorrentAction.RemoveFiles)}>{t("downloader.removeFiles")}</button>
      <button type="button" className="ghost-button" aria-expanded={details} onClick={() => setDetails(!details)}>{t("downloader.details")}</button>
      {pending ? <span role="status">{t("common.loading")}</span> : null}
    </div>
    {confirm ? <div className="image-cache-confirm" role="group" aria-label={t("downloader.confirmTitle")}><p>{t(confirm === TorrentAction.Recheck ? "downloader.confirmRecheck" : confirm === TorrentAction.RemoveFiles ? "downloader.confirmRemoveFiles" : "downloader.confirmRemove")}</p><button type="button" className="ghost-button" disabled={disabled} onClick={() => run(confirm)}>{t("downloader.confirm")}</button><button type="button" onClick={() => setConfirm(null)}>{t("downloader.cancel")}</button></div> : null}
    {details ? <TorrentDetails torrent={torrent} showNotice={false} /> : null}
  </article>;
}
