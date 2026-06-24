import { useMutation, useQuery } from "urql";
import {
  DiscoverScenesDocumentDocument,
  QueueDiscoveredSceneDocumentDocument,
  type DiscoverScenesDocumentQuery,
  type DiscoverScenesDocumentQueryVariables,
  type DiscoverScenesInput,
  type QueueDiscoveredSceneDocumentMutation,
  type QueueDiscoveredSceneDocumentMutationVariables
} from "../graphql/generated/graphql";

export function useDiscoverScenes(deferredQuery: string, enabled: boolean) {
  const [{ data, fetching, error }] = useQuery<
    DiscoverScenesDocumentQuery,
    DiscoverScenesDocumentQueryVariables
  >({
    query: DiscoverScenesDocumentDocument,
    variables: {
      input: {
        query: deferredQuery,
        limit: 24
      } satisfies DiscoverScenesInput
    },
    pause: !enabled || deferredQuery.length === 0
  });

  const [, queueDiscoveredScene] = useMutation<
    QueueDiscoveredSceneDocumentMutation,
    QueueDiscoveredSceneDocumentMutationVariables
  >(QueueDiscoveredSceneDocumentDocument);

  return {
    result: data?.discoverScenes ?? null,
    results: data?.discoverScenes.items ?? [],
    fetching,
    error,
    queueDiscoveredScene
  };
}
