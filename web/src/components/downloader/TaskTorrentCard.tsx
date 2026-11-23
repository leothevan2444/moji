import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { TorrentPieces } from "./TorrentPieces";
import { TorrentAction } from "../../graphql/generated/graphql";
import { formatBytes, formatBytesRate, formatDurationSeconds } from "../../utils/formatters";
import { qbittorrentStateLabel } from "../../utils/qbittorrentState";
import type { DashboardTask } from "../../utils";
import { TorrentActions, torrentActionUnavailableReason } from "./TorrentActions";
import { TorrentNotice, type Torrent } from "./TorrentDetails";
import { TorrentIdentity, TorrentInformation, type CopyTorrentValue } from "./TorrentInformation";

function Metric({label,value,hint}:{label:string;value:string;hint?:string}) {
  return <div title={hint}><span className="muted">{label}</span><strong>{value}</strong></div>;
}

export function TaskTorrentCard({torrent,task,onCopy,disabled,pending,onAction,embedded,showState=true,observationAvailable}:{
  torrent:Torrent;task?:DashboardTask;onCopy?:CopyTorrentValue;disabled:boolean;pending:boolean;
  onAction:(hash:string,action:TorrentAction)=>Promise<void>;embedded?:boolean;showState?:boolean;observationAvailable?:boolean;
}) {
  const {t}=useTranslation();
  const [confirm,setConfirm]=useState<TorrentAction|null>(null);
  const actionsRef=useRef<HTMLDivElement>(null);
  const closeConfirm=()=>{setConfirm(null);actionsRef.current?.querySelector<HTMLButtonElement>(`button[data-torrent-action="${confirm??TorrentAction.Recheck}"]`)?.focus();};
  const confirmRef=useRef<HTMLDivElement>(null);
  useEffect(()=>{if(confirm) confirmRef.current?.querySelector<HTMLButtonElement>("button")?.focus();},[confirm]);
  const paused=/^(paused|stopped)/u.test(torrent.state);
  const seeding=!paused && /^(uploading|forcedUP|stalledUP|queuedUP)$/u.test(torrent.state);
  const downloading=!paused && /^(downloading|forcedDL|stalledDL|queuedDL|metaDL|forcedMetaDL|allocating)$/u.test(torrent.state);
  const count=(connected:number,total?:number|null)=>`${connected>=0?connected:t("downloader.unknown")} (${total!=null&&total>=0?total:t("downloader.unknown")})`;
  const size=torrent.size>0 ? `${formatBytes(torrent.completedBytes)} / ${formatBytes(torrent.size)}` : t("downloader.unknown");
  const confirmDisabled=confirm?Boolean(torrentActionUnavailableReason(torrent,confirm,disabled)):disabled;
  return <article className={`${embedded?"download-torrent--embedded":"drawer-card"} download-torrent${torrent.needsAttention?" download-torrent--attention":""}`}>
    {showState ? <div className="download-manager__toolbar"><span className={`status-chip ${torrent.needsAttention?"tone-danger":"tone-neutral"}`}>{qbittorrentStateLabel(torrent.state,t)}</span></div> : null}
    <TorrentIdentity task={task} torrent={torrent} onCopy={onCopy} />
    <TorrentNotice torrent={torrent} />
    <div className="download-metrics">
      {seeding ? <>
        <Metric label={t("downloader.upload")} value={formatBytesRate(torrent.uploadSpeed)} />
        <Metric label={t("downloader.ratio")} value={torrent.ratio.toFixed(2)} />
        <Metric label={t("downloader.seedTime")} value={t("downloader.hours",{value:(torrent.seedingSeconds/3600).toFixed(1)})} />
      </> : downloading ? <>
        <Metric label={t("downloader.download")} value={formatBytesRate(torrent.downloadSpeed)} />
        <Metric label={t("downloader.upload")} value={formatBytesRate(torrent.uploadSpeed)} />
        <Metric label={t("downloader.eta")} value={torrent.eta>0 && torrent.eta<8640000?formatDurationSeconds(torrent.eta):t("downloader.unknown")} />
        <Metric label={t("torrentCard.completedSize")} value={size} />
      </> : <>
        <Metric label={t("torrentCard.completedSize")} value={size} />
        <Metric label={t("downloader.progress")} value={`${Math.max(0,Math.min(100,torrent.progress*100)).toFixed(1)}%`} />
      </>}
      {torrent.present ? <>
        <Metric label={t("torrentCard.seeders")} value={count(torrent.seeds,torrent.seedsTotal)} hint={t("torrentCard.seedersHint")} />
        <Metric label={t("torrentCard.peers")} value={count(torrent.peers,torrent.peersTotal)} hint={t("torrentCard.peersHint")} />
      </>:null}
    </div>
    <TorrentPieces key={torrent.hash} hash={torrent.hash} observedAt={torrent.lastSeenAt} enabled={torrent.present&&(observationAvailable??!disabled)}/>
    <div className="task-torrent-actions" ref={actionsRef}><TorrentActions flat torrent={torrent} disabled={disabled} pending={pending} onAction={onAction} onConfirm={setConfirm} />{pending?<span role="status">{t("common.loading")}</span>:null}</div>
    {confirm ? <div ref={confirmRef} className="image-cache-confirm" role="group" aria-label={t("downloader.confirmTitle")}>
      <p>{t(confirm===TorrentAction.Recheck?"downloader.confirmRecheck":confirm===TorrentAction.RemoveFiles?"downloader.confirmRemoveFiles":"downloader.confirmRemove")}</p>
      <div className="image-cache-confirm__actions"><button type="button" className="ghost-button" disabled={confirmDisabled} onClick={()=>{const action=confirm;closeConfirm();void onAction(torrent.hash,action);}}>{t("downloader.confirm")}</button><button type="button" className="ghost-button" onClick={closeConfirm}>{t("downloader.cancel")}</button></div>
    </div> : null}
    <TorrentInformation task={task} torrent={torrent} onCopy={onCopy} />
  </article>;
}
