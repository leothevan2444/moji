import { useMutation, useQuery } from "urql";
import {
  DiscoverScenesDocumentDocument,
  QueueDiscoveredSceneDocumentDocument,
  type DiscoverScenesDocumentQuery,
  type DiscoverScenesDocumentQueryVariables,
  type DiscoverScenesInput,
  type DiscoverSortBy,
  type QueueDiscoveredSceneDocumentMutation,
  type QueueDiscoveredSceneDocumentMutationVariables
} from "../graphql/generated/graphql";

const DISCOVER_PAGE_SIZE = 50;

interface UseDiscoverScenesOptions {
  enabled: boolean;
  sortBy?: DiscoverSortBy;
}

export function useDiscoverScenes(deferredQuery: string, options: UseDiscoverScenesOptions | boolean) {
  const { enabled, sortBy } = typeof options === "boolean" ? { enabled: options, sortBy: undefined } : options;

  const [{ data, fetching, error }] = useQuery<
    DiscoverScenesDocumentQuery,
    DiscoverScenesDocumentQueryVariables
  >({
    query: DiscoverScenesDocumentDocument,
    variables: {
      input: {
        query: deferredQuery,
        limit: DISCOVER_PAGE_SIZE,
        ...(sortBy ? { sortBy } : {})
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
