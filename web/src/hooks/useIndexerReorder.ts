import { useRef, useState, type HTMLAttributes } from "react";

// Shared desktop drag, touch/pen capture and keyboard move semantics.
export function useIndexerReorder<T extends HTMLElement>(order: string[], onMove: (from: number, to: number) => void, disabled = false) {
  const list = useRef<T>(null);
  const touch = useRef<{ id: string; target: string } | null>(null);
  const [dragged, setDragged] = useState<string | null>(null);
  const [over, setOver] = useState<string | null>(null);
  const reset = () => { touch.current = null; setDragged(null); setOver(null); };
  const move = (from: string, to: string) => { const start = order.indexOf(from), end = order.indexOf(to); if (!disabled && start >= 0 && end >= 0 && start !== end) onMove(start, end); reset(); };
  const row = (id: string): HTMLAttributes<HTMLElement> => ({
    draggable: !disabled,
    onDragStart: event => { if (disabled) return; event.dataTransfer.effectAllowed = "move"; event.dataTransfer.setData("text/plain", id); setDragged(id); },
    onDragOver: event => { if (!dragged || disabled) return; event.preventDefault(); event.dataTransfer.dropEffect = "move"; setOver(id); },
    onDrop: event => { event.preventDefault(); if (dragged) move(dragged, id); }, onDragEnd: reset
  });
  const handle = (id: string): HTMLAttributes<HTMLElement> => ({
    onPointerDown: event => { if (disabled || event.pointerType === "mouse") return; event.preventDefault(); event.currentTarget.setPointerCapture(event.pointerId); touch.current = { id, target: id }; setDragged(id); },
    onPointerMove: event => {
      if (!touch.current || disabled) return;
      const target = document.elementFromPoint(event.clientX, event.clientY)?.closest<HTMLElement>("[data-indexer-id]");
      if (target && list.current?.contains(target)) { touch.current.target = target.dataset.indexerId!; setOver(touch.current.target); }
      const scroller = list.current?.closest(".settings-main"); const bounds = scroller?.getBoundingClientRect();
      if (bounds && event.clientY < bounds.top + 48) scroller?.scrollBy(0, -12);
      if (bounds && event.clientY > bounds.bottom - 48) scroller?.scrollBy(0, 12);
    }, onPointerUp: () => { if (touch.current) move(touch.current.id, touch.current.target); }, onPointerCancel: reset
  });
  return { list, dragged, over, row, handle };
}
