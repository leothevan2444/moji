// @vitest-environment jsdom

import { act, cleanup, fireEvent, render, renderHook, screen, waitFor, within } from "@testing-library/react";
import { Outlet, RouterProvider, createMemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { buildPerformerSceneQueueInput, type SceneSnapshotSource, usePerformerSceneSelection } from "./performerSceneSelection";

const mocks = vi.hoisted(() => ({
  pushToast: vi.fn(),
  queuePerformerScenes: vi.fn(),
  queueSinglePerformerScene: vi.fn(),
  startCheck: vi.fn(),
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

afterEach(() => cleanup());

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
      activeTaskCount: 0,
      sceneStatsComplete: true,
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
    queueSinglePerformerScene: mocks.queueSinglePerformerScene,
    reloadSubscription: vi.fn(),
    reloadStashPerformers: vi.fn(),
    subscribePerformer: vi.fn(),
    unsubscribePerformer: vi.fn(),
    refreshSubscribedPerformer: vi.fn(),
    refreshSubscriptionsNow: vi.fn(),
    subscribePerformers: vi.fn(),
    unsubscribePerformers: vi.fn(),
    startCheck: mocks.startCheck,
    refreshChecks: vi.fn(),
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
    expect(input.sceneKeys).toEqual(["one", "shared", "two"]);
    expect(new Set(input.sceneKeys).size).toBe(3);
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
    mocks.queueSinglePerformerScene.mockReset();
    mocks.startCheck.mockReset();
    mocks.startCheck.mockResolvedValue({ run: { id: "check-1" } });
    mocks.workspace.mockReset();
    mocks.workspace.mockImplementation((options: any) => workspaceForPage(options.performerScenePage));
  });

  it("confirms an unsubscribed actor check before submitting and leaves cancellation harmless", async () => {
    const router = createMemoryRouter([{ element: <TestLayout />, children: [{ path: "/performers/:performerId", element: <PerformersRoute /> }] }], { initialEntries: ["/performers/performer-1"] });
    render(<RouterProvider router={router} />);
    fireEvent.click(screen.getByRole("button", { name: "Check Alpha for updates" }));
    let dialog = await screen.findByRole("dialog");
    expect(await within(dialog).findByText("Check Alpha for updates?")).toBeTruthy();
    expect(within(dialog).getByText(/one-time manual check/)).toBeTruthy();
    expect(mocks.startCheck).not.toHaveBeenCalled();
    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(mocks.startCheck).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Check Alpha for updates" }));
    dialog = await screen.findByRole("dialog");
    fireEvent.click(await within(dialog).findByRole("button", { name: "Start check" }));
    await waitFor(() => expect(mocks.startCheck).toHaveBeenCalledExactlyOnceWith(["performer-1"]));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("includes unsubscribed actors in an explicitly confirmed batch", async () => {
    mocks.workspace.mockImplementation((options: any) => {
      const view = workspaceForPage(options.performerScenePage);
      const first = view.performerDetail.performer;
      const second = { ...first, id: "performer-2", name: "Beta", subscribed: true };
      return { ...view, stashPerformers: [first, second], subscribedPerformers: [{ performer: second, recentReleases: [] }] };
    });
    const router = createMemoryRouter([{ element: <TestLayout />, children: [{ path: "/performers", element: <PerformersRoute /> }] }], { initialEntries: ["/performers"] });
    render(<RouterProvider router={router} />);
    fireEvent.click(screen.getByRole("button", { name: "Select multiple performers" }));
    const actions = within(screen.getByRole("region", { name: "Selected performer actions" }));
    fireEvent.click(actions.getByRole("button", { name: "Select this page (2)" }));
    fireEvent.click(actions.getByRole("button", { name: "Check selected (2)" }));
    const dialog = await screen.findByRole("dialog");
    expect(await within(dialog).findByText(/Unsubscribed performers: 1/)).toBeTruthy();
    expect(mocks.startCheck).not.toHaveBeenCalled();
    fireEvent.click(within(dialog).getByRole("button", { name: "Start check" }));
    await waitFor(() => expect(mocks.startCheck).toHaveBeenCalledExactlyOnceWith(["performer-1", "performer-2"]));
  });

  it("checks a subscribed actor immediately", async () => {
    mocks.workspace.mockImplementation((options: any) => { const view = workspaceForPage(options.performerScenePage); view.performerDetail.performer.subscribed = true; return view; });
    const router = createMemoryRouter([{ element: <TestLayout />, children: [{ path: "/performers/:performerId", element: <PerformersRoute /> }] }], { initialEntries: ["/performers/performer-1"] });
    render(<RouterProvider router={router} />);
    fireEvent.click(screen.getByRole("button", { name: "Check Alpha for updates" }));
    await waitFor(() => expect(mocks.startCheck).toHaveBeenCalledExactlyOnceWith(["performer-1"]));
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("submits both pages through the route mutation and clears only queued scenes", async () => {
    mocks.queuePerformerScenes.mockResolvedValue({
      data: {
        queuePerformerScenes: {
          summary: { requestedCount: 4, queuedCount: 2, skippedCount: 1, failedCount: 1 },
          results: [
            { key: "one", status: "QUEUED", reasonCode: "QUEUED", task: null, resolvedCode: "ONE" },
            { key: "two", status: "SKIPPED", reasonCode: "ALREADY_IN_LIBRARY", task: null, resolvedCode: "TWO" },
            { key: "three", status: "FAILED", reasonCode: "QUEUE_FAILED", task: null, resolvedCode: "THREE" },
            { key: "four", status: "QUEUED", reasonCode: "QUEUED", task: null, resolvedCode: "FOUR" }
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

    fireEvent.click(await screen.findByRole("button", { name: "Select scenes" }));
    fireEvent.click(screen.getByRole("button", { name: "Select this page (2)" }));
    expect(screen.getByRole("button", { name: "Create download tasks (2)" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Select this page (2)" })).toBeTruthy());
    fireEvent.click(screen.getByRole("button", { name: "Select this page (2)" }));
    fireEvent.click(screen.getByRole("button", { name: "Create download tasks (4)" }));

    await waitFor(() => expect(mocks.queuePerformerScenes).toHaveBeenCalledTimes(1));
    expect(mocks.queuePerformerScenes).toHaveBeenCalledWith({
      input: {
        performerId: "performer-1",
        sceneKeys: ["one", "two", "three", "four"]
      }
    });
    await waitFor(() => expect(screen.getByRole("button", { name: "Create download tasks (2)" })).toBeTruthy());
    expect(screen.getByRole("button", { name: "Deselect THREE" }).getAttribute("aria-pressed")).toBe("true");
    expect(screen.getByRole("button", { name: "Select FOUR" }).getAttribute("aria-pressed")).toBe("false");
  });

  it("keeps selection controls behind an explicit mode and resets them on exit and performer change", async () => {
    const router = createMemoryRouter([
      { element: <TestLayout />, children: [{ path: "/performers/:performerId", element: <PerformersRoute /> }] }
    ], { initialEntries: ["/performers/performer-1"] });
    render(<RouterProvider router={router} />);

    expect(screen.queryByRole("button", { name: "Select this page (2)" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Select ONE" })).toBeNull();
    fireEvent.click(screen.getByText("ONE").closest("article")!);
    expect(screen.queryByText("1 scenes selected (kept across pages)")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Select scenes" }));
    fireEvent.click(screen.getByRole("button", { name: "Select ONE" }));
    expect(screen.getByRole("button", { name: "Create download tasks (1)" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Exit selection" }));
    expect(screen.queryByRole("button", { name: "Create download tasks (1)" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Select ONE" })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Select scenes" }));
    expect(screen.getByRole("button", { name: "Select ONE" }).getAttribute("aria-pressed")).toBe("false");
    fireEvent.click(screen.getByRole("button", { name: "Select ONE" }));
    await act(async () => { await router.navigate("/performers/performer-2"); });
    expect(screen.getByRole("button", { name: "Select scenes" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Select this page (2)" })).toBeNull();
  });

  it("submits a key-only single-scene request and localizes its reason code", async () => {
    mocks.queueSinglePerformerScene.mockResolvedValue({
      data: {
        queuePerformerScenes: {
          results: [{
            key: "one",
            status: "SKIPPED",
            reasonCode: "NO_STASHBOX_SOURCE",
            task: null,
            resolvedCode: "ONE"
          }]
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

    fireEvent.click((await screen.findAllByRole("button", { name: "Create task" }))[0]);

    await waitFor(() => expect(mocks.queueSinglePerformerScene).toHaveBeenCalledWith({
      input: { performerId: "performer-1", sceneKeys: ["one"] }
    }));
    expect(mocks.pushToast).toHaveBeenCalledWith(
      "tone-info",
      "No downloadable StashBox source is available for this scene."
    );
  });
});
