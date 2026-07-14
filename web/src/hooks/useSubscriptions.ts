import { useMutation } from "urql";
import { RefreshSubscribedPerformerDocument, RefreshSubscribedPerformersDocument, RefreshSubscriptionsNowDocument, SubscribePerformerDocument, SubscribePerformersDocument, UnsubscribePerformerDocument, UnsubscribePerformersDocument, type PerformerWorkspaceQuery, type RefreshSubscribedPerformerMutation, type RefreshSubscribedPerformerMutationVariables, type RefreshSubscribedPerformersMutation, type RefreshSubscribedPerformersMutationVariables, type RefreshSubscriptionsNowMutation, type RefreshSubscriptionsNowMutationVariables, type SubscribePerformerMutation, type SubscribePerformerMutationVariables, type SubscribePerformersMutation, type SubscribePerformersMutationVariables, type UnsubscribePerformerMutation, type UnsubscribePerformerMutationVariables, type UnsubscribePerformersMutation, type UnsubscribePerformersMutationVariables } from "../graphql/generated/graphql";
import { usePerformerSubscriptionEvents } from "./usePerformerSubscriptionEvents";

type SubscribedPerformer = PerformerWorkspaceQuery["performerWorkspace"]["subscribedPerformers"][number];

export function useSubscriptions(enabled: boolean, subscribedPerformers: SubscribedPerformer[], refreshSubscription: () => void | Promise<unknown>) {
  usePerformerSubscriptionEvents({ enabled, onRefresh: refreshSubscription });
  const [, subscribePerformer] = useMutation<SubscribePerformerMutation, SubscribePerformerMutationVariables>(SubscribePerformerDocument);
  const [, unsubscribePerformer] = useMutation<UnsubscribePerformerMutation, UnsubscribePerformerMutationVariables>(UnsubscribePerformerDocument);
  const [, refreshSubscribedPerformer] = useMutation<RefreshSubscribedPerformerMutation, RefreshSubscribedPerformerMutationVariables>(RefreshSubscribedPerformerDocument);
  const [{ fetching: refreshingSubscriptionNow }, refreshSubscriptionsNow] = useMutation<RefreshSubscriptionsNowMutation, RefreshSubscriptionsNowMutationVariables>(RefreshSubscriptionsNowDocument);
  const [{ fetching: subscribingPerformers }, subscribePerformers] = useMutation<SubscribePerformersMutation, SubscribePerformersMutationVariables>(SubscribePerformersDocument);
  const [{ fetching: unsubscribingPerformers }, unsubscribePerformers] = useMutation<UnsubscribePerformersMutation, UnsubscribePerformersMutationVariables>(UnsubscribePerformersDocument);
  const [{ fetching: refreshingSubscribedPerformers }, refreshSubscribedPerformers] = useMutation<RefreshSubscribedPerformersMutation, RefreshSubscribedPerformersMutationVariables>(RefreshSubscribedPerformersDocument);
  return { subscribedPerformers, fetchingSubscription: false, subscriptionError: null, refreshSubscription, subscribePerformer, unsubscribePerformer, refreshSubscribedPerformer, refreshingSubscriptionNow, refreshSubscriptionsNow, subscribingPerformers, subscribePerformers, unsubscribingPerformers, unsubscribePerformers, refreshingSubscribedPerformers, refreshSubscribedPerformers };
}
