import { gql } from "@urql/core";
import { cacheExchange } from "@urql/exchange-graphcache";
import type { Cache, Data, KeyingConfig, UpdatesConfig } from "@urql/exchange-graphcache";

const PERFORMER_SUBSCRIBED_FRAGMENT = gql`
  fragment PerformerSubscribedCacheValue on StashPerformer {
    subscribed
  }
`;

const PERFORMER_SCENE_TASK_FRAGMENT = gql`
  fragment PerformerSceneTaskEventValue on PerformerSceneTask {
    id
    stage
    stageStatus
    stageLabel
    stageStatusLabel
    progress
  }
`;

type Entity = Data & { __typename: string };

function existingLinks(cache: Cache, fieldName: string): string[] | null {
  const value = cache.resolve("Query", fieldName);
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === "string") : null;
}

function linkTask(cache: Cache, task: Entity | null | undefined, prepend = false) {
  if (!task) return;
  const taskKey = cache.keyOfEntity(task);
  const links = existingLinks(cache, "tasks");
  if (!taskKey || !links) return;
  const next = links.filter((key) => key !== taskKey);
  cache.link("Query", "tasks", prepend ? [taskKey, ...next] : links.includes(taskKey) ? links : [taskKey, ...links]);
}

function unlinkTask(cache: Cache, id: string) {
  const taskKey = cache.keyOfEntity({ __typename: "Task", id });
  const links = existingLinks(cache, "tasks");
  if (taskKey && links) cache.link("Query", "tasks", links.filter((key) => key !== taskKey));
  if (taskKey) cache.invalidate(taskKey);
}

function updatePerformerSceneTask(cache: Cache, task: any) {
  if (!task?.id) return;
  cache.writeFragment(PERFORMER_SCENE_TASK_FRAGMENT, {
    __typename: "PerformerSceneTask",
    id: task.id,
    stage: task.stage,
    stageStatus: task.stageStatus,
    stageLabel: task.stageLabel,
    stageStatusLabel: task.stageStatusLabel,
    progress: task.progress
  });
}

function invalidatePerformerSceneQueries(cache: Cache) {
  for (const field of cache.inspectFields("Query")) {
    if (field.fieldName === "stashPerformerScenes") cache.invalidate("Query", field.fieldName, field.arguments ?? {});
  }
}

function subscribedPerformerID(cache: Cache, link: string): string | null {
  const performerLink = cache.resolve(link, "performer");
  if (typeof performerLink !== "string") return null;
  const id = cache.resolve(performerLink, "id");
  return typeof id === "string" ? id : null;
}

function replaceSubscribedPerformers(cache: Cache, performers: Entity[]) {
  const current = existingLinks(cache, "subscribedPerformers");
  if (!current) return;
  const links = performers.map((item) => cache.keyOfEntity(item)).filter((key): key is string => Boolean(key));
  cache.link("Query", "subscribedPerformers", links);
}

function upsertSubscribedPerformer(cache: Cache, performer: Entity | null | undefined) {
  if (!performer) return;
  const key = cache.keyOfEntity(performer);
  const links = existingLinks(cache, "subscribedPerformers");
  if (!key || !links) return;
  cache.link("Query", "subscribedPerformers", [key, ...links.filter((item) => item !== key)]);
}

function removeSubscribedPerformer(cache: Cache, id: string) {
  const links = existingLinks(cache, "subscribedPerformers");
  if (links) cache.link("Query", "subscribedPerformers", links.filter((link) => subscribedPerformerID(cache, link) !== id));
  cache.writeFragment(PERFORMER_SUBSCRIBED_FRAGMENT, { __typename: "StashPerformer", id, subscribed: false });
}

function linkQueuedSceneTasks(cache: Cache, payload: any) {
  for (const item of payload?.results ?? []) {
    if (!item?.key || !item?.task) continue;
    const sceneKey = cache.keyOfEntity({ __typename: "StashPerformerScene", key: item.key });
    const taskKey = cache.keyOfEntity(item.task);
    if (sceneKey && taskKey) cache.link(sceneKey, "mojiTask", taskKey);
    linkTask(cache, item.task, true);
  }
}

const singleton = () => "singleton";

