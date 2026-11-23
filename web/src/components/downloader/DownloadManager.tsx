import { useEffect, useState } from "react";
import { Provider, useMutation, useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faRotate } from "@fortawesome/free-solid-svg-icons/faRotate";
import { settingsClient } from "../../graphql/client";
import { ControlTorrentDocumentDocument, DownloaderDocumentDocument, RefreshDownloaderDocumentDocument, TorrentAction, type DownloaderDocumentQuery } from "../../graphql/generated/graphql";
import { qbittorrentStateLabel } from "../../utils/qbittorrentState";
import { describeQueryError } from "../../services/queryError";
import { TorrentTable } from "./TorrentTable";
import { TaskTorrentCard } from "./TaskTorrentCard";
import { TorrentIdentity, TorrentInformation, type CopyTorrentValue } from "./TorrentInformation";
import type { DashboardTask } from "../../utils";
import "../../styles/downloader.scss";

type Overview = DownloaderDocumentQuery["downloader"];

interface DownloadManagerProps { embedded?: boolean; taskId?: string; task?:DashboardTask;onCopy?:CopyTorrentValue; onCountChange?: (count: number) => void }
export function DownloadManager(props: DownloadManagerProps) {
  return <Provider value={settingsClient}><Manager key={props.taskId??"global"} {...props} /></Provider>;
}

function Manager({ taskId, onCountChange, embedded, task, onCopy }: DownloadManagerProps) {
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
  const [confirmed,setConfirmed]=useState<Overview|null>(null);
  useEffect(()=>{if(data?.downloader && !error && !fetching) setConfirmed(data.downloader);},[data,error,fetching]);
  const snapshot = (error ? confirmed : data?.downloader) ?? confirmed;
  const torrents = snapshot?.torrents.filter(torrent => taskId || torrent.present) ?? [];
  useEffect(() => { if (!fetching && snapshot && offset > 0 && offset >= snapshot.total) setOffset(Math.max(0, Math.floor((snapshot.total - 1) / 30) * 30)); }, [fetching, snapshot, offset]);
  useEffect(() => { if (!taskId && snapshot) onCountChange?.(snapshot.total); }, [onCountChange, snapshot?.total, taskId]);
  const observationAvailable=Boolean(snapshot && snapshot.connected && !snapshot.stale && !error);
  const disabled = Boolean(pending) || !observationAvailable;
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
  const savedInformation=embedded?<><TorrentIdentity task={task} onCopy={onCopy}/><TorrentInformation task={task} onCopy={onCopy}/></>:null;
  return <section className={`download-manager${embedded ? " download-manager--embedded" : ""}`} aria-label={t(taskId ? "downloader.taskTitle" : "downloader.title")}>
    {taskId ? <div className="download-manager__toolbar task-torrent-heading">{embedded?<h3>{t("fileList.title")}</h3>:<h4>{t("downloader.taskTitle")}</h4>}{embedded && torrents.length===1?<span className={`status-chip ${torrents[0].needsAttention?"tone-danger":"tone-neutral"}`}>{qbittorrentStateLabel(torrents[0].state,t)}</span>:null}{refreshButton}</div> : <div className="download-manager__toolbar download-manager__filters">
      <input type="search" aria-label={t("downloader.search")} placeholder={t("downloader.search")} value={search} onChange={event => { setSearch(event.target.value); setOffset(0); }} />
      <label><input type="checkbox" checked={attentionOnly} onChange={event => { setAttentionOnly(event.target.checked); setOffset(0); }} />{t("downloader.attention", { count: snapshot?.attentionCount ?? 0 })}</label>
      {refreshButton}
    </div>}
    {error ? <p role="alert" className="tone-danger">{describeQueryError(error)}</p> : null}
    {feedback ? <p role="alert" className="tone-danger">{feedback}</p> : null}
    {snapshot && (snapshot.stale || error) ? <p className="tone-danger" role="status">{t("downloader.stale")}</p> : null}
    {!snapshot ? <>{!error?<p role="status">{t("common.loading")}</p>:null}{savedInformation}</> : <>
      <div className="download-manager__list">
        {torrents.length ? taskId ? torrents.map(torrent => <TaskTorrentCard observationAvailable={observationAvailable} key={torrent.hash} torrent={torrent} embedded={embedded} task={task} onCopy={onCopy} showState={!embedded || torrents.length!==1} disabled={disabled} pending={pending === torrent.hash} onAction={action} />) : <TorrentTable torrents={torrents} disabled={disabled} pending={pending} onAction={action} /> : <><p>{t("downloader.empty")}</p>{savedInformation}</>}
      </div>
      {snapshot.total > 30 ? <div className="download-manager__toolbar">
        <button type="button" disabled={fetching || offset === 0} onClick={() => setOffset(Math.max(0, offset - 30))}>{t("downloader.previous")}</button>
        <button type="button" disabled={fetching || offset + 30 >= snapshot.total} onClick={() => setOffset(offset + 30)}>{t("downloader.next")}</button>
      </div> : null}
    </>}
  </section>;
}