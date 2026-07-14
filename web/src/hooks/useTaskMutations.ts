import { useMutation } from "urql";
import {
  AddTorrentDocumentDocument,
  DeleteTaskDocumentDocument,
  DeleteTasksDocumentDocument,
  ProcessTaskIngestDocumentDocument,
  RetryTaskDocumentDocument,
  RetryTasksDocumentDocument,
  SyncTaskProgressDocumentDocument,
  TriggerStashScansDocumentDocument,
  TriggerTaskStashScanDocumentDocument,
  type AddTorrentDocumentMutation,
  type AddTorrentDocumentMutationVariables,
  type DeleteTaskDocumentMutation,
  type DeleteTaskDocumentMutationVariables,
  type DeleteTasksDocumentMutation,
  type DeleteTasksDocumentMutationVariables,
  type ProcessTaskIngestDocumentMutation,
  type ProcessTaskIngestDocumentMutationVariables,
  type RetryTaskDocumentMutation,
  type RetryTaskDocumentMutationVariables,
  type RetryTasksDocumentMutation,
  type RetryTasksDocumentMutationVariables,
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

  const [{ fetching: syncingTaskProgress }, syncTaskProgress] = useMutation<
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

  const [{ fetching: retryingTasks }, retryTasks] = useMutation<RetryTasksDocumentMutation, RetryTasksDocumentMutationVariables>(RetryTasksDocumentDocument);
  const [{ fetching: processingTaskIngest }, processTaskIngest] = useMutation<ProcessTaskIngestDocumentMutation, ProcessTaskIngestDocumentMutationVariables>(ProcessTaskIngestDocumentDocument);
  const [{ fetching: deletingTasks }, deleteTasks] = useMutation<DeleteTasksDocumentMutation, DeleteTasksDocumentMutationVariables>(DeleteTasksDocumentDocument);

  const [, triggerTaskStashScan] = useMutation<
    TriggerTaskStashScanDocumentMutation,
    TriggerTaskStashScanDocumentMutationVariables
  >(TriggerTaskStashScanDocumentDocument);

  const [{ fetching: triggeringStashScans }, triggerStashScans] = useMutation<
    TriggerStashScansDocumentMutation,
    TriggerStashScansDocumentMutationVariables
  >(TriggerStashScansDocumentDocument);

  return {
    addTorrent,
    deleteTask,
    retryTask,
    retryTasks,
    retryingTasks,
    processTaskIngest,
    processingTaskIngest,
    deleteTasks,
    deletingTasks,
    syncTaskProgress,
    syncingTaskProgress,
    triggerTaskStashScan,
    triggerStashScans,
    triggeringStashScans
  };
}
