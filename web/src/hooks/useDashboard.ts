import { useMutation, useQuery } from "urql";
import {
  AddTorrentDocumentDocument,
  DashboardDocumentDocument,
  SyncTaskProgressDocumentDocument,
  TriggerStashScansDocumentDocument,
  TriggerTaskStashScanDocumentDocument,
  type AddTorrentDocumentMutation,
  type AddTorrentDocumentMutationVariables,
  type DashboardDocumentQuery,
  type DashboardDocumentQueryVariables,
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
export function useDashboard() {
  const [{ data, fetching, error }, refreshDashboard] = useQuery<
    DashboardDocumentQuery,
    DashboardDocumentQueryVariables
  >({
    query: DashboardDocumentDocument,
    requestPolicy: "cache-and-network"
  });

  const [, addTorrent] = useMutation<
    AddTorrentDocumentMutation,
    AddTorrentDocumentMutationVariables
  >(AddTorrentDocumentDocument);

  const [, syncTaskProgress] = useMutation<
    SyncTaskProgressDocumentMutation,
    SyncTaskProgressDocumentMutationVariables
  >(SyncTaskProgressDocumentDocument);

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
    syncTaskProgress,
    triggerTaskStashScan,
    triggerStashScans
  };
}
