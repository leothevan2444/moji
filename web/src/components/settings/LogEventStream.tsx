import { useEffect, useRef } from "react";
import { useLogEvents, type StreamedLogEntry } from "../../hooks/useLogEvents";

interface LogEventStreamProps {
  pause: boolean;
  onEntries: (entries: StreamedLogEntry[]) => void;
  onResync: () => void;
  batchDelay?: number;
}

export function LogEventStream({ pause, onEntries, onResync, batchDelay = 75 }: LogEventStreamProps) {
  const entriesCallback = useRef(onEntries);
  const resyncCallback = useRef(onResync);
  const pending = useRef<StreamedLogEntry[]>([]);
  const timer = useRef<number | null>(null);

  useEffect(() => {
    entriesCallback.current = onEntries;
    resyncCallback.current = onResync;
  }, [onEntries, onResync]);

  const clearPending = () => {
    pending.current = [];
    if (timer.current !== null) window.clearTimeout(timer.current);
    timer.current = null;
  };

  useEffect(() => {
    if (pause) clearPending();
    return clearPending;
  }, [pause]);

  useLogEvents({
    pause,
    onEvent(entry) {
      pending.current.unshift(entry);
      if (timer.current !== null) return;
      timer.current = window.setTimeout(() => {
        timer.current = null;
        const entries = pending.current;
        pending.current = [];
        if (entries.length > 0) entriesCallback.current(entries);
      }, batchDelay);
    },
    onResync() {
      clearPending();
      resyncCallback.current();
    }
  });

  return null;
}
