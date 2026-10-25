import { useMutation } from "urql";
import { SubscribePerformerDocument, SubscribePerformersDocument, UnsubscribePerformerDocument, UnsubscribePerformersDocument, type PerformerWorkspaceQuery } from "../graphql/generated/graphql";
import { usePerformerSubscriptionEvents } from "./usePerformerSubscriptionEvents";

type SubscribedPerformer = PerformerWorkspaceQuery["performerWorkspace"]["subscribedPerformers"][number];

export function useSubscriptions(enabled: boolean, subscribedPerformers: SubscribedPerformer[], refreshSubscription: () => void | Promise<unknown>) {
  usePerformerSubscriptionEvents({ enabled, onRefresh: refreshSubscription });
  const [, subscribePerformer] = useMutation(SubscribePerformerDocument);
  const [, unsubscribePerformer] = useMutation(UnsubscribePerformerDocument);
  const [{ fetching: subscribingPerformers }, subscribePerformers] = useMutation(SubscribePerformersDocument);
  const [{ fetching: unsubscribingPerformers }, unsubscribePerformers] = useMutation(UnsubscribePerformersDocument);
  return { subscribedPerformers, fetchingSubscription: false, subscriptionError: null, refreshSubscription, subscribePerformer, unsubscribePerformer, subscribingPerformers, subscribePerformers, unsubscribingPerformers, unsubscribePerformers };
}
