import { useQuery } from "urql";
import { StashPerformerScenesDocument, type LibraryFilter, type SceneSourceFilter, type StashPerformerScenesQuery, type StashPerformerScenesQueryVariables } from "../graphql/generated/graphql";

interface Options { enabled: boolean; performerId: string | null; search: string | null; source: SceneSourceFilter; library: LibraryFilter; page: number; pageSize: number; }

export function usePerformerScenes({ enabled, performerId, search, source, library, page, pageSize }: Options) {
  const [{ data, fetching: fetchingPerformerScenes, error: performerScenesError }, refreshPerformerScenes] = useQuery<StashPerformerScenesQuery, StashPerformerScenesQueryVariables>({
    query: StashPerformerScenesDocument,
    variables: { id: performerId ?? "", input: { search, source, inLibrary: library, page, pageSize } },
    requestPolicy: "cache-and-network", pause: !enabled || !performerId
  });
  return { performerScenePage: data?.stashPerformerScenes ?? null, performerScenes: data?.stashPerformerScenes.items ?? [], fetchingPerformerScenes, performerScenesError, refreshPerformerScenes };
}
