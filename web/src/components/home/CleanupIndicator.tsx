import "../../styles/downloader.scss";
import { useState } from "react";
import { createPortal } from "react-dom";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faBroom } from "@fortawesome/free-solid-svg-icons";
import { Tooltip } from "../common/Tooltip";
import { Drawer } from "../layout/Drawer";
import { formatDateTime } from "../../utils";
import type { DownloaderSummaryDocumentQuery } from "../../graphql/generated/graphql";

type Status=DownloaderSummaryDocumentQuery["downloader"]["cleanup"];
export function CleanupIndicator({status,unavailable}:{status:Status;unavailable:boolean}){
 const {t}=useTranslation();
 const [open,setOpen]=useState(false);
 if(!status.enabled&&status.phase==="IDLE")return null;
 const phase=unavailable?"WARNING":status.phase;
 const reason=unavailable?"STATUS_UNKNOWN":status.reason;
 const label=t(`downloader.cleanupUi.phases.${phase}`);
 const detail=reason?t(`downloader.cleanupUi.reasons.${reason}`):t("downloader.cleanupUi.enabled");
 return <>
  <Tooltip className={`home-cleanup home-cleanup--${phase.toLowerCase()}`} label={t("downloader.cleanupUi.label",{state:label})} info={detail} onClick={()=>setOpen(true)}>
   <FontAwesomeIcon icon={faBroom} aria-hidden="true"/>
   {phase!=="IDLE"?<span role="status">{label}</span>:null}
  </Tooltip>
  {open?createPortal(<Drawer visibleDrawer="downloader-cleanup" closing={false} title={t("downloader.cleanupUi.title")} onClose={()=>setOpen(false)}>
   <article className="drawer-card"><div className="drawer-card__head"><h3>{label}</h3></div>
    <p className="settings-note">{detail}</p>
    <dl className="download-details">
     {status.name?<><dt>{t("downloader.cleanupUi.torrent")}</dt><dd>{status.name}</dd></>:null}
     {status.updatedAt?<><dt>{t("downloader.cleanupUi.lastAction")}</dt><dd>{formatDateTime(status.updatedAt)}</dd></>:null}
    </dl>
    <Link to="/settings/downloader?panel=cleanup" onClick={()=>setOpen(false)}>{t("downloader.settings")}</Link>
   </article>
  </Drawer>,document.body):null}
 </>;
}
