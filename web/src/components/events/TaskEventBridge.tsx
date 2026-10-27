import { useEffect, useSyncExternalStore } from "react";
import { useTaskEvents } from "../../hooks/useTaskEvents";
import { notifyTaskSnapshotRecovery } from "../../graphql/taskRecovery";
import { getTaskReadyGeneration, subscribeTaskReady } from "../../graphql/taskRealtime";

// Each mounted consumer recovers its own bounded snapshot.
export function TaskEventBridge() {
  useTaskEvents({ onSequenceGap: notifyTaskSnapshotRecovery });
  const ready = useSyncExternalStore(subscribeTaskReady, getTaskReadyGeneration, getTaskReadyGeneration);
  useEffect(() => { if (ready) notifyTaskSnapshotRecovery(); }, [ready]);
  return null;
}
