import { useSyncExternalStore } from "react";

let generation = 0;
const listeners = new Set<() => void>();

export function notifyTaskSnapshotRecovery() {
  generation += 1;
  listeners.forEach((listener) => listener());
}

export function subscribeTaskSnapshotRecovery(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function getTaskSnapshotRecoveryGeneration() {
  return generation;
}

export function useTaskSnapshotRecoveryGeneration() {
  return useSyncExternalStore(
    subscribeTaskSnapshotRecovery,
    getTaskSnapshotRecoveryGeneration,
    getTaskSnapshotRecoveryGeneration
  );
}
