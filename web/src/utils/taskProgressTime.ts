import type { DashboardTask } from "./taskUtils";

export function taskProgressTime(task: DashboardTask) {
  if (task.lastProgressAt) return task.lastProgressAt;
  // Historical tasks have no progress clock. Use only recorded milestones,
  // never the record's polling timestamp.
  return [task.sourcingCompletedAt, task.downloadCompletedAt, task.stashScanStartedAt, task.ingestCompletedAt]
    .filter((time): time is string => typeof time === "string" && Number.isFinite(Date.parse(time)))
    .sort((a, b) => Date.parse(b) - Date.parse(a))[0] ?? null;
}
