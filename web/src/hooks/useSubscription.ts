import { useMutation, useQuery } from "urql";
import {
  RefreshSubscribedPerformerDocument,
  RefreshSubscriptionNowDocument,
  StashPerformersDocument,
  SubscribePerformerDocument,
  SubscribedPerformersDocument,
  UnsubscribePerformerDocument,
  type RefreshSubscribedPerformerMutation,
  type RefreshSubscribedPerformerMutationVariables,
  type RefreshSubscriptionNowMutation,
  type RefreshSubscriptionNowMutationVariables,
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
}

/**
 * Hook bundling all subscription-related queries and mutations.
 * Queries are paused when the subscription tab is not active.
 */
export function useSubscription({ enabled, search, page, pageSize }: UseSubscriptionOptions) {
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
    pause: !enabled
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

  const stashPerformerPage = stashPerformersData?.stashPerformers ?? null;

  const reloadSubscription = async () => {
    await Promise.all([
      refreshSubscription({ requestPolicy: "network-only" }),
      refreshStashPerformers({ requestPolicy: "network-only" })
    ]);
  };

  return {
    stashPerformerPage,
    stashPerformers: stashPerformerPage?.items ?? [],
    subscribedPerformers: subscriptionData?.subscribedPerformers ?? [],
    fetchingStashPerformers,
    fetchingSubscription,
    stashPerformersError,
    subscriptionError,
    refreshingSubscriptionNow,
    subscribePerformer,
    unsubscribePerformer,
    refreshSubscribedPerformer,
    refreshSubscriptionsNow,
    reloadSubscription
  };
}
