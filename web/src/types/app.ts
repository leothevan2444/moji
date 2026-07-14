/**
 * Application-level type definitions shared across components.
 */

export type DrawerKey = "stats" | "settings" | "help" | "task" | "task-resolution" | "task-batch-result" | "discovery" | "confirm" | null;

export type ToastTone = "tone-success" | "tone-danger" | "tone-info";
export type ToastPhase = "entering" | "leaving";
export type ToastItem = {
  id: number;
  tone: ToastTone;
  message: string;
  phase: ToastPhase;
  /** Override the default lifetime (ms) for the progress bar animation. */
  lifetimeMs?: number;
};

export type SettingsTab = "connections" | "ingest" | "automation" | "system" | "logs" | "about";

export type TaskStatusFilter = "all" | "running" | "completed" | "failed" | "scanPending";
export type TaskSortKey = "createdAt" | "updatedAt" | "progress";
