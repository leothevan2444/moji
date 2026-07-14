import { useClient, useQuery } from "urql";
import {
	PerformerWorkspaceDocument,
	StashPerformerDetailDocument,
	type PerformerWorkspaceQuery,
	type PerformerWorkspaceQueryVariables,
	type StashPerformerDetailQuery,
	type StashPerformerDetailQueryVariables
} from "../graphql/generated/graphql";

interface Options { enabled: boolean; search: string | null; page: number; pageSize: number; performerId: string | null; }

export function usePerformers({ enabled, search, page, pageSize, performerId }: Options) {
	const client = useClient();
  const [{ data: listData, fetching: fetchingStashPerformers, error: stashPerformersError }, refreshStashPerformers] = useQuery<PerformerWorkspaceQuery, PerformerWorkspaceQueryVariables>({
    query: PerformerWorkspaceDocument, variables: { search, page, pageSize }, requestPolicy: "cache-and-network", pause: !enabled
  });
  const [{ data: detailData, fetching: fetchingPerformerDetail, error: performerDetailError }, refreshPerformerDetail] = useQuery<StashPerformerDetailQuery, StashPerformerDetailQueryVariables>({
    query: StashPerformerDetailDocument, variables: { id: performerId ?? "" }, requestPolicy: "cache-and-network", pause: !enabled || !performerId
  });
  const stashPerformerPage = listData?.performerWorkspace.performers ?? null;
  const reloadStashPerformers = () => client.query(PerformerWorkspaceDocument, { search, page, pageSize }, { requestPolicy: "network-only" }).toPromise();
  return { stashPerformerPage, stashPerformers: stashPerformerPage?.items ?? [], subscribedPerformers: listData?.performerWorkspace.subscribedPerformers ?? [], performerDetail: detailData?.stashPerformerDetail ?? null, fetchingStashPerformers, fetchingPerformerDetail, stashPerformersError, performerDetailError, refreshStashPerformers, reloadStashPerformers, refreshPerformerDetail };
}
