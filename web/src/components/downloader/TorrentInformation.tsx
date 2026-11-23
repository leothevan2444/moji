import { useTranslation } from "react-i18next";
import { TaskValue } from "../tasks/TaskValue";
import { isMagnetLink, type DashboardTask } from "../../utils";
import { TorrentDetails, type Torrent } from "./TorrentDetails";

import "../../styles/downloader.scss";

export type CopyTorrentValue = (value:string,message:string)=>void|Promise<void>;
interface Props {task?:DashboardTask;torrent?:Torrent;onCopy?:CopyTorrentValue}

export function usableTorrentURL(value?:string|null):string|undefined {
  const candidate=value?.trim();
  if (!candidate) return undefined;
  if (isMagnetLink(candidate)) return candidate;
  try {const url=new URL(candidate);return ["http:","https:"].includes(url.protocol)&&url.hostname?candidate:undefined;} catch {return undefined;}
}

export function TorrentIdentity({task,torrent,onCopy}:Props) {
  const {t}=useTranslation();
  const name=(torrent ? torrent.name : task?.torrentName)?.trim();
  return name ? <dl className="torrent-identity"><TaskValue hideLabel label={t("taskDetail.torrentName")} value={name} onCopy={onCopy} /></dl> : null;
}

/** Task paths are persisted qB observations; no content path is inferred. */
export function TorrentInformation({task,torrent,onCopy}:Props) {
  const {t}=useTranslation();
  const hash=torrent?.hash??task?.torrentHash?.trim();
  const contentPath=task?.contentPath?.trim();
  const location=contentPath || (torrent ? torrent.savePath : task?.savePath)?.trim();
  const tags=torrent ? torrent.tags : task?.tags?.trim();
  const category=torrent ? torrent.category : task?.category?.trim();
  const url=usableTorrentURL(task?.torrentUrl);
  if (!torrent && !hash && !location && !tags && !category && !url) return null;
  return <details className="torrent-information">
    <summary>{t("taskDetail.moreInformation")}</summary>
    <dl className="torrent-info">
      {hash ? <TaskValue label={t("taskDetail.torrentHash")} value={hash} onCopy={onCopy} /> : null}
      {url ? <TaskValue label={t(isMagnetLink(url)?"taskDetail.magnet":"taskExperience.torrentUrl")} value={url} onCopy={onCopy} /> : null}
      {location ? <TaskValue label={t(contentPath?"taskInfo.contentLocation":"taskInfo.saveDirectory")} value={location} onCopy={onCopy} /> : null}
      {category ? <TaskValue label={t("taskDetail.category")} value={category} onCopy={onCopy} /> : null}
      {tags ? <TaskValue label={t("taskDetail.tags")} value={tags} onCopy={onCopy} /> : null}
    </dl>
    {torrent ? <TorrentDetails torrent={torrent} showNotice={false} omitIdentity /> : null}
  </details>;
}
