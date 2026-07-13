import { useQuery } from "urql";
import {
  SearchDocumentDocument,
  type JackettSearchInput,
  type JackettSortBy,
  type SearchDocumentQuery,
  type SearchDocumentQueryVariables
} from "../graphql/generated/graphql";

const JACKETT_PAGE_SIZE = 50;

interface UseJackettSearchOptions {
  enabled?: boolean;
  trackers?: string[];
  sortBy?: JackettSortBy;
}

/**
 * Hook wrapping the Jackett search query. Pauses automatically when the
 * deferred query is empty.
 *
 *  - `trackers` 透传到后端的 `JackettSearchInput.trackers`，空数组表示不过滤。
 *  - `sortBy` 透传到后端做内存排序。
 */
export function useJackettSearch(deferredQuery: string, options: UseJackettSearchOptions | boolean = true) {
  const { enabled = true, trackers, sortBy } = typeof options === "boolean" ? { enabled: options } : options;

  const [{ data, fetching, error }] = useQuery<
    SearchDocumentQuery,
    SearchDocumentQueryVariables
  >({
    query: SearchDocumentDocument,
    variables: {
      input: {
        query: deferredQuery,
        limit: JACKETT_PAGE_SIZE,
        ...(trackers && trackers.length > 0 ? { trackers } : {}),
        ...(sortBy ? { sortBy } : {})
      } satisfies JackettSearchInput
    },
    requestPolicy: "network-only",
    pause: !enabled || deferredQuery.length === 0
  });

  return {
    results: data?.jackettSearch ?? [],
    fetching,
    error
  };
}
