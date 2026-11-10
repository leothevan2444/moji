import { useCallback } from "react";
import { useQuery } from "urql";
import { PerformerActivityDocument } from "../graphql/generated/graphql";
import { useSubscriptionCheckPolling } from "./useSubscriptionCheckPolling";

export function usePerformerActivity(id: string | null, checking: boolean) {
 const [result, refresh] = useQuery({ query: PerformerActivityDocument, variables: {id:id ?? ""}, pause:!id, requestPolicy: "network-only" });
 const reload = useCallback(() => refresh({requestPolicy:"network-only"}),[refresh]);
 useSubscriptionCheckPolling(Boolean(id),checking,reload);
 return { activity: result.data?.performerActivity.id === id ? result.data.performerActivity : undefined, error:result.operation?.variables.id === id ? result.error : undefined, reload };
}
