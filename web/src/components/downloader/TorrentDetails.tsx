import { useAppTimezone } from "../../hooks/useAppTimezone";
import { useTranslation } from "react-i18next";
import type { DownloaderDocumentQuery } from "../../graphql/generated/graphql";
import { formatBytes, formatDateTime, formatDurationSeconds } from "../../utils/formatters";

export type Torrent = DownloaderDocumentQuery["downloader"]["torrents"][number];

export function TorrentNotice({ torrent }: { torrent: Torrent }) {
  useAppTimezone();
  const { t } = useTranslation();
  return <>
    {torrent.state === "missing" || torrent.state === "missingFiles" ? <p className={torrent.needsAttention ? "settings-feedback tone-danger" : "home-muted"}>{t(torrent.allCompleted ? "downloader.completedSourceGone" : "downloader.missingHint")}</p> : null}
    {torrent.cleanupError && torrent.needsAttention ? <p role="alert" className="tone-danger">{t("downloader.cleanupError")}</p> : null}
    {torrent.removedAt ? <p>{t("downloader.removed", { time: formatDateTime(torrent.removedAt), reason: t(`downloader.reasons.${torrent.removalReason}`) })}</p> : null}
  </>;
}
export function TorrentDetails({ torrent, showNotice = true, omitIdentity = false }: { torrent: Torrent; showNotice?: boolean; omitIdentity?: boolean }) {
  useAppTimezone();
  const { t } = useTranslation();
  const fields:Array<{key:string;value:string;wide?:boolean;hint?:string}>=[];
  if (!omitIdentity) fields.push({key:"hash",value:torrent.hash,wide:true},{key:"path",value:torrent.savePath,wide:true});
  fields.push(
    {key:"downloaded",value:formatBytes(torrent.downloaded)},
    {key:"uploaded",value:formatBytes(torrent.uploaded)},
    {key:"eta",value:torrent.eta>0&&torrent.eta<8640000?formatDurationSeconds(torrent.eta):t("downloader.unknown")},
    {key:"availability",value:Number.isFinite(torrent.availability)&&torrent.availability>=0?torrent.availability.toFixed(2):t("downloader.unknown"),hint:t("torrentCard.availabilityHint")},
  );
  if (!omitIdentity) fields.push({key:"peers",value:`${torrent.seeds>=0?torrent.seeds:t("downloader.unknown")} (${torrent.seedsTotal??t("downloader.unknown")}) / ${torrent.peers>=0?torrent.peers:t("downloader.unknown")} (${torrent.peersTotal??t("downloader.unknown")})`});
  if (omitIdentity) fields.push({key:"ratio",value:torrent.ratio.toFixed(2)},{key:"seedTime",value:t("downloader.hours",{value:(torrent.seedingSeconds/3600).toFixed(1)})});
  else fields.push({key:"category",value:torrent.category||"—"},{key:"tags",value:torrent.tags||"—"});
  fields.push({key:"activeTime",value:torrent.activeSeconds!=null&&torrent.activeSeconds>=0?(formatDurationSeconds(torrent.activeSeconds)||"0:00"):t("downloader.unknown")},{key:"seenComplete",value:torrent.lastSeenCompleteAt?formatDateTime(torrent.lastSeenCompleteAt):t("downloader.unknown")});
  fields.push({key:"lastSeen",value:formatDateTime(torrent.lastSeenAt),hint:t("torrentCard.lastObservedHint")});
  return <div className="torrent-details">{showNotice?<TorrentNotice torrent={torrent}/>:null}<dl className="download-details">
    {fields.map(field=><div key={field.key} className={field.wide?"download-details__wide":undefined} title={field.hint}><dt>{t(`downloader.${field.key}`)}</dt><dd>{field.value}</dd></div>)}
  </dl></div>;
}
