import { TorrentActions, torrentActionUnavailableReason } from "./TorrentActions";
import { Fragment, useEffect, useId, useRef, useState } from "react";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { TorrentAction } from "../../graphql/generated/graphql";
import { formatBytesRate } from "../../utils/formatters";
import { qbittorrentStateLabel } from "../../utils/qbittorrentState";
import { TorrentDetails, type Torrent } from "./TorrentDetails";
import "../../styles/settings-shell.scss";

type OnAction = (hash: string, action: TorrentAction) => Promise<void>;
export function TorrentTable({ torrents, disabled, pending, onAction }: { torrents: Torrent[]; disabled: boolean; pending: string | null; onAction: OnAction }) {
  const { t } = useTranslation();
  const columns = ["torrent", "linkedTasks", "stateProgress", "transferSpeeds", "ratio", "seedTime", "actions"];
  return <table className="torrent-table" aria-label={t("downloader.torrentList")}>
    <colgroup>{columns.map(key => <col key={key} className={`torrent-col--${key}`} />)}</colgroup>
    <thead><tr>{columns.map(key => <th scope="col" key={key}>{t(`downloader.${key}`)}</th>)}</tr></thead>
    <tbody>{torrents.map(torrent => <TorrentRow key={torrent.hash} torrent={torrent} disabled={disabled} pending={pending === torrent.hash} onAction={onAction} />)}</tbody>
  </table>;
}
function TorrentRow({ torrent, disabled, pending, onAction }: { torrent: Torrent; disabled: boolean; pending: boolean; onAction: OnAction }) {
  const { t } = useTranslation();
  const detailId = useId();
  const [expanded, setExpanded] = useState(false);
  const [confirm, setConfirm] = useState<TorrentAction | null>(null);
  const confirmRef = useRef<HTMLDivElement>(null);
  const progress = Math.round(Math.max(0, Math.min(1, torrent.progress)) * 100);
  const stateLabel = qbittorrentStateLabel(torrent.state, t);
  useEffect(() => { if (confirm) confirmRef.current?.querySelector<HTMLButtonElement>("button")?.focus(); }, [confirm]);
  return <Fragment>
    <tr className={`torrent-row${torrent.needsAttention ? " torrent-row--attention" : ""}`}>
      <td className="torrent-cell--name" data-label={t("downloader.torrent")}><button type="button" className="home-task-title torrent-name" title={torrent.name || torrent.hash} aria-expanded={expanded} aria-controls={detailId} onClick={() => setExpanded(!expanded)}>{torrent.name || torrent.hash}</button></td>
      <td className="torrent-cell--tasks" data-label={t("downloader.linkedTasks")}>{torrent.taskIds.length ? torrent.taskIds.map((id, index) => <Link key={id} to={`/tasks/${encodeURIComponent(id)}`}>{torrent.taskCodes[index] || id}</Link>) : <span className="home-muted" title={t("downloader.unmanaged")}>—</span>}</td>
      <td className="torrent-cell--state" data-label={t("downloader.stateProgress")}><div className="torrent-state"><span className={torrent.needsAttention ? "tone-danger" : "home-muted"} title={stateLabel}>{stateLabel}</span><span>{progress}%</span></div><div className="home-progress" role="progressbar" aria-label={t("downloader.progress")} aria-valuemin={0} aria-valuemax={100} aria-valuenow={progress}><span style={{width: `${progress}%`}} /></div></td>
      <td className="torrent-cell--speed" data-label={t("downloader.transferSpeeds")}><span title={t("downloader.download")}><span aria-hidden="true">↓ </span><span className="sr-only">{t("downloader.download")} </span>{formatBytesRate(torrent.downloadSpeed)}</span><span title={t("downloader.upload")}><span aria-hidden="true">↑ </span><span className="sr-only">{t("downloader.upload")} </span>{formatBytesRate(torrent.uploadSpeed)}</span></td>
      <td className="torrent-cell--ratio" data-label={t("downloader.ratio")}>{torrent.ratio.toFixed(2)}</td>
      <td className="torrent-cell--seed" data-label={t("downloader.seedTime")}>{t("downloader.hours", {value: (torrent.seedingSeconds / 3600).toFixed(1)})}</td>
      <td className="torrent-cell--actions" data-label={t("downloader.actions")}><TorrentActions torrent={torrent} disabled={disabled} pending={pending} onAction={onAction} onConfirm={setConfirm} onDetails={() => setExpanded(!expanded)} /></td>
    </tr>
    {expanded || confirm ? <tr className="torrent-expanded-row" id={detailId}><td colSpan={7}>
      {confirm ? <div ref={confirmRef} className="image-cache-confirm" role="group" aria-label={t("downloader.confirmTitle")}><p>{t(confirm === TorrentAction.Recheck ? "downloader.confirmRecheck" : confirm === TorrentAction.RemoveFiles ? "downloader.confirmRemoveFiles" : "downloader.confirmRemove")}</p><div className="image-cache-confirm__actions"><button type="button" disabled={Boolean(torrentActionUnavailableReason(torrent,confirm,disabled))} onClick={() => { setConfirm(null); void onAction(torrent.hash, confirm); }}>{t("downloader.confirm")}</button><button type="button" className="ghost-button" onClick={() => setConfirm(null)}>{t("downloader.cancel")}</button></div></div> : null}
      {expanded ? <TorrentDetails torrent={torrent} /> : null}
    </td></tr> : null}
  </Fragment>;
}