export const graphcacheKeys: KeyingConfig = {
  Settings: singleton,
  StashSettings: singleton,
  IngestSettings: singleton,
  DownloadsIngestSettings: singleton,
  LibraryIngestSettings: singleton,
  TransferIngestSettings: singleton,
  JackettSettings: singleton,
  QBittorrentSettings: singleton,
  AutomationSettings: singleton,
  SubscriptionReleasePolicy: singleton,
  TorrentSelectionSettings: singleton,
  SystemSettings: singleton,
  ImageCacheSettings: singleton,
  SettingsStatus: singleton,
  IngestStatus: singleton,
  StashStats: singleton,
  JackettStats: singleton,
  QBittorrentStats: singleton,
  AutomationStatus: singleton,
  StashBoxStatus: singleton,
  ImageCacheStatus: singleton,
  DashboardStats: singleton,
  StashPerformerScene: (data) => typeof data.key === "string" ? data.key : null,
  DiscoveredScene: (data) => typeof data.key === "string" ? data.key : null,
  SubscriptionRelease: (data) => typeof data.key === "string" ? data.key : null,
  QueuePerformerSceneResult: (data) => typeof data.key === "string" ? data.key : null,
  SubscribedPerformer: (data) => {
    const performer = data.performer as { id?: unknown } | undefined;
    return typeof performer?.id === "string" ? performer.id : null;
  },
  StashPerformerDetail: (data) => {
    const performer = data.performer as { id?: unknown } | undefined;
    return typeof performer?.id === "string" ? performer.id : null;
  },
  MatchedStashBox: (data) => typeof data.endpoint === "string" && typeof data.performerId === "string" ? `${data.endpoint}:${data.performerId}` : null,
  StashSceneID: (data) => typeof data.endpoint === "string" && typeof data.stashId === "string" ? `${data.endpoint}:${data.stashId}` : null,
  JackettSearchResult: (data) => {
    if (typeof data.trackerId !== "string") return null;
    const identity = typeof data.infoHash === "string" && data.infoHash ? data.infoHash : data.link;
    return typeof identity === "string" && identity ? `${data.trackerId}:${identity}` : null;
  },
  StashBoxEndpoint: (data) => typeof data.endpoint === "string" ? data.endpoint : null,
  StashLibrary: (data) => typeof data.path === "string" ? data.path : null,
  LogEntry: (data) => typeof data.sequence === "number" ? String(data.sequence) : null
};

export const graphcacheUpdates: UpdatesConfig = {
  Subscription: {
    taskEvents(result: any, _args, cache) {
      const event = result.taskEvents;
      if (!event) return;
      if (event.type === "DELETED") unlinkTask(cache, event.taskId);
      else {
        linkTask(cache, event.task, event.type === "CREATED");
        updatePerformerSceneTask(cache, event.task);
      }
      if (event.type === "CREATED" || event.type === "DELETED") invalidatePerformerSceneQueries(cache);
    },
    performerSubscriptionEvents(result: any, _args, cache) {
      const event = result.performerSubscriptionEvents;
      if (!event) return;
      if (event.type === "DELETED") removeSubscribedPerformer(cache, event.performerId);
      else upsertSubscribedPerformer(cache, event.state);
    }
  },
  Mutation: {
    addTorrent(result: any, _args, cache) { linkTask(cache, result.addTorrent, true); },
    downloadMedia(result: any, _args, cache) { linkTask(cache, result.downloadMedia, true); },
    queueDiscoveredScene(result: any, _args, cache) { linkTask(cache, result.queueDiscoveredScene, true); },
    retryTask(result: any, _args, cache) { linkTask(cache, result.retryTask); },
    resolveBlockedSourcingTask(result: any, _args, cache) { linkTask(cache, result.resolveBlockedSourcingTask); },
    deleteTask(result: any, args: any, cache) { unlinkTask(cache, result.deleteTask?.id ?? args.id); },
    subscribePerformer(result: any, _args, cache) { upsertSubscribedPerformer(cache, result.subscribePerformer); },
    unsubscribePerformer(_result: any, args: any, cache) { removeSubscribedPerformer(cache, args.stashPerformerID); },
    refreshSubscribedPerformer(result: any, _args, cache) { upsertSubscribedPerformer(cache, result.refreshSubscribedPerformer); },
    refreshSubscriptionsNow(result: any, _args, cache) { replaceSubscribedPerformers(cache, result.refreshSubscriptionsNow ?? []); },
    queuePerformerScenes(result: any, _args, cache) { linkQueuedSceneTasks(cache, result.queuePerformerScenes); }
  }
};

export const graphcacheExchange = cacheExchange({
  keys: graphcacheKeys,
  updates: graphcacheUpdates
});
