import { useMutation, useQuery } from "urql";
import {
  type LibraryFilter,
  QueuePerformerScenesDocument,
  RefreshSubscribedPerformerDocument,
  RefreshSubscriptionNowDocument,
  type SceneSourceFilter,
  StashPerformerDetailDocument,
  type StashPerformerDetailQuery,
  type StashPerformerDetailQueryVariables,
  StashPerformerScenesDocument,
  type StashPerformerScenesQuery,
  type StashPerformerScenesQueryVariables,
  StashPerformersDocument,
  SubscribePerformerDocument,
  SubscribedPerformersDocument,
  UnsubscribePerformerDocument,
  type RefreshSubscribedPerformerMutation,
  type RefreshSubscribedPerformerMutationVariables,
  type RefreshSubscriptionNowMutation,
  type RefreshSubscriptionNowMutationVariables,
  type QueuePerformerScenesMutation,
  type QueuePerformerScenesMutationVariables,
  type StashPerformersQuery,
  type StashPerformersQueryVariables,
  type SubscribePerformerMutation,
  type SubscribePerformerMutationVariables,
  type SubscribedPerformersQuery,
  type UnsubscribePerformerMutation,
  type UnsubscribePerformerMutationVariables
} from "../graphql/generated/graphql";

interface UseSubscriptionOptions {
  enabled: boolean;
  search: string | null;
  page: number;
  pageSize: number;
  performerId: string | null;
  performerSceneSearch: string | null;
  performerSceneSource: SceneSourceFilter;
  performerSceneLibrary: LibraryFilter;
  performerScenePage: number;
  performerScenePageSize: number;
}

/**
 * Hook bundling all subscription-related queries and mutations.
 * Queries are paused when the subscription tab is not active.
 */
export function useSubscription({
  enabled,
  search,
  page,
  pageSize,
  performerId,
  performerSceneSearch,
  performerSceneSource,
  performerSceneLibrary,
  performerScenePage,
  performerScenePageSize
}: UseSubscriptionOptions) {
  const [
    {
      data: stashPerformersData,
      fetching: fetchingStashPerformers,
      error: stashPerformersError
    },
    refreshStashPerformers
  ] = useQuery<StashPerformersQuery, StashPerformersQueryVariables>({
    query: StashPerformersDocument,
    variables: {
      search,
      page,
      pageSize
    },
    requestPolicy: "cache-and-network",
    pause: !enabled || !!performerId
  });

  const [
    {
      data: performerDetailData,
      fetching: fetchingPerformerDetail,
      error: performerDetailError
    },
    refreshPerformerDetail
  ] = useQuery<StashPerformerDetailQuery, StashPerformerDetailQueryVariables>({
    query: StashPerformerDetailDocument,
    variables: { id: performerId ?? "" },
    requestPolicy: "cache-and-network",
    pause: !enabled || !performerId
  });

  const [
    {
      data: performerScenesData,
      fetching: fetchingPerformerScenes,
      error: performerScenesError
    },
    refreshPerformerScenes
  ] = useQuery<StashPerformerScenesQuery, StashPerformerScenesQueryVariables>({
    query: StashPerformerScenesDocument,
    variables: {
      id: performerId ?? "",
      input: {
        search: performerSceneSearch,
        source: performerSceneSource,
        inLibrary: performerSceneLibrary,
        page: performerScenePage,
        pageSize: performerScenePageSize
      }
    },
    requestPolicy: "cache-and-network",
    pause: !enabled || !performerId
  });

  const [
    {
      data: subscriptionData,
      fetching: fetchingSubscription,
      error: subscriptionError
    },
    refreshSubscription
  ] = useQuery<SubscribedPerformersQuery, Record<string, never>>({
    query: SubscribedPerformersDocument,
    requestPolicy: "cache-and-network",
    pause: !enabled
  });

  const [, subscribePerformer] = useMutation<
    SubscribePerformerMutation,
    SubscribePerformerMutationVariables
  >(SubscribePerformerDocument);

  const [, unsubscribePerformer] = useMutation<
    UnsubscribePerformerMutation,
    UnsubscribePerformerMutationVariables
  >(UnsubscribePerformerDocument);

  const [, refreshSubscribedPerformer] = useMutation<
    RefreshSubscribedPerformerMutation,
    RefreshSubscribedPerformerMutationVariables
  >(RefreshSubscribedPerformerDocument);

  const [{ fetching: refreshingSubscriptionNow }, refreshSubscriptionsNow] = useMutation<
    RefreshSubscriptionNowMutation,
    RefreshSubscriptionNowMutationVariables
  >(RefreshSubscriptionNowDocument);

  const [{ fetching: queueingPerformerScenes }, queuePerformerScenes] = useMutation<
    QueuePerformerScenesMutation,
    QueuePerformerScenesMutationVariables
  >(QueuePerformerScenesDocument);

  const [, queueSinglePerformerScene] = useMutation<
    QueuePerformerScenesMutation,
    QueuePerformerScenesMutationVariables
  >(QueuePerformerScenesDocument);

  const stashPerformerPage = stashPerformersData?.stashPerformers ?? null;

  const reloadSubscription = async () => {
    await Promise.all([
      refreshSubscription({ requestPolicy: "network-only" }),
      refreshStashPerformers({ requestPolicy: "network-only" }),
      refreshPerformerDetail({ requestPolicy: "network-only" }),
      refreshPerformerScenes({ requestPolicy: "network-only" })
    ]);
  };

  return {
    stashPerformerPage,
    stashPerformers: stashPerformerPage?.items ?? [],
    performerDetail: performerDetailData?.stashPerformerDetail ?? null,
    performerScenePage: performerScenesData?.stashPerformerScenes ?? null,
    performerScenes: performerScenesData?.stashPerformerScenes.items ?? [],
    subscribedPerformers: subscriptionData?.subscribedPerformers ?? [],
    fetchingStashPerformers,
    fetchingPerformerDetail,
    fetchingPerformerScenes,
    fetchingSubscription,
    stashPerformersError,
    performerDetailError,
    performerScenesError,
    subscriptionError,
    refreshingSubscriptionNow,
    queueingPerformerScenes,
    subscribePerformer,
    unsubscribePerformer,
    refreshSubscribedPerformer,
    refreshSubscriptionsNow,
    queuePerformerScenes,
    queueSinglePerformerScene,
    reloadSubscription
  };
}
