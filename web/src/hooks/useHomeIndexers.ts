import { useCallback, useEffect, useState } from "react";
import { useQuery, useSubscription } from "urql";
import { useIndexerEvents } from "./useIndexerEvents";
import { HomeIndexerSummaryDocument, HomeIndexerDefinitionEventsDocument, IndexerEventKind, type HomeIndexerSummaryQuery } from "../graphql/generated/graphql";

export function useHomeIndexers() {
  const [{ data, error, fetching }, requery] = useQuery({ query: HomeIndexerSummaryDocument, requestPolicy: "network-only" });
  const [{ data: definitions, error: definitionError }, restartDefinitions] = useSubscription({ query: HomeIndexerDefinitionEventsDocument });
  const [snapshot, setSnapshot] = useState<HomeIndexerSummaryQuery["homeIndexerSummary"] | null>(null);
  const read = useCallback(() => requery({ requestPolicy: "network-only" }), [requery]);
  const events = useIndexerEvents(read, [IndexerEventKind.Configuration, IndexerEventKind.Catalog, IndexerEventKind.Site]);
  const refresh = useCallback(() => { read(); if (events.error) events.retry(); restartDefinitions({ pause: false }); }, [read, events.error, events.retry, restartDefinitions]);
  useEffect(() => { if (data?.homeIndexerSummary) setSnapshot(data.homeIndexerSummary); }, [data]);
  useEffect(() => {
    if (!definitions) return;
    events.schedule();
  }, [definitions, events.schedule]);
  useEffect(() => {
    const visible = () => { if (!document.hidden) refresh(); };
    document.addEventListener("visibilitychange", visible);
    return () => document.removeEventListener("visibilitychange", visible);
  }, [refresh]);
  return { snapshot, error: error ?? events.error ?? definitionError, fetching, refresh };
}
