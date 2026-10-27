import { useEffect, useRef, useState } from "react";
import { useQuery } from "urql";
import { TaskDetailDocumentDocument } from "../graphql/generated/graphql";
import { latestTaskEvent, subscribeTaskBatches, taskStreamCursor } from "../graphql/taskRealtime";
import { useTaskSnapshotRecoveryGeneration } from "../graphql/taskRecovery";

// Detail payloads have a separate cache entity. Progress is patched locally;
// durable changes refetch only this detail after an in-flight read has settled.
export function useTaskDetail(id: string | null | undefined) {
  const [raw, refresh] = useQuery({ query: TaskDetailDocumentDocument, variables: { id: id ?? "" }, pause: !id, requestPolicy: "cache-and-network" });
  const [tick, setTick] = useState(0);
  const recovery = useTaskSnapshotRecoveryGeneration();
  const observedRecovery = useRef(recovery);
  const pendingRecovery = useRef<{streamId:string;sequence:number} | null>(null);
  useEffect(() => subscribeTaskBatches(events => {
    if (events.some(event => event.taskId === id)) setTick(tick => tick + 1);
  }), [id]);
  useEffect(() => {
    if (!id) return;
    if (observedRecovery.current !== recovery) { observedRecovery.current = recovery; pendingRecovery.current = {streamId:taskStreamCursor().streamId,sequence:taskStreamCursor().sequence}; }
    const cursor = taskStreamCursor(); const snapshot = raw.data?.taskDetailSnapshot;
    const boundary=pendingRecovery.current;
    if (boundary && snapshot && snapshot.streamId === boundary.streamId && snapshot.sequence >= boundary.sequence) pendingRecovery.current=null;
    const event = latestTaskEvent(id); const full = snapshot?.task;
    const staleRevision = full && event?.task && Number(event.task.revision) > full.revision;
    const staleBoundary = snapshot && cursor.readyGeneration && (snapshot.streamId !== cursor.streamId || snapshot.sequence < cursor.journalFloor);
    if (!raw.fetching && !raw.error && event?.type !== "DELETED" && (pendingRecovery.current || staleRevision || staleBoundary)) {
      pendingRecovery.current = null;
      refresh({ requestPolicy: "network-only" });
    }
  }, [id, recovery, refresh, raw.fetching, raw.data, raw.error, tick]);
  const event = id ? latestTaskEvent(id) : undefined;
  const full = raw.data?.taskDetailSnapshot.task;
  const stale = full && event?.task && Number(event.task.revision) > full.revision;
  const progressPatch = event?.task ? Object.fromEntries(
    ["version", "progress", "torrentName", "qbittorrentState", "lastProgressAt", "updatedAt"]
      .filter(field => field in event.task!).map(field => [field, event.task![field]])
  ) : {};
  const task = event?.type === "DELETED" ? null : full && !stale && event?.task && event.task.version > full.version
    ? { ...full, ...progressPatch, __typename: "Task" as const } : stale ? null : full;
  return [{ ...raw, data: event?.type === "DELETED" ? { task:null } : raw.data ? { task } : undefined,
    error: event?.type === "DELETED" ? undefined : raw.error,
    fetching: event?.type === "DELETED" ? false : raw.fetching || Boolean(stale) }, refresh] as const;
}
