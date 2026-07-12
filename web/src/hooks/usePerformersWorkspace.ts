import type { LibraryFilter, SceneSourceFilter } from "../graphql/generated/graphql";
import { usePerformers } from "./usePerformers";
import { usePerformerScenes } from "./usePerformerScenes";
import { useQueuePerformerScenes } from "./useQueuePerformerScenes";
import { useSubscriptions } from "./useSubscriptions";

interface Options {
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
  const subscriptions = useSubscriptions(options.enabled);
  const queue = useQueuePerformerScenes();

  const reloadSubscription = async () => {
    await Promise.all([
      subscriptions.refreshSubscription({ requestPolicy: "network-only" }),
      performers.refreshStashPerformers({ requestPolicy: "network-only" }),
      performers.refreshPerformerDetail({ requestPolicy: "network-only" }),
      scenes.refreshPerformerScenes({ requestPolicy: "network-only" })
    ]);
  };

  return { ...performers, ...scenes, ...subscriptions, ...queue, reloadSubscription };
}
