import { useMutation } from "urql";
import {
  AddTorrentDocumentDocument,
  DeleteTaskDocumentDocument,
  DeleteTasksDocumentDocument,
  RestartTaskSourcingDocumentDocument,
  RetryTaskDocumentDocument,
  RetryTasksDocumentDocument,
  type AddTorrentDocumentMutation,
  type AddTorrentDocumentMutationVariables,
  type DeleteTaskDocumentMutation,
  type DeleteTaskDocumentMutationVariables,
  type DeleteTasksDocumentMutation,
  type DeleteTasksDocumentMutationVariables,
  type RetryTaskDocumentMutation,
  type RetryTaskDocumentMutationVariables,
  type RetryTasksDocumentMutation,
  type RetryTasksDocumentMutationVariables,
} from "../graphql/generated/graphql";

/**
 * Hook bundling all dashboard-level queries and the mutations that mutate
 * task state (add torrent, sync downloads and advance ingest).
 */
export function useTaskMutations() {
  const [, addTorrent] = useMutation<
    AddTorrentDocumentMutation,
    AddTorrentDocumentMutationVariables
  >(AddTorrentDocumentDocument);

  const [, deleteTask] = useMutation<
    DeleteTaskDocumentMutation,
    DeleteTaskDocumentMutationVariables
  >(DeleteTaskDocumentDocument);

  const [{ fetching: restartingTaskSourcing }, restartTaskSourcing] = useMutation(RestartTaskSourcingDocumentDocument);

  const [, retryTask] = useMutation<
    RetryTaskDocumentMutation,
    RetryTaskDocumentMutationVariables
  >(RetryTaskDocumentDocument);

  const [{ fetching: retryingTasks }, retryTasks] = useMutation<RetryTasksDocumentMutation, RetryTasksDocumentMutationVariables>(RetryTasksDocumentDocument);
  const [{ fetching: deletingTasks }, deleteTasks] = useMutation<DeleteTasksDocumentMutation, DeleteTasksDocumentMutationVariables>(DeleteTasksDocumentDocument);


  return {
    addTorrent,
    deleteTask,
    retryTask,
    restartTaskSourcing,
    restartingTaskSourcing,
    retryTasks,
    retryingTasks,
    deleteTasks,
    deletingTasks,
  };
}
