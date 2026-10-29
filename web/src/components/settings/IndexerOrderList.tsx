import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faGripVertical } from "@fortawesome/free-solid-svg-icons/faGripVertical";

interface IndexerInfo { id: string; name: string; siteUrl: string; siteType: string; language: string }

export default function IndexerOrderList({ order, indexers, onMove }: {
  order: string[]; indexers: IndexerInfo[]; onMove: (from: number, to: number) => void;
}) {
  const { t } = useTranslation();
  const list = useRef<HTMLOListElement>(null);
  const touch = useRef<{ id: string; target: string } | null>(null);
  const [dragged, setDragged] = useState<string | null>(null);
  const [over, setOver] = useState<string | null>(null);
  const reset = () => { touch.current = null; setDragged(null); setOver(null); };
  const move = (from: string, to: string) => {
    const start = order.indexOf(from), end = order.indexOf(to);
    if (start >= 0 && end >= 0 && start !== end) onMove(start, end);
    reset();
  };
  return <ol ref={list} className="settings-order settings-indexer-order">{order.map((id, index) => {
    const site = indexers.find(item => item.id === id);
    const siteUrl = site?.siteUrl ?? "";
    const urlIsLink = /^https?:\/\//i.test(siteUrl);
    return <li key={id} data-indexer-id={id} draggable
      className={[dragged === id ? "is-dragging" : "", over === id && over !== dragged ? "is-drop-target" : ""].join(" ")}
      onDragStart={event => { event.dataTransfer.effectAllowed = "move"; event.dataTransfer.setData("text/plain", id); setDragged(id); }}
      onDragOver={event => { if (!dragged) return; event.preventDefault(); event.dataTransfer.dropEffect = "move"; setOver(id); }}
      onDrop={event => { event.preventDefault(); if (dragged) move(event.dataTransfer.getData("text/plain"), id); }}
      onDragEnd={reset}>
      <span className="settings-indexer-order__handle" title={t("indexerSettings.drag")} aria-hidden="true"
        onPointerDown={event => {
          if (event.pointerType === "mouse") return;
          event.preventDefault(); event.currentTarget.setPointerCapture(event.pointerId);
          touch.current = { id, target: id }; setDragged(id);
        }}
        onPointerMove={event => {
          if (!touch.current) return;
          const row = document.elementFromPoint(event.clientX, event.clientY)?.closest<HTMLElement>("[data-indexer-id]");
          if (row && list.current?.contains(row)) { touch.current.target = row.dataset.indexerId!; setOver(touch.current.target); }
          const scroller = list.current?.closest(".settings-main");
          const bounds = scroller?.getBoundingClientRect();
          if (bounds && event.clientY < bounds.top + 48) scroller?.scrollBy(0, -12);
          if (bounds && event.clientY > bounds.bottom - 48) scroller?.scrollBy(0, 12);
        }}
        onPointerUp={() => { if (touch.current) move(touch.current.id, touch.current.target); }}
        onPointerCancel={() => { if (touch.current) reset(); }}>
        <FontAwesomeIcon icon={faGripVertical} />
      </span>
      <div className="settings-indexer-order__info">
        <strong>{site?.name ?? id}</strong>
        {siteUrl ? urlIsLink ? <a href={siteUrl} target="_blank" rel="noreferrer" draggable={false}>{siteUrl}</a> : <span>{siteUrl}</span> : null}
        {(site?.siteType || site?.language) ? <small>{[site.siteType, site.language].filter(Boolean).join(" · ")}</small> : null}
      </div>
      <div className="settings-actions">
        <button type="button" disabled={index === 0} onClick={() => onMove(index, index - 1)} aria-label={`${t("settings.moveUp")} ${id}`}>↑</button>
        <button type="button" disabled={index === order.length - 1} onClick={() => onMove(index, index + 1)} aria-label={`${t("settings.moveDown")} ${id}`}>↓</button>
      </div>
    </li>;
  })}</ol>;
}
