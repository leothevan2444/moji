import { useListReorder } from "./useListReorder";

export function useIndexerReorder<T extends HTMLElement>(order: string[], onMove: (from: number, to: number) => void, disabled = false) {
  return useListReorder<T>(order, onMove, disabled, { attribute: "data-indexer-id", scrollSelector: ".settings-main" });
}
