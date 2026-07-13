import { useCallback, useEffect, useRef } from "react";
import { useClient } from "urql";
import { TasksOverviewDocumentDocument } from "../../graphql/generated/graphql";
import { notifyTaskSnapshotRecovery } from "../../graphql/taskRecovery";
import { useTaskEvents } from "../../hooks/useTaskEvents";

export function TaskEventBridge({ delay = 200 }: { delay?: number }) {
  const client = useClient();
  const timer = useRef<number | null>(null);
  const initialCalibrationScheduled = useRef(false);

  const scheduleRecovery = useCallback((reason: "initial" | "reconnect" | "sequence-gap") => {
    if (timer.current !== null) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => {
      timer.current = null;
      notifyTaskSnapshotRecovery();
      if (import.meta.env.DEV) console.info("GraphQL task snapshots recalibrating", reason);
      void client.query(TasksOverviewDocumentDocument, {}, { requestPolicy: "network-only" }).toPromise().then((result) => {
        if (result.error && import.meta.env.DEV) console.error("GraphQL task snapshot recalibration failed", result.error);
      });
    }, delay);
  }, [client, delay]);

  const subscription = useTaskEvents({
    onSequenceGap: () => scheduleRecovery("sequence-gap"),
    onReconnect: () => scheduleRecovery("reconnect")
  });

  useEffect(() => {
    if (subscription.connectionStatus !== "connected" || initialCalibrationScheduled.current) return;
    initialCalibrationScheduled.current = true;
    scheduleRecovery("initial");
  }, [scheduleRecovery, subscription.connectionStatus]);

  useEffect(() => () => {
    if (timer.current !== null) window.clearTimeout(timer.current);
  }, []);

  return null;
}
