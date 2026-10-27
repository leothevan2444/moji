// A bounded transport journal protects HTTP snapshots from newer socket updates.
// Keep this outside React: a burst may contain updates for many different tasks.
export interface TaskRealtimeEvent {
  sequence: number;
  streamId: string;
  type: string;
  taskId: string;
  listChanged: boolean;
  task?: Record<string, any> | null;
  dashboardStats?: Record<string, any>;
}
let streamId = "";
let sequence = 0;
let readyGeneration = 0;
let membershipSequence = 0;
let journalFloor = 0;
let lastStats: Record<string, any> | undefined;
const latest = new Map<string, TaskRealtimeEvent>();
const listeners = new Set<(events: TaskRealtimeEvent[]) => void>();
const readyListeners = new Set<() => void>();
export const taskStreamCursor = () => ({ streamId, sequence, membershipSequence, readyGeneration, journalFloor });
export const getTaskReadyGeneration = () => readyGeneration;
export function subscribeTaskReady(listener: () => void) { readyListeners.add(listener); return () => { readyListeners.delete(listener); }; }
export function subscribeTaskBatches(listener: (events: TaskRealtimeEvent[]) => void) { listeners.add(listener); return () => { listeners.delete(listener); }; }
export function latestTaskEvent(id: string) { return latest.get(id); }
export function recordTaskEvent(event: TaskRealtimeEvent) {
  if (event.type === "READY") {
    streamId = event.streamId;
    sequence = event.sequence;
    membershipSequence = event.sequence;
    latest.clear();
    journalFloor = event.sequence;
    lastStats = undefined;
    readyGeneration++;
    readyListeners.forEach(listener => listener());
    return;
  }
  sequence = event.sequence;
  if (event.dashboardStats) lastStats = event.dashboardStats;
  if (event.streamId) streamId = event.streamId;
  if (event.listChanged) membershipSequence = event.sequence;
  latest.delete(event.taskId);
  latest.set(event.taskId, event);
  if (latest.size > 512) { const key=latest.keys().next().value!;journalFloor=Math.max(journalFloor,latest.get(key)!.sequence);latest.delete(key); }
}
export function publishTaskBatch(events: TaskRealtimeEvent[]) { listeners.forEach(listener => listener(events)); }

export function protectTaskSnapshot(value: any): any {
  if (Array.isArray(value)) return value.map(protectTaskSnapshot).filter(item => item !== null);
  if (!value || typeof value !== "object") return value;
  if (value.__typename === "TaskSummary") {
    const event = latest.get(value.id);
    if (event?.type === "DELETED") return null;
    if (event?.task && event.task.version > value.version) return { ...value, ...event.task };
    return value;
  }
  const result: Record<string, any> = {};
  for (const [key, item] of Object.entries(value)) result[key] = protectTaskSnapshot(item);
  if (value.streamId === streamId && typeof value.sequence === "number" && value.sequence < sequence && lastStats) {
    if (value.__typename === "TaskPage") result.stats = lastStats;
    if (value.__typename === "HomeTaskSnapshot") Object.assign(result, { total: lastStats.total, attention: lastStats.attention, waiting: lastStats.waiting, active: lastStats.total-lastStats.completed-lastStats.attention-lastStats.waiting });
  }
  return result;
}
