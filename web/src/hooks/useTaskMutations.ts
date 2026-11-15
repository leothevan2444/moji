import { useMutation } from "urql";
import {
  DeleteTaskDocumentDocument,
  DeleteTasksDocumentDocument,
  RestartTaskSourcingDocumentDocument,
  RetryTaskDocumentDocument,
  RetryTasksDocumentDocument,
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


  const [{ fetching: deletingTask }, deleteTask] = useMutation<
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
    deleteTask,
    deletingTask,
    retryTask,
    restartTaskSourcing,
    restartingTaskSourcing,
    retryTasks,
    retryingTasks,
    deleteTasks,
    deletingTasks,
  };
}