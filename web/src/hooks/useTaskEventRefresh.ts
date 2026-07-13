import { useCallback, useEffect, useRef } from "react";
import type { TaskEvent } from "./useTaskEvents";

interface UseTaskEventRefreshOptions {
  taskId: string | undefined;
  refreshOverview: () => void | Promise<unknown>;
  refreshDetail: () => void | Promise<unknown>;
  delay?: number;
}

export function useTaskEventRefresh({
  taskId,
  refreshOverview,
  refreshDetail,
  delay = 200
}: UseTaskEventRefreshOptions) {
  const latest = useRef({ taskId, refreshOverview, refreshDetail });
  const timer = useRef<number | null>(null);
  const refreshCurrentDetail = useRef(false);

  useEffect(() => {
    latest.current = { taskId, refreshOverview, refreshDetail };
  }, [taskId, refreshOverview, refreshDetail]);

  const schedule = useCallback((includeDetail: boolean) => {
    refreshCurrentDetail.current ||= includeDetail;
    if (timer.current !== null) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => {
      timer.current = null;
      const shouldRefreshDetail = refreshCurrentDetail.current;
      refreshCurrentDetail.current = false;
      void latest.current.refreshOverview();
      if (shouldRefreshDetail && latest.current.taskId) void latest.current.refreshDetail();
    }, delay);
  }, [delay]);

  useEffect(() => () => {
    if (timer.current !== null) window.clearTimeout(timer.current);
  }, []);

  const onEvent = useCallback((event: TaskEvent) => {
    schedule(Boolean(latest.current.taskId && latest.current.taskId === event.taskId));
  }, [schedule]);

  const onFullRefresh = useCallback(() => {
    schedule(Boolean(latest.current.taskId));
  }, [schedule]);

  return { onEvent, onFullRefresh };
}
