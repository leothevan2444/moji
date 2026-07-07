import { useQuery } from "urql";
import {
  PreviewJackettSelectionDocumentDocument,
  type PreviewJackettSelectionInput,
  type PreviewJackettSelectionDocumentQuery,
  type PreviewJackettSelectionDocumentQueryVariables,
  type SearchDocumentQuery
} from "../graphql/generated/graphql";

type JackettResult = SearchDocumentQuery["jackettSearch"][number];

interface UsePreviewJackettSelectionOptions {
  enabled?: boolean;
  applyFastRules: boolean;
  applyFileRules: boolean;
  inspectionCandidateLimit?: number;
}

export function usePreviewJackettSelection(
  query: string,
  results: ReadonlyArray<JackettResult>,
  options: UsePreviewJackettSelectionOptions
) {
  const {
    enabled = true,
    applyFastRules,
    applyFileRules,
    inspectionCandidateLimit
  } = options;

  const shouldRun =
    enabled &&
    query.trim().length > 0 &&
    results.length > 0 &&
    (applyFastRules || applyFileRules);

  const [{ data, fetching, error }] = useQuery<
    PreviewJackettSelectionDocumentQuery,
    PreviewJackettSelectionDocumentQueryVariables
  >({
    query: PreviewJackettSelectionDocumentDocument,
    variables: {
      input: {
        query,
        results: results.map((result) => ({
          title: result.title,
          size: result.size,
          seeders: result.seeders,
          peers: result.peers,
          tracker: result.tracker,
          trackerId: result.trackerId,
          categoryDesc: result.categoryDesc,
          publishDate: result.publishDate,
          details: result.details,
          link: result.link,
          magnetUri: result.magnetUri,
          infoHash: result.infoHash
        })),
        applyFastRules,
        applyFileRules,
        ...(inspectionCandidateLimit && inspectionCandidateLimit > 0 ? { inspectionCandidateLimit } : {})
      } satisfies PreviewJackettSelectionInput
    },
    pause: !shouldRun
  });

  return {
    results: (data?.previewJackettSelection.results ?? []) as SearchDocumentQuery["jackettSearch"],
    previewMeta: data?.previewJackettSelection.previewMeta ?? null,
    fetching,
    error
  };
}
