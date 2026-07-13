import { useMutation, useQuery } from "urql";
import { RefreshSubscribedPerformerDocument, RefreshSubscriptionsNowDocument, SubscribePerformerDocument, SubscribedPerformersDocument, UnsubscribePerformerDocument, type RefreshSubscribedPerformerMutation, type RefreshSubscribedPerformerMutationVariables, type RefreshSubscriptionsNowMutation, type RefreshSubscriptionsNowMutationVariables, type SubscribePerformerMutation, type SubscribePerformerMutationVariables, type SubscribedPerformersQuery, type UnsubscribePerformerMutation, type UnsubscribePerformerMutationVariables } from "../graphql/generated/graphql";
import { usePerformerSubscriptionEvents } from "./usePerformerSubscriptionEvents";

export function useSubscriptions(enabled: boolean) {
  const [{ data, fetching: fetchingSubscription, error: subscriptionError }, refreshSubscription] = useQuery<SubscribedPerformersQuery, Record<string, never>>({ query: SubscribedPerformersDocument, requestPolicy: "cache-and-network", pause: !enabled });
  usePerformerSubscriptionEvents({ enabled, onRefresh: () => refreshSubscription({ requestPolicy: "network-only" }) });
  const [, subscribePerformer] = useMutation<SubscribePerformerMutation, SubscribePerformerMutationVariables>(SubscribePerformerDocument);
  const [, unsubscribePerformer] = useMutation<UnsubscribePerformerMutation, UnsubscribePerformerMutationVariables>(UnsubscribePerformerDocument);
  const [, refreshSubscribedPerformer] = useMutation<RefreshSubscribedPerformerMutation, RefreshSubscribedPerformerMutationVariables>(RefreshSubscribedPerformerDocument);
  const [{ fetching: refreshingSubscriptionNow }, refreshSubscriptionsNow] = useMutation<RefreshSubscriptionsNowMutation, RefreshSubscriptionsNowMutationVariables>(RefreshSubscriptionsNowDocument);
  return { subscribedPerformers: data?.subscribedPerformers ?? [], fetchingSubscription, subscriptionError, refreshSubscription, subscribePerformer, unsubscribePerformer, refreshSubscribedPerformer, refreshingSubscriptionNow, refreshSubscriptionsNow };
}
