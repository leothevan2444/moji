import { RefreshStashPerformerScenesDocument, type LibraryFilter, type SceneSourceFilter } from "../graphql/generated/graphql";
import { useEffect, useRef } from "react";
import { useMutation } from "urql";
import { usePerformers } from "./usePerformers";
import { usePerformerScenes } from "./usePerformerScenes";
import { useQueuePerformerScenes } from "./useQueuePerformerScenes";
import { useSubscriptions } from "./useSubscriptions";

interface Options {
  options?: import("../graphql/generated/graphql").PerformerListOptions;
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

/** Composes the independent performer-page domains without hiding their boundaries. */
export function usePerformersWorkspace(options: Options) {
  const performers = usePerformers(options);
  const scenes = usePerformerScenes({
    enabled: options.enabled,
    performerId: options.performerId,
    search: options.performerSceneSearch,
    source: options.performerSceneSource,
    library: options.performerSceneLibrary,
    page: options.performerScenePage,
    pageSize: options.performerScenePageSize
  });
  const cacheRefreshAttempts = useRef(0);
  useEffect(() => { cacheRefreshAttempts.current = 0; }, [options.performerId]);
  const stale = Boolean(performers.performerSummary?.sceneStatsStale || scenes.performerScenePage?.cacheStale);
  useEffect(() => {
    if (!options.enabled || !options.performerId || !stale || performers.fetchingPerformerSummary || scenes.fetchingPerformerScenes || cacheRefreshAttempts.current >= 5) return;
    const timer = setTimeout(() => {
      cacheRefreshAttempts.current += 1;
      performers.refreshPerformerSummary({ requestPolicy: "network-only" });
      scenes.refreshPerformerScenes({ requestPolicy: "network-only" });
    }, 1000 * 2 ** cacheRefreshAttempts.current);
    return () => clearTimeout(timer);
  }, [options.enabled, options.performerId, stale, performers.fetchingPerformerSummary, scenes.fetchingPerformerScenes, performers.refreshPerformerSummary, scenes.refreshPerformerScenes]);
  const subscriptions = useSubscriptions(options.enabled, performers.subscribedPerformers, performers.reloadStashPerformers);
  const queue = useQueuePerformerScenes();
  const [{ fetching: refreshingPerformerCache }, refreshPerformerCache] = useMutation(RefreshStashPerformerScenesDocument);

  const reloadSubscription = async () => {
    if (options.performerId) {
      const cacheResult = await refreshPerformerCache({ id: options.performerId, input: { search: options.performerSceneSearch, source: options.performerSceneSource, inLibrary: options.performerSceneLibrary, page: options.performerScenePage, pageSize: options.performerScenePageSize } });
      performers.refreshPerformerDetail({ requestPolicy: "network-only" });
      performers.refreshPerformerSummary({ requestPolicy: "network-only" });
      scenes.refreshPerformerScenes({ requestPolicy: "network-only" });
      return [cacheResult];
    }
    return [await performers.reloadStashPerformers()];
  };

  return { ...performers, ...scenes, ...subscriptions, ...queue, fetchingSubscription: performers.fetchingStashPerformers, subscriptionError: performers.stashPerformersError, refreshingPerformerCache, reloadSubscription };
}
