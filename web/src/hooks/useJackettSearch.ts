import { useQuery } from "urql";
import {
  SearchDocumentDocument,
  type JackettSearchInput,
  type SearchDocumentQuery,
  type SearchDocumentQueryVariables
} from "../graphql/generated/graphql";

/**
 * Hook wrapping the Jackett search query. Pauses automatically when the
 * deferred query is empty.
 */
export function useJackettSearch(deferredQuery: string) {
  const [{ data, fetching, error }] = useQuery<
    SearchDocumentQuery,
    SearchDocumentQueryVariables
  >({
    query: SearchDocumentDocument,
    variables: {
      input: {
        query: deferredQuery,
        limit: 18
      } satisfies JackettSearchInput
    },
    pause: deferredQuery.length === 0
  });

  return {
    results: data?.jackettSearch ?? [],
    fetching,
    error
  };
}
