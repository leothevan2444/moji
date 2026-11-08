import { useCallback, useEffect, useState } from "react";
import { useQuery } from "urql";
import { useIndexerEvents } from "./useIndexerEvents";
import { HomeIndexerSummaryDocument, IndexerEventKind, type HomeIndexerSummaryQuery } from "../graphql/generated/graphql";

export function useHomeIndexers(hours = 72) {
  const [{ data, error, fetching }, requery] = useQuery({ query: HomeIndexerSummaryDocument, variables: {hours}, requestPolicy: "network-only" });
  const [snapshot, setSnapshot] = useState<HomeIndexerSummaryQuery["homeIndexerSummary"] | null>(null);
  const read = useCallback(() => requery({ requestPolicy: "network-only" }), [requery]);
  const events = useIndexerEvents(read, [IndexerEventKind.Configuration, IndexerEventKind.Catalog, IndexerEventKind.Site]);
  const refresh = useCallback(() => { read(); if (events.error) events.retry(); }, [read, events.error, events.retry]);
  useEffect(() => { if (data?.homeIndexerSummary?.hours === hours) setSnapshot(data.homeIndexerSummary); }, [data, hours]);
  useEffect(() => {
    const visible = () => { if (!document.hidden) refresh(); };
    document.addEventListener("visibilitychange", visible);
    return () => document.removeEventListener("visibilitychange", visible);
  }, [refresh]);
  return { snapshot, error: error ?? events.error, fetching, refresh };
}
