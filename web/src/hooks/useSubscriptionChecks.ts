import { useCallback, useMemo } from "react";
import { useMutation, useQuery } from "urql";
import { StartSubscriptionCheckDocument, SubscriptionCheckActivityDocument, type SubscriptionCheckActivityQuery } from "../graphql/generated/graphql";
import { useSubscriptionCheckPolling } from "./useSubscriptionCheckPolling";
import { describeQueryError } from "../services/queryError";

export type SubscriptionCheckActivityView = SubscriptionCheckActivityQuery["subscriptionCheckActivity"];

// Actor pages need reservations only; the Home route owns the check overview and details.
export function useSubscriptionChecks(enabled: boolean, performerIds: string[] = []) {
  const key = [...new Set(performerIds)].sort().join("\u0000");
  const ids = useMemo(() => key ? key.split("\u0000") : [], [key]);
  const [{ data, error }, refresh] = useQuery({ query: SubscriptionCheckActivityDocument, variables: { ids }, pause: !enabled, requestPolicy: "network-only" });
  const [{ fetching: startingCheck }, start] = useMutation(StartSubscriptionCheckDocument);
  const checkActivity = data?.subscriptionCheckActivity ?? null;
  const refreshChecks = useCallback(() => refresh({ requestPolicy: "network-only" }), [refresh]);
  useSubscriptionCheckPolling(enabled, Boolean(checkActivity?.activeCount), refreshChecks);
  const startCheck = async (ids?: string[]) => {
    const result = await start(ids ? { ids } : {});
    refreshChecks();
    return { error: result.error, run: result.data?.startSubscriptionCheck ?? null };
  };
  return { checkActivity, checksLoading: enabled && !checkActivity && !error, checksError: error ? describeQueryError(error) : null, startingCheck, startCheck, refreshChecks };
}

export function performerCheckRun(activity: SubscriptionCheckActivityView | null | undefined, id: string) {
  return activity?.performers.find(item => item.performerId === id);
}
