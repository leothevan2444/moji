import { describe, expect, it, vi } from "vitest";
import { graphcacheKeys, graphcacheUpdates } from "./cache";

function fakeCache(initial: Record<string, unknown> = {}, inspectedFields: Array<{ fieldName: string; arguments?: Record<string, unknown>; }> = []) {
  const fields = new Map(Object.entries(initial));
  const links = vi.fn((entity: string, field: string, value: unknown) => fields.set(`${entity}.${field}`, value));
  const cache = {
    keyOfEntity(entity: any) {
      if (typeof entity === "string") return entity;
      if (entity.__typename === "StashPerformerScene") return `StashPerformerScene:${entity.key}`;
      if (entity.__typename === "SubscribedPerformer") return `SubscribedPerformer:${entity.performer.id}`;
      return entity.id ? `${entity.__typename}:${entity.id}` : null;
    },
    resolve(entity: string, field: string) { return fields.get(`${entity}.${field}`) ?? null; },
    link: links,
    invalidate: vi.fn(),
    writeFragment: vi.fn(),
    inspectFields: vi.fn(() => inspectedFields)
  };
  return { cache: cache as any, fields, links };
}

const updates = graphcacheUpdates as any;

describe("graphcache configuration", () => {
  it("keeps the restarted task in the list and invalidates cached Stash scenes", () => {
    const { cache, fields } = fakeCache({ "Query.tasks": ["Task:task-1"] }, [
      { fieldName: "taskStashScenes", arguments: { id: "task-1" } }
    ]);
    updates.Mutation.restartTaskSourcing({ restartTaskSourcing: { __typename: "Task", id: "task-1", stage: "SOURCING" } }, {}, cache);
    expect(fields.get("Query.tasks")).toEqual(["Task:task-1"]);
    expect(cache.invalidate).toHaveBeenCalledWith("Query", "taskStashScenes", { id: "task-1" });
  });
  it("uses stable singleton and composite keys", () => {
    expect(graphcacheKeys.Settings?.({} as any)).toBe("singleton");
    expect(graphcacheKeys.StashPerformerScene?.({ key: "scene-1" } as any)).toBe("scene-1");
    expect(graphcacheKeys.MatchedStashBox?.({ endpoint: "https://stashbox", performerId: "p1" } as any)).toBe("https://stashbox:p1");
    expect(graphcacheKeys.JackettSearchResult?.({ trackerId: "t1", infoHash: "hash" } as any)).toBe("t1:hash");
  });

  it("inserts and removes tasks from the shared root list", () => {
    const { cache, fields } = fakeCache({ "Query.tasks": ["Task:old"] });
    updates.Subscription.taskEvents({ taskEvents: { type: "CREATED", taskId: "new", task: { __typename: "Task", id: "new" } } }, {}, cache);
    expect(fields.get("Query.tasks")).toEqual(["Task:new", "Task:old"]);

    updates.Subscription.taskEvents({ taskEvents: { type: "DELETED", taskId: "old", task: null } }, {}, cache);
    expect(fields.get("Query.tasks")).toEqual(["Task:new"]);
    expect(cache.invalidate).toHaveBeenCalledWith("Task:old");
  });

  it("updates performer scene task summaries and invalidates scene queries when membership can change", () => {
    const { cache } = fakeCache({ "Query.tasks": ["Task:task-1"] }, [
      { fieldName: "stashPerformerScenes", arguments: { id: "p1", input: { page: 1 } } },
      { fieldName: "stashPerformerDetail", arguments: { id: "p1" } },
      { fieldName: "tasks" }
    ]);
    updates.Subscription.taskEvents({ taskEvents: { type: "UPDATED", taskId: "task-1", task: { id: "task-1", stage: "DOWNLOADING", stageStatus: "RUNNING", stageLabel: "Downloading", stageStatusLabel: "Running", progress: 0.5 } } }, {}, cache);
    expect(cache.writeFragment).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ __typename: "PerformerSceneTask", id: "task-1", progress: 0.5 }));
    expect(cache.invalidate).not.toHaveBeenCalledWith("Query", "stashPerformerScenes", expect.anything());

    updates.Subscription.taskEvents({ taskEvents: { type: "CREATED", taskId: "task-2", task: { __typename: "Task", id: "task-2" } } }, {}, cache);
    expect(cache.invalidate).toHaveBeenCalledWith("Query", "stashPerformerScenes", { id: "p1", input: { page: 1 } });
    expect(cache.invalidate).toHaveBeenCalledWith("Query", "stashPerformerDetail", { id: "p1" });
    cache.invalidate.mockClear();
    updates.Subscription.taskEvents({ taskEvents: { type: "UPDATED", taskId: "task-1", task: { id: "task-1", stage: "COMPLETED" } } }, {}, cache);
    expect(cache.invalidate).toHaveBeenCalledWith("Query", "stashPerformerDetail", { id: "p1" });
  });

  it("updates subscribed performer membership without broad query refreshes", () => {
    const { cache, fields } = fakeCache({ "Query.subscribedPerformers": [] });
    const performer = { __typename: "SubscribedPerformer", performer: { id: "p1" } };
    updates.Mutation.subscribePerformer({ subscribePerformer: performer }, {}, cache);
    expect(fields.get("Query.subscribedPerformers")).toEqual(["SubscribedPerformer:p1"]);

    fields.set("SubscribedPerformer:p1.performer", "StashPerformer:p1");
    fields.set("StashPerformer:p1.id", "p1");
    updates.Mutation.unsubscribePerformer({}, { stashPerformerID: "p1" }, cache);
    expect(fields.get("Query.subscribedPerformers")).toEqual([]);
    expect(cache.writeFragment).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ id: "p1", subscribed: false }));
  });

  it("does not add an unsubscribed actor to subscriptions after a manual check", () => {
    const { cache, fields } = fakeCache({ "Query.subscribedPerformers": ["SubscribedPerformer:p1"], "SubscribedPerformer:p1.performer": "StashPerformer:p1", "StashPerformer:p1.id": "p1" });
    updates.Subscription.performerSubscriptionEvents({ performerSubscriptionEvents: { type: "UPDATED", performerId: "p1", state: { __typename: "SubscribedPerformer", performer: { id: "p1", subscribed: false } } } }, {}, cache);
    expect(fields.get("Query.subscribedPerformers")).toEqual([]);
    expect(cache.writeFragment).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ id: "p1", subscribed: false }));
    updates.Mutation.refreshSubscriptionsNow({ refreshSubscriptionsNow: [{ __typename: "SubscribedPerformer", performer: { id: "p1", subscribed: false } }] }, {}, cache);
    expect(fields.get("Query.subscribedPerformers")).toEqual([]);
  });

  it("refreshes filtered and sorted performer pages after subscription changes", () => {
    const filtered = { options: { filter: "SUBSCRIBED", sort: "DEFAULT" } };
    const sorted = { options: { filter: "ALL", sort: "LAST_CHECKED" } };
    const { cache } = fakeCache({}, [
      { fieldName: "performerWorkspace", arguments: filtered },
      { fieldName: "performerWorkspace", arguments: sorted },
      { fieldName: "performerWorkspace", arguments: { options: { filter: "ALL", sort: "DEFAULT" } } }
    ]);
    updates.Mutation.subscribePerformer({ subscribePerformer: { __typename: "SubscribedPerformer", performer: { id: "p1" } } }, {}, cache);
    expect(cache.invalidate).toHaveBeenCalledTimes(2);
    expect(cache.invalidate).toHaveBeenCalledWith("Query", "performerWorkspace", filtered);
    expect(cache.invalidate).toHaveBeenCalledWith("Query", "performerWorkspace", sorted);
  });

  it("applies performer subscription events idempotently", () => {
    const { cache, fields } = fakeCache({ "Query.subscribedPerformers": [] });
    const state = { __typename: "SubscribedPerformer", performer: { id: "p1" } };
    updates.Subscription.performerSubscriptionEvents({ performerSubscriptionEvents: { type: "CREATED", performerId: "p1", state } }, {}, cache);
    updates.Subscription.performerSubscriptionEvents({ performerSubscriptionEvents: { type: "UPDATED", performerId: "p1", state } }, {}, cache);
    expect(fields.get("Query.subscribedPerformers")).toEqual(["SubscribedPerformer:p1"]);

    fields.set("SubscribedPerformer:p1.performer", "StashPerformer:p1");
    fields.set("StashPerformer:p1.id", "p1");
    updates.Subscription.performerSubscriptionEvents({ performerSubscriptionEvents: { type: "DELETED", performerId: "p1", state: null } }, {}, cache);
    expect(fields.get("Query.subscribedPerformers")).toEqual([]);
  });

  it("updates performer workspace membership for batch mutations", () => {
    const parent = "PerformerWorkspaceSnapshot:singleton.subscribedPerformers";
    const { cache, fields } = fakeCache({ [parent]: [] });
    const state = { __typename: "SubscribedPerformer", performer: { id: "p1" } };
    updates.Mutation.subscribePerformers({ subscribePerformers: { results: [{ status: "SUCCEEDED", performerId: "p1", state }] } }, {}, cache);
    expect(fields.get(parent)).toEqual(["SubscribedPerformer:p1"]);

    fields.set("SubscribedPerformer:p1.performer", "StashPerformer:p1");
    fields.set("StashPerformer:p1.id", "p1");
    updates.Mutation.unsubscribePerformers({ unsubscribePerformers: { results: [{ status: "SUCCEEDED", performerId: "p1" }] } }, {}, cache);
    expect(fields.get(parent)).toEqual([]);
  });

  it("links queued scene results to normalized tasks", () => {
    const { cache, links } = fakeCache({ "Query.tasks": [] });
    updates.Mutation.queuePerformerScenes({
      queuePerformerScenes: {
        results: [{ key: "scene-1", task: { __typename: "Task", id: "task-1" } }]
      }
    }, {}, cache);
    expect(links).toHaveBeenCalledWith("StashPerformerScene:scene-1", "mojiTask", "Task:task-1");
    expect(links).toHaveBeenCalledWith("Query", "tasks", ["Task:task-1"]);
  });
});
