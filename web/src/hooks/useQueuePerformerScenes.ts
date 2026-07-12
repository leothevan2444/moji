import { useMutation } from "urql";
import { QueuePerformerScenesDocument, type QueuePerformerScenesMutation, type QueuePerformerScenesMutationVariables } from "../graphql/generated/graphql";

export function useQueuePerformerScenes() {
  const [{ fetching: queueingPerformerScenes }, queuePerformerScenes] = useMutation<QueuePerformerScenesMutation, QueuePerformerScenesMutationVariables>(QueuePerformerScenesDocument);
  const [, queueSinglePerformerScene] = useMutation<QueuePerformerScenesMutation, QueuePerformerScenesMutationVariables>(QueuePerformerScenesDocument);
  return { queueingPerformerScenes, queuePerformerScenes, queueSinglePerformerScene };
}
