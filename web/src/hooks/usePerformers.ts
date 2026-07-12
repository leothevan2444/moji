import { useQuery } from "urql";
import {
  StashPerformerDetailDocument,
  StashPerformersDocument,
  type StashPerformerDetailQuery,
  type StashPerformerDetailQueryVariables,
  type StashPerformersQuery,
  type StashPerformersQueryVariables
} from "../graphql/generated/graphql";

interface Options { enabled: boolean; search: string | null; page: number; pageSize: number; performerId: string | null; }

export function usePerformers({ enabled, search, page, pageSize, performerId }: Options) {
  const [{ data: listData, fetching: fetchingStashPerformers, error: stashPerformersError }, refreshStashPerformers] = useQuery<StashPerformersQuery, StashPerformersQueryVariables>({
    query: StashPerformersDocument, variables: { search, page, pageSize }, requestPolicy: "cache-and-network", pause: !enabled || !!performerId
  });
  const [{ data: detailData, fetching: fetchingPerformerDetail, error: performerDetailError }, refreshPerformerDetail] = useQuery<StashPerformerDetailQuery, StashPerformerDetailQueryVariables>({
    query: StashPerformerDetailDocument, variables: { id: performerId ?? "" }, requestPolicy: "cache-and-network", pause: !enabled || !performerId
  });
  const stashPerformerPage = listData?.stashPerformers ?? null;
  return { stashPerformerPage, stashPerformers: stashPerformerPage?.items ?? [], performerDetail: detailData?.stashPerformerDetail ?? null, fetchingStashPerformers, fetchingPerformerDetail, stashPerformersError, performerDetailError, refreshStashPerformers, refreshPerformerDetail };
}
