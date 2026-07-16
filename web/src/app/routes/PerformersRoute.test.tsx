// @vitest-environment jsdom

import { act, fireEvent, render, renderHook, screen, waitFor } from "@testing-library/react";
import { Outlet, RouterProvider, createMemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { buildPerformerSceneQueueInput, type SceneSnapshotSource, usePerformerSceneSelection } from "./performerSceneSelection";

const mocks = vi.hoisted(() => ({
  pushToast: vi.fn(),
  queuePerformerScenes: vi.fn(),
  workspace: vi.fn()
}));

vi.mock("urql", () => ({
  useQuery: () => [{ data: { settings: { stash: { url: "http://stash.example" } } } }, vi.fn()]
}));

vi.mock("../../hooks/usePerformersWorkspace", () => ({
  usePerformersWorkspace: (options: unknown) => mocks.workspace(options)
}));

import i18n from "../../i18n/i18n";
import { Component as PerformersRoute } from "./PerformersRoute";

function scene(key: string): SceneSnapshotSource {
  return {
    key,
    sourceSceneId: `source-${key}`,
    stashBoxSceneId: `stashbox-${key}`,
    stashBoxEndpoint: "https://stashbox.example/graphql",
    code: key.toUpperCase(),
    title: `Scene ${key}`,
    inLibrary: false
  };
}

function routeScene(key: string) {
  return {
    __typename: "StashPerformerScene",
    ...scene(key),
    primarySource: "STASHBOX",
    date: null,
    studioName: null,
    performerCount: 0,
    tagCount: 0,
    imageUrl: null,
    url: null,
    matchedStashSceneId: null,
    hasStashSource: false,
    hasStashBoxSource: true,
    sourceLabels: ["StashBox"],
    performers: [],
    tags: [],
    stashIds: [],
    mojiTask: null
  };
}

function workspaceForPage(page: number) {
  const items = page === 2 ? [routeScene("three"), routeScene("four")] : [routeScene("one"), routeScene("two")];
  return {
    stashPerformerPage: null,
    stashPerformers: [],
    subscribedPerformers: [],
    performerDetail: {
      performer: { id: "performer-1", name: "Alpha", aliasList: [], favorite: false, imagePath: null, sceneCount: 4, subscribed: false },
      disambiguation: null,
      birthdate: null,
      ethnicity: null,
      country: null,
      eyeColor: null,
      heightCm: null,
      rating100: null,
      urls: [],
      matchedStashBox: null,
      totalSceneCount: 4,
      stashSceneCount: 0,
      stashBoxSceneCount: 4,
      dedupedSceneCount: 4
    },
    performerScenePage: {
      page,
      pageSize: 24,
      totalCount: 4,
      totalPages: 2,
      hasPrevPage: page > 1,
      hasNextPage: page < 2,
      stashSceneCount: 0,
      stashBoxCount: 4,
      dedupedCount: 4,
      totalCountExact: true,
      stashBoxRemoteCount: 4,
      stashBoxLoadedCount: 4,
      cacheComplete: true,
      cacheUpdatedAt: null,
      cacheStale: false,
      items
    },
    performerScenes: items,
    fetchingStashPerformers: false,
    fetchingSubscription: false,
    fetchingPerformerDetail: false,
    fetchingPerformerScenes: false,
    refreshingSubscriptionNow: false,
    subscribingPerformers: false,
    unsubscribingPerformers: false,
    refreshingSubscribedPerformers: false,
    queueingPerformerScenes: false,
    subscriptionError: undefined,
    stashPerformersError: undefined,
    performerDetailError: undefined,
    performerScenesError: undefined,
    queuePerformerScenes: mocks.queuePerformerScenes,
    queueSinglePerformerScene: vi.fn(),
    reloadSubscription: vi.fn(),
    reloadStashPerformers: vi.fn(),
    subscribePerformer: vi.fn(),
    unsubscribePerformer: vi.fn(),
    refreshSubscribedPerformer: vi.fn(),
    refreshSubscriptionsNow: vi.fn(),
    subscribePerformers: vi.fn(),
    unsubscribePerformers: vi.fn(),
    refreshSubscribedPerformers: vi.fn()
  };
}

function TestLayout() {
  return <Outlet context={{ pushToast: mocks.pushToast, copyText: vi.fn(), openHelp: vi.fn() }} />;
}

describe("PerformersRoute cross-page scene selection", () => {
  it("submits selections from two pages exactly once and preserves them across page and filter changes", () => {
    const pageOne = [scene("one"), scene("shared")];
    const pageTwo = [scene("two"), scene("shared")];
    const { result, rerender } = renderHook(
      ({ performerId, visible }) => usePerformerSceneSelection(performerId, visible),
      { initialProps: { performerId: "performer-1", visible: pageOne } }
    );

    act(() => result.current.addVisible(["one", "shared"]));
    rerender({ performerId: "performer-1", visible: pageTwo });
    expect(result.current.selectedKeys).toEqual(["one", "shared"]);

    act(() => result.current.addVisible(["two", "shared"]));
    rerender({ performerId: "performer-1", visible: [] });
    const input = buildPerformerSceneQueueInput("performer-1", result.current.selected);
    expect(input.scenes.map((item) => item.key)).toEqual(["one", "shared", "two"]);
    expect(new Set(input.scenes.map((item) => item.key)).size).toBe(3);
  });

  it("clears queued keys while retaining skipped, failed, and unreported keys", () => {
    const visible = [scene("queued"), scene("skipped"), scene("failed"), scene("unreported")];
    const { result } = renderHook(() => usePerformerSceneSelection("performer-1", visible));
    act(() => result.current.addVisible(visible.map((item) => item.key)));

    act(() => result.current.applyResults([
      { key: "queued", status: "QUEUED" },
      { key: "skipped", status: "SKIPPED" },
      { key: "failed", status: "FAILED" }
    ]));

    expect(result.current.selectedKeys).toEqual(["skipped", "failed", "unreported"]);
  });

  it("clears all scene selections when the performer changes", () => {
    const { result, rerender } = renderHook(
      ({ performerId, visible }) => usePerformerSceneSelection(performerId, visible),
      { initialProps: { performerId: "performer-1", visible: [scene("one")] } }
    );
    act(() => result.current.toggle("one"));
    expect(result.current.selectedKeys).toEqual(["one"]);

    rerender({ performerId: "performer-2", visible: [scene("two")] });
    expect(result.current.selectedKeys).toEqual([]);
  });
});

describe("PerformersRoute mutation integration", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("en");
    mocks.pushToast.mockReset();
    mocks.queuePerformerScenes.mockReset();
    mocks.workspace.mockReset();
    mocks.workspace.mockImplementation((options: any) => workspaceForPage(options.performerScenePage));
  });

  it("submits both pages through the route mutation and clears only queued scenes", async () => {
    mocks.queuePerformerScenes.mockResolvedValue({
      data: {
        queuePerformerScenes: {
          summary: { requestedCount: 4, queuedCount: 2, skippedCount: 1, failedCount: 1 },
          results: [
            { key: "one", status: "QUEUED", reasonCode: "QUEUED", message: "queued", task: null, resolvedCode: "ONE" },
            { key: "two", status: "SKIPPED", reasonCode: "ALREADY_IN_LIBRARY", message: "skipped", task: null, resolvedCode: "TWO" },
            { key: "three", status: "FAILED", reasonCode: "QUEUE_FAILED", message: "failed", task: null, resolvedCode: "THREE" },
            { key: "four", status: "QUEUED", reasonCode: "QUEUED", message: "queued", task: null, resolvedCode: "FOUR" }
          ]
        }
      },
      error: undefined
    });

    const router = createMemoryRouter([
      {
        element: <TestLayout />,
        children: [{ path: "/performers/:performerId", element: <PerformersRoute /> }]
      }
    ], { initialEntries: ["/performers/performer-1"] });
    render(<RouterProvider router={router} />);

    fireEvent.click(await screen.findByRole("button", { name: "Select this page (2)" }));
    expect(screen.getByRole("button", { name: "Create download tasks (2)" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Select this page (2)" })).toBeTruthy());
    fireEvent.click(screen.getByRole("button", { name: "Select this page (2)" }));
    fireEvent.click(screen.getByRole("button", { name: "Create download tasks (4)" }));

    await waitFor(() => expect(mocks.queuePerformerScenes).toHaveBeenCalledTimes(1));
    expect(mocks.queuePerformerScenes).toHaveBeenCalledWith({
      input: {
        performerId: "performer-1",
        scenes: ["one", "two", "three", "four"].map((key) => expect.objectContaining({ key }))
      }
    });
    await waitFor(() => expect(screen.getByRole("button", { name: "Create download tasks (2)" })).toBeTruthy());
    expect(screen.getByRole("button", { name: "Deselect THREE" }).getAttribute("aria-pressed")).toBe("true");
    expect(screen.getByRole("button", { name: "Select FOUR" }).getAttribute("aria-pressed")).toBe("false");
  });
});
