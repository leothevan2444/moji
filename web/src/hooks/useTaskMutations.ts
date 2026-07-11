import { useMutation } from "urql";
import {
  AddTorrentDocumentDocument,
  DeleteTaskDocumentDocument,
  RetryTaskDocumentDocument,
  SyncTaskProgressDocumentDocument,
  TriggerStashScansDocumentDocument,
  TriggerTaskStashScanDocumentDocument,
  type AddTorrentDocumentMutation,
  type AddTorrentDocumentMutationVariables,
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
export function useTaskMutations() {
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
    addTorrent,
    deleteTask,
    retryTask,
    syncTaskProgress,
    triggerTaskStashScan,
    triggerStashScans
  };
}
