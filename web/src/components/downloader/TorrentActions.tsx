import { useEffect, useId, useRef, useState, type KeyboardEvent } from "react";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faEllipsisVertical, faPause, faPlay } from "@fortawesome/free-solid-svg-icons";
import { TorrentAction } from "../../graphql/generated/graphql";
import { useClickOutside } from "../../hooks/useClickOutside";
import type { Torrent } from "./TorrentDetails";
type OnAction = (hash: string, action: TorrentAction) => Promise<void>;

/** Explain the same backend-owned eligibility at the menu and confirmation. */
export function torrentActionUnavailableReason(torrent:Torrent,action:TorrentAction,unavailable:boolean):string|undefined {
  if (unavailable) return "torrentCard.actionUnavailable";
  if (!torrent.present) return "torrentCard.actionAbsent";
  if ([TorrentAction.Remove,TorrentAction.RemoveFiles].includes(action) && torrent.taskIds.length>0 && !torrent.allCompleted) return "torrentCard.actionIngestActive";
  if (action===TorrentAction.RemoveFiles && !torrent.canDeleteFiles) return "torrentCard.actionFilesUnsafe";
  return undefined;
}

export function TorrentActions({ torrent, disabled, pending, onAction, onConfirm, onDetails, flat=false }: { torrent: Torrent; disabled: boolean; pending: boolean; onAction: OnAction; onConfirm: (value: TorrentAction) => void; onDetails?: () => void; flat?:boolean }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuId = useId();
  useClickOutside([rootRef], () => setOpen(false), open);
  useEffect(() => { if (open) { menuRef.current?.scrollIntoView?.({ block: "nearest" }); menuRef.current?.querySelector<HTMLButtonElement>("button:not(:disabled)")?.focus(); } }, [open]);
  const paused = /^(paused|stopped)/u.test(torrent.state);
  const primaryLabel = t(paused ? "downloader.resume" : "downloader.pause");
  const removeReason=torrentActionUnavailableReason(torrent,TorrentAction.Remove,disabled);
  const removeFilesReason=torrentActionUnavailableReason(torrent,TorrentAction.RemoveFiles,disabled);
  const reasons=[...new Set([removeReason,removeFilesReason].filter((reason):reason is string=>Boolean(reason)))];
  const close = () => { setOpen(false); triggerRef.current?.focus(); };
  const run = (callback: () => void) => { close(); callback(); };
  const keyboard = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!open) return;
    if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); close(); return; }
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
    event.preventDefault(); event.stopPropagation();
    const items = Array.from(menuRef.current?.querySelectorAll<HTMLButtonElement>("button:not(:disabled)") ?? []);
    if (!items.length) return;
    const index = items.indexOf(document.activeElement as HTMLButtonElement);
    const next = event.key === "Home" ? 0 : event.key === "End" ? items.length - 1 : (index + (event.key === "ArrowUp" ? -1 : 1) + items.length) % items.length;
    items[next]?.focus();
  };
  if (flat) return <div ref={rootRef} className="torrent-actions torrent-actions--flat">
    <button type="button" className="ghost-button" data-torrent-action={paused?TorrentAction.Resume:TorrentAction.Pause} disabled={disabled||!torrent.present} aria-busy={pending} onClick={()=>void onAction(torrent.hash,paused?TorrentAction.Resume:TorrentAction.Pause)}>{primaryLabel}</button>
    {([{action:TorrentAction.Reannounce,key:"reannounce"},{action:TorrentAction.Recheck,key:"recheck"},{action:TorrentAction.Remove,key:"remove"},{action:TorrentAction.RemoveFiles,key:"removeFiles"}]).map(item=>{
      const reason=torrentActionUnavailableReason(torrent,item.action,disabled);
      return <button key={item.action} type="button" className="ghost-button" data-torrent-action={item.action} disabled={Boolean(reason)} title={reason?t(reason):undefined} onClick={()=>item.action===TorrentAction.Reannounce?void onAction(torrent.hash,item.action):onConfirm(item.action)}>{t(`downloader.${item.key}`)}</button>;
    })}
  </div>;
  return <div ref={rootRef} className="torrent-actions" onKeyDown={keyboard}>
    <button type="button" className="icon-button" title={primaryLabel} aria-label={primaryLabel} aria-busy={pending} disabled={disabled || !torrent.present} onClick={() => void onAction(torrent.hash, paused ? TorrentAction.Resume : TorrentAction.Pause)}><FontAwesomeIcon icon={paused ? faPlay : faPause} aria-hidden="true" /></button>
    <button ref={triggerRef} type="button" className="icon-button" title={t("downloader.moreActions")} aria-label={t("downloader.moreActions")} aria-haspopup="menu" aria-controls={menuId} aria-expanded={open} onClick={() => setOpen(!open)}><FontAwesomeIcon icon={faEllipsisVertical} aria-hidden="true" /></button>
    {open ? <div ref={menuRef} id={menuId} className="torrent-action-menu" role="menu" aria-label={t("downloader.moreActions")}>
      {onDetails ? <button type="button" role="menuitem" onClick={() => run(onDetails)}>{t("downloader.details")}</button> : null}
      <button type="button" role="menuitem" disabled={disabled || !torrent.present} onClick={() => run(() => void onAction(torrent.hash, TorrentAction.Reannounce))}>{t("downloader.reannounce")}</button>
      <button type="button" role="menuitem" disabled={disabled || !torrent.present} onClick={() => run(() => onConfirm(TorrentAction.Recheck))}>{t("downloader.recheck")}</button>
      <button type="button" role="menuitem" title={removeReason?t(removeReason):undefined} disabled={Boolean(removeReason)} onClick={() => run(() => onConfirm(TorrentAction.Remove))}>{t("downloader.remove")}</button>
      <button type="button" role="menuitem" title={removeFilesReason?t(removeFilesReason):undefined} disabled={Boolean(removeFilesReason)} onClick={() => run(() => onConfirm(TorrentAction.RemoveFiles))}>{t("downloader.removeFiles")}</button>
      {reasons.length?<p className="torrent-action-menu__hint">{reasons.map(reason=>t(reason)).join(" ")}</p>:null}
    </div> : null}
  </div>;
}
