import { useState } from "react";
import { useLocation, useNavigate, useOutletContext } from "react-router";
import { useQuery } from "urql";
import { HomePage } from "../../pages/HomePage";
import { useTaskMutations } from "../../hooks/useTaskMutations";
import { useServiceStatusEvents } from "../../hooks/useServiceStatusEvents";
import { describeQueryError } from "../../services/queryError";
import { taskSummary } from "../../utils/taskUtils";
import type { SettingsTab } from "../../types";
import type { AppOutletContext } from "../AppLayout";
import { HomePageDocumentDocument, HomeServiceStatusDocument } from "../../graphql/generated/graphql";
import { useTranslation } from "react-i18next";

const settingsSlugs: Record<SettingsTab, string> = {
  connections: "connections", ingest: "ingest", automation: "automation",
  system: "system", logs: "logs", about: "about"
};

export function Component() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const [pendingTaskScanId, setPendingTaskScanId] = useState<string | null>(null);
  const [pendingTaskRetryId, setPendingTaskRetryId] = useState<string | null>(null);
  const [{ data, error, fetching }, refreshDashboard] = useQuery({
    query: HomePageDocumentDocument,
    requestPolicy: "cache-and-network"
  });
  const [{ data: serviceStatusData, error: serviceStatusError, fetching: serviceStatusFetching }, refreshServiceStatus] = useQuery({
    query: HomeServiceStatusDocument,
    requestPolicy: "cache-and-network"
  });
  useServiceStatusEvents({
    onRefresh: () => refreshServiceStatus({ requestPolicy: "network-only" })
  });
  const { retryTask, triggerTaskStashScan } = useTaskMutations();
  const tasks = data?.tasks ?? [];

  const runTaskScan = async (taskId: string) => {
    setPendingTaskScanId(taskId);
    try {
      const result = await triggerTaskStashScan({ id: taskId });
      if (result.error) pushToast("tone-danger", describeQueryError(result.error));
    } finally {
      setPendingTaskScanId(null);
    }
  };

  const runRetryTask = async (taskId: string) => {
    const task = tasks.find((entry) => entry.id === taskId);
    setPendingTaskRetryId(taskId);
    try {
      const result = await retryTask({ id: taskId });
      if (result.error) pushToast("tone-danger", describeQueryError(result.error));
      else if (!result.data?.retryTask?.id) pushToast("tone-danger", t("home.retryNoResult"));
      else pushToast("tone-success", t("home.retried", { task: task ? taskSummary(task) : taskId }));
    } finally {
      setPendingTaskRetryId(null);
    }
  };

  const pageError = error && !data ? error : serviceStatusError && !serviceStatusData ? serviceStatusError : null;
  if (pageError) {
    return <div className="empty-card"><h3>{t("home.loadFailed")}</h3><p>{describeQueryError(pageError)}</p><button type="button" disabled={fetching || serviceStatusFetching} onClick={() => { refreshDashboard({ requestPolicy: "network-only" }); refreshServiceStatus({ requestPolicy: "network-only" }); }}>{t("common.retry")}</button></div>;
  }

  return <HomePage
    tasks={tasks}
    runtimeSettings={data?.settings ?? null}
    runtimeStatus={serviceStatusData?.settingsStatus ?? null}
    pendingTaskScanId={pendingTaskScanId}
    pendingTaskRetryId={pendingTaskRetryId}
    onOpenTask={(id) => navigate(`/tasks/${encodeURIComponent(id)}`, { state: { backgroundLocation: location } })}
    onScanTask={(id) => void runTaskScan(id)}
    onRetryTask={(id) => void runRetryTask(id)}
    onResolveTask={(id) => navigate(`/tasks/${encodeURIComponent(id)}/resolve`, { state: { backgroundLocation: location } })}
    onOpenSettings={(tab) => navigate(`/settings/${settingsSlugs[tab]}`)}
  />;
}
