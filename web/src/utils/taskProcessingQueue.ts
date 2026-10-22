import { taskGroup, type DashboardTask } from "./taskUtils";

/** A review session has a stable order, starting with the oldest tasks. */
export function taskProcessingQueue(tasks: DashboardTask[]) {
  return tasks.filter(task => taskGroup(task) === "attention").sort((a, b) =>
    (Date.parse(a.createdAt) || 0) - (Date.parse(b.createdAt) || 0) || a.id.localeCompare(b.id));
}

export interface TaskQueueDraft {
  magnet?: string;
  videoIndices?: number[];
}
