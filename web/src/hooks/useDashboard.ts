import { useMutation, useQuery } from "urql";
import {
  AddTorrentDocumentDocument,
  DashboardDocumentDocument,
  DeleteTaskDocumentDocument,
  RetryTaskDocumentDocument,
  SyncTaskProgressDocumentDocument,
  TriggerStashScansDocumentDocument,
  TriggerTaskStashScanDocumentDocument,
  type AddTorrentDocumentMutation,
  type AddTorrentDocumentMutationVariables,
  type DashboardDocumentQuery,
  type DashboardDocumentQueryVariables,
  type DeleteTaskDocumentMutation,
  type DeleteTaskDocumentMutationVariables,
  type RetryTaskDocumentMutation,
  type RetryTaskDocumentMutationVariables,
  type SyncTaskProgressDocumentMutation,
  type SyncTaskProgressDocumentMutationVariables,
  type TriggerStashScansDocumentMutation,
  type TriggerStashScansDocumentMutationVariables,
  type TriggerTaskStashScanDocumentMutation,
  type TriggerTaskStashScanDocumentMutationVariables
} from "../graphql/generated/graphql";

/**
 * Hook bundling all dashboard-level queries and the mutations that mutate
 * task state (add torrent, sync progress, trigger stash scans).
 */
export function useDashboard(enabled = true) {
  const [{ data, fetching, error }, refreshDashboard] = useQuery<
    DashboardDocumentQuery,
    DashboardDocumentQueryVariables
  >({
    query: DashboardDocumentDocument,
    requestPolicy: "cache-and-network",
    pause: !enabled
  });

  const [, addTorrent] = useMutation<
    AddTorrentDocumentMutation,
    AddTorrentDocumentMutationVariables
  >(AddTorrentDocumentDocument);

  const [, syncTaskProgress] = useMutation<
    SyncTaskProgressDocumentMutation,
    SyncTaskProgressDocumentMutationVariables
  >(SyncTaskProgressDocumentDocument);

  const [, deleteTask] = useMutation<
    DeleteTaskDocumentMutation,
    DeleteTaskDocumentMutationVariables
  >(DeleteTaskDocumentDocument);

  const [, retryTask] = useMutation<
    RetryTaskDocumentMutation,
    RetryTaskDocumentMutationVariables
  >(RetryTaskDocumentDocument);

  const [, triggerTaskStashScan] = useMutation<
    TriggerTaskStashScanDocumentMutation,
    TriggerTaskStashScanDocumentMutationVariables
  >(TriggerTaskStashScanDocumentDocument);

  const [, triggerStashScans] = useMutation<
    TriggerStashScansDocumentMutation,
    TriggerStashScansDocumentMutationVariables
  >(TriggerStashScansDocumentDocument);

  return {
    data,
    fetching,
    error,
    refreshDashboard,
    addTorrent,
    deleteTask,
    retryTask,
    syncTaskProgress,
    triggerTaskStashScan,
    triggerStashScans
  };
}
