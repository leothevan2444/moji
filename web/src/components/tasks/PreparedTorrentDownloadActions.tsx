import { useTranslation } from "react-i18next";
import { usePreparedTorrentDownload } from "../../hooks/usePreparedTorrentDownload";
import { describeQueryError } from "../../services/queryError";

export function PreparedTorrentDownloadActions({searchId,candidateId,disabled=false}:{searchId:string;candidateId:string;disabled?:boolean}){
 const {t}=useTranslation();const {prepare,ready,busy,error}=usePreparedTorrentDownload(searchId,candidateId);
 return <span className="prepared-torrent-download">
  {ready ? <a href={ready.magnet ?? ready.url} target={ready.magnet ? undefined : "_blank"} rel="noopener noreferrer" referrerPolicy="no-referrer">{t(ready.magnet ? "sourcingManual.magnet" : "sourcingManual.download")}</a>
   : <button type="button" className="ghost-button" disabled={disabled||busy} aria-busy={busy} onClick={()=>void prepare()}>{t(busy ? "preparedDownload.preparing" : "preparedDownload.prepare")}</button>}
  {error ? <span className="inline-error" role="alert">{describeQueryError(error)}</span> : null}
 </span>;
}
