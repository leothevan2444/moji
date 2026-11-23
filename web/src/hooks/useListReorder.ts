import { useRef, useState, type HTMLAttributes } from "react";

// Shared desktop drag, touch/pen capture and keyboard move semantics.
export function useListReorder<T extends HTMLElement>(order: string[], onMove: (from: number, to: number) => void, disabled = false,
  { attribute = "data-reorder-id", scrollSelector = ".drawer-body", horizontal = false }: { attribute?: string; scrollSelector?: string; horizontal?: boolean } = {}) {
  const list = useRef<T>(null);
  const touch = useRef<{ id: string; target: string } | null>(null);
  const [dragged, setDragged] = useState<string | null>(null);
  const [over, setOver] = useState<string | null>(null);
  const reset = () => { touch.current = null; setDragged(null); setOver(null); };
  const move = (from: string, to: string) => { const start = order.indexOf(from), end = order.indexOf(to); if (!disabled && start >= 0 && end >= 0 && start !== end) onMove(start, end); reset(); };
  const row = (id: string): HTMLAttributes<HTMLElement> => ({
    [attribute]: id,
    onDragOver: event => { if (!dragged || disabled) return; event.preventDefault(); event.dataTransfer.dropEffect = "move"; setOver(id); },
    onDrop: event => { event.preventDefault(); if (dragged) move(dragged, id); }, onDragEnd: reset
  });
  const handle = (id: string): HTMLAttributes<HTMLElement> => ({
    draggable: !disabled,
    onKeyDown: event => {
      let direction = event.key === "ArrowUp" ? -1 : event.key === "ArrowDown" ? 1 : 0;
      if (horizontal && (event.key === "ArrowLeft" || event.key === "ArrowRight")) {
        const rtl = getComputedStyle(event.currentTarget).direction === "rtl";
        direction = (event.key === "ArrowRight" ? 1 : -1) * (rtl ? -1 : 1);
      }
      if (disabled || !direction) return;
      event.preventDefault();
      const index = order.indexOf(id);
      const target = order[index + direction];
      if (target) move(id, target);
    },
    onDragStart: event => { if (disabled) { event.preventDefault(); return; } event.dataTransfer.effectAllowed = "move"; event.dataTransfer.setData("text/plain", id); setDragged(id); },
    onDragEnd: reset,
    onPointerDown: event => { if (disabled || event.pointerType === "mouse") return; event.preventDefault(); event.currentTarget.setPointerCapture(event.pointerId); touch.current = { id, target: id }; setDragged(id); },
    onPointerMove: event => {
      if (!touch.current || disabled) return;
      const target = document.elementFromPoint(event.clientX, event.clientY)?.closest<HTMLElement>(`[${attribute}]`);
      if (target && list.current?.contains(target)) { touch.current.target = target.getAttribute(attribute)!; setOver(touch.current.target); }
      const scroller = list.current?.closest(scrollSelector); const bounds = scroller?.getBoundingClientRect();
      if (bounds && event.clientY < bounds.top + 48) scroller?.scrollBy(0, -12);
      if (bounds && event.clientY > bounds.bottom - 48) scroller?.scrollBy(0, 12);
    },
    onPointerUp: () => { if (touch.current) move(touch.current.id, touch.current.target); },
    // Native mouse dragstart emits pointercancel; it must not clear the drag.
    onPointerCancel: () => { if (touch.current) reset(); }
  });
  return { list, dragged, over, row, handle };
}
