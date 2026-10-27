import { useCallback, useEffect, useRef } from "react";
import { useClient } from "urql";
import { HomePageDocumentDocument, TasksOverviewDocumentDocument } from "../../graphql/generated/graphql";
import { notifyTaskSnapshotRecovery } from "../../graphql/taskRecovery";
import { useTaskEvents } from "../../hooks/useTaskEvents";

type RecoveryReason = "initial" | "reconnect" | "sequence-gap";

export function TaskEventBridge({ delay = 200, scope = "tasks" }: { delay?: number; scope?: "home" | "tasks" }) {
  const client = useClient();
  const currentScope = useRef(scope);
  useEffect(() => { currentScope.current = scope; }, [scope]);
  const timer = useRef<number | null>(null);
  const inFlight = useRef(false);
  const pendingReason = useRef<RecoveryReason | null>(null);
  const mounted = useRef(true);
  const initialCalibrationScheduled = useRef(false);

  const scheduleRecovery = useCallback((reason: RecoveryReason): void => {
    pendingReason.current = reason;
    if (timer.current !== null || inFlight.current) return;
    timer.current = window.setTimeout(() => {
      timer.current = null;
      const currentReason = pendingReason.current;
      pendingReason.current = null;
      inFlight.current = true;
      notifyTaskSnapshotRecovery();
      if (import.meta.env.DEV) console.info("GraphQL task snapshots recalibrating", currentReason);
      void client.query(currentScope.current === "home" ? HomePageDocumentDocument : TasksOverviewDocumentDocument, {}, { requestPolicy: "network-only" }).toPromise().then((result) => {
        if (result.error && import.meta.env.DEV) console.error("GraphQL task snapshot recalibration failed", result.error);
      }).catch(error => {
        if (import.meta.env.DEV) console.error("GraphQL task snapshot recalibration failed", error);
      }).finally(() => {
        inFlight.current = false;
        // A gap after the query took its snapshot still needs one follow-up.
        if (mounted.current && pendingReason.current) scheduleRecovery(pendingReason.current);
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

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      initialCalibrationScheduled.current = false;
      pendingReason.current = null;
      if (timer.current !== null) window.clearTimeout(timer.current);
      timer.current = null;
    };
  }, []);

  return null;
}
