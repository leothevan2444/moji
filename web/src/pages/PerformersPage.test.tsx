// @vitest-environment jsdom
import { MemoryRouter } from "react-router";

import { fireEvent, render as baseRender, screen, within } from "@testing-library/react";
import type { ComponentProps } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import userEvent from "@testing-library/user-event";
import { LibraryFilter, SceneSourceFilter, TaskStage, TaskStageStatus } from "../graphql/generated/graphql";
import i18n from "../i18n/i18n";
import { PerformerSceneCard } from "../components/performers/PerformerSceneCard";
import type { StashPerformerSceneEntry } from "../components/performers/types";
import { PerformersPage } from "./PerformersPage";

const render = (ui: React.ReactNode) => baseRender(ui, { wrapper: MemoryRouter });

function props(overrides: Partial<ComponentProps<typeof PerformersPage>> = {}): ComponentProps<typeof PerformersPage> {
  const noop = vi.fn();
  return {
    stashBaseURL: null,
    stashPerformerPage: { __typename: "StashPerformerConnection", page: 1, pageSize: 24, totalCount: 1, totalPages: 1, hasPrevPage: false, hasNextPage: false, items: [] },
    stashPerformers: [{ __typename: "StashPerformer", id: "p1", name: "Alpha", aliasList: [], favorite: false, imagePath: null, sceneCount: 3, subscribed: false }],
    performerDetail: null,
    performerScenePage: null,
    performerScenes: [],
    subscribedPerformers: [],
    fetchingStashPerformers: false,
    fetchingPerformerDetail: false,
    fetchingPerformerScenes: false,
    fetchingSubscription: false,
    refreshingSubscriptionNow: false,
    refreshingList: false,
    performerBatchPending: false,
    queueingPerformerScenes: false,
    subscriptionSearch: "",
    subscriptionPageSize: 24,
    selectedPerformerId: null,
    performerSceneSearch: "",
    performerSceneSourceFilter: SceneSourceFilter.All,
    performerSceneLibraryFilter: LibraryFilter.All,
    performerScenePageSize: 24,
    selectedSceneKeys: [],
    sceneSelectionMode: false,
    onToggleSceneSelectionMode: noop,
    selectedPerformerIds: [],
    performerSelectionMode: false,
    lastRefreshedAt: null,
    pendingSceneKeys: [],
    pendingSubscriptionID: null,
    subscriptionError: null,
    stashPerformersError: null,
    performerDetailError: null,
    performerScenesError: null,
    onSearchChange: noop,
    onPageSizeChange: noop,
    onReload: noop,
    onRefreshAll: noop,
    onTogglePerformerSelectionMode: noop,
    onTogglePerformerSelection: noop,
    onSelectVisiblePerformers: noop,
    onClearPerformerSelection: noop,
    onBatchSubscribePerformers: noop,
    onBatchUnsubscribePerformers: noop,
    onBatchRefreshPerformers: noop,
    onToggle: noop,
    onRefreshOne: noop,
    onPrevPage: noop,
    onNextPage: noop,
    onPageChange: noop,
    onOpenPerformer: noop,
    onOpenTask: noop,
    onBackToList: noop,
    onPerformerSceneSearchChange: noop,
    onPerformerSceneSourceChange: noop,
    onPerformerSceneLibraryChange: noop,
    onPerformerScenePageSizeChange: noop,
    onPrevPerformerScenePage: noop,
    onNextPerformerScenePage: noop,
    onToggleSceneSelection: noop,
    onSelectCurrentScenePage: noop,
    onClearSceneSelection: noop,
    onQueueSelectedScenes: noop,
    onQueueScene: noop,
    ...overrides
  };
}

describe("PerformersPage list selection", () => {
  beforeEach(async () => i18n.changeLanguage("en"));
  afterEach(() => document.body.replaceChildren());

  it("shows durable review counts for a performer who is no longer subscribed", () => {
    render(<PerformersPage {...props({ reviewPendingCount: 2, reviewCounts: { p1: 2 } })} />);
    expect(screen.getByRole("link", { name: "2 awaiting review" }).getAttribute("href")).toBe("/performers/review?performer=p1");
    expect(screen.getByRole("link", { name: "Release review · 2" }).getAttribute("href")).toBe("/performers/review");
  });

  it("opens a performer normally and selects it in multi-select mode", () => {
    const onOpenPerformer = vi.fn();
    const onTogglePerformerSelection = vi.fn();
    const view = render(<PerformersPage {...props({ onOpenPerformer, onTogglePerformerSelection })} />);
    fireEvent.click(screen.getByText("Alpha").closest("article")!);
    expect(onOpenPerformer).toHaveBeenCalledWith("p1");

    view.rerender(<PerformersPage {...props({ performerSelectionMode: true, onOpenPerformer, onTogglePerformerSelection })} />);
    expect((screen.getByRole("checkbox", { name: "Select Alpha" }) as HTMLInputElement).checked).toBe(false);
    fireEvent.click(screen.getByText("Alpha").closest("article")!);
    expect(onTogglePerformerSelection).toHaveBeenCalledWith("p1");
    expect(onOpenPerformer).toHaveBeenCalledTimes(1);
  });

  it("shows subscription eligibility actions for selected performers", () => {
    render(<PerformersPage {...props({ performerSelectionMode: true, selectedPerformerIds: ["p1"] })} />);
    const actions = within(screen.getByRole("region", { name: "Selected performer actions" }));
    expect((actions.getByRole("button", { name: "Subscribe (1)" }) as HTMLButtonElement).disabled).toBe(false);
    expect((actions.getByRole("button", { name: "Check selected (0)" }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.queryByRole("button", { name: "Check Alpha now" })).toBeNull();
  });

  it("shows reload and check-all as toolbar icons with tooltips", () => {
    const onRefreshAll = vi.fn();
    const pageProps = props({ onRefreshAll });
    const view = render(<PerformersPage {...pageProps} />);
    const reload = screen.getByRole("button", { name: "Reload from Stash" });
    const checkAll = screen.getByRole("button", { name: "Check all" }) as HTMLButtonElement;
    expect(reload.textContent).toBe("");
    expect(reload.title).toBe("Reload from Stash");
    expect(checkAll.textContent).toBe("");
    expect(checkAll.title).toBe("Check all");
    expect(checkAll.disabled).toBe(true);
    expect(checkAll.closest(".toolbar-inline")).toBe(reload.closest(".toolbar-inline"));
    expect(screen.queryByRole("button", { name: "More actions" })).toBeNull();

    view.rerender(<PerformersPage {...pageProps} subscribedPerformers={[{ performer: pageProps.stashPerformers[0], recentReleases: [], lastCheckedAt: null, pendingReleaseCount: 0, reviewReleaseCount: 0, processedReleaseCount: 0 }]} />);
    expect(checkAll.disabled).toBe(false);
    fireEvent.click(checkAll);
    expect(onRefreshAll).toHaveBeenCalledTimes(1);

    view.rerender(<PerformersPage {...pageProps} refreshingSubscriptionNow />);
    const checking = screen.getByRole("button", { name: "Checking..." }) as HTMLButtonElement;
    expect(checking.disabled).toBe(true);
    expect(checking.title).toBe("Checking...");
  });

  it("jumps to first, last, and valid specified pages and disables navigation while loading", () => {
    const onPageChange = vi.fn();
    const pageProps = props({
      onPageChange,
      stashPerformerPage: { ...props().stashPerformerPage!, page: 3, totalPages: 10, hasPrevPage: true, hasNextPage: true }
    });
    const view = render(<PerformersPage {...pageProps} />);
    const pagination = within(view.container.querySelector(".pagination-bar--performers")! as HTMLElement);
    fireEvent.click(pagination.getByRole("button", { name: "First page" }));
    expect(onPageChange).toHaveBeenLastCalledWith(1);
    fireEvent.click(pagination.getByRole("button", { name: "Last page" }));
    expect(onPageChange).toHaveBeenLastCalledWith(10);

    const input = pagination.getByRole("spinbutton", { name: "Page number" });
    const submit = pagination.getByRole("button", { name: "Go" }) as HTMLButtonElement;
    for (const value of ["", "0", "11", "2.5", "3"]) {
      fireEvent.change(input, { target: { value } });
      expect(submit.disabled).toBe(true);
      fireEvent.submit(input.closest("form")!);
    }
    expect(onPageChange).toHaveBeenCalledTimes(2);
    fireEvent.change(input, { target: { value: "7" } });
    expect(submit.disabled).toBe(false);
    fireEvent.submit(input.closest("form")!);
    expect(onPageChange).toHaveBeenLastCalledWith(7);

    view.rerender(<PerformersPage {...pageProps} fetchingStashPerformers />);
    for (const name of ["First page", "Previous", "Next", "Last page", "Go"]) {
      expect((pagination.getByRole("button", { name }) as HTMLButtonElement).disabled).toBe(true);
    }

    view.rerender(<PerformersPage {...pageProps} stashPerformerPage={{ ...pageProps.stashPerformerPage!, page: 1, hasPrevPage: false }} />);
    expect((pagination.getByRole("button", { name: "First page" }) as HTMLButtonElement).disabled).toBe(true);
    expect((input as HTMLInputElement).value).toBe("");
    view.rerender(<PerformersPage {...pageProps} stashPerformerPage={{ ...pageProps.stashPerformerPage!, page: 10, hasNextPage: false }} />);
    expect((pagination.getByRole("button", { name: "Last page" }) as HTMLButtonElement).disabled).toBe(true);
  });
});


describe("Performer workspace interactions", () => {
  beforeEach(async () => i18n.changeLanguage("en"));
  afterEach(() => document.body.replaceChildren());

  it("activates a nested subscription button with Space without opening the performer", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    const onOpenPerformer = vi.fn();
    render(<PerformersPage {...props({ onToggle, onOpenPerformer })} />);
    screen.getByRole("button", { name: "Subscribe to Alpha" }).focus();
    await user.keyboard(" ");
    expect(onToggle).toHaveBeenCalledTimes(1);
    expect(onOpenPerformer).not.toHaveBeenCalled();
    expect(screen.getByRole("link", { name: "Alpha" }).getAttribute("href")).toBe("/performers/p1");
  });

  it("exposes a checked checkbox in selection mode and keeps individual actions out of the selection flow", () => {
    const onTogglePerformerSelection = vi.fn();
    render(<PerformersPage {...props({ performerSelectionMode: true, selectedPerformerIds: ["p1"], onTogglePerformerSelection })} />);
    const checkbox = screen.getByRole("checkbox", { name: "Deselect Alpha" }) as HTMLInputElement;
    expect(checkbox.checked).toBe(true);
    fireEvent.click(checkbox);
    expect(onTogglePerformerSelection).toHaveBeenCalledExactlyOnceWith("p1");
    expect(screen.queryByRole("button", { name: "Subscribe to Alpha" })).toBeNull();
  });

  it("provides named sorting and filter controls", () => {
    const onListFilterChange = vi.fn();
    const onListSortChange = vi.fn();
    render(<PerformersPage {...props({ onListFilterChange, onListSortChange })} />);
    fireEvent.click(screen.getByRole("button", { name: "Awaiting review" }));
    expect(onListFilterChange).toHaveBeenCalledWith("review");
    fireEvent.change(screen.getByRole("combobox", { name: "Sort" }), { target: { value: "scene_count" } });
    expect(onListSortChange).toHaveBeenCalledWith("scene_count");
  });

  it("distinguishes loading and failures from empty results and supports retry", () => {
    const onReload = vi.fn();
    const view = render(<PerformersPage {...props({ fetchingStashPerformers: true, stashPerformers: [] })} />);
    expect(screen.getByRole("status", { name: "Loading performer information…" })).toBeTruthy();
    expect(screen.queryByText("No matching performers")).toBeNull();
    view.rerender(<PerformersPage {...props({ stashPerformers: [], stashPerformersError: new Error("offline"), onReload })} />);
    expect(screen.getByRole("alert")).toBeTruthy();
    expect(screen.queryByText("No matching performers")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Reload" }));
    expect(onReload).toHaveBeenCalledTimes(1);
  });

  it("opens full check errors in a separate dialog without opening the performer", async () => {
    const performer = { ...props().stashPerformers[0], subscribed: true };
    const onOpenPerformer = vi.fn();
    render(<PerformersPage {...props({ onOpenPerformer, stashPerformers: [performer], subscribedPerformers: [{ performer, lastCheckedAt: null, lastError: "StashBox unavailable", recentReleases: [], pendingReleaseCount: 0, reviewReleaseCount: 0, processedReleaseCount: 0 }] })} />);
    expect(screen.queryByRole("dialog")).toBeNull();
    fireEvent.click(within(screen.getByText("Alpha").closest("article")!).getByRole("button", { name: "Check failed" }));
    expect(within(screen.getByRole("dialog")).getByText("StashBox unavailable")).toBeTruthy();
    expect(onOpenPerformer).not.toHaveBeenCalled();
    await userEvent.setup().keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("uses sourcing and blocked task labels instead of a generic download label", () => {
    const scene: StashPerformerSceneEntry = {
      key: "one", primarySource: "STASHBOX" as StashPerformerSceneEntry["primarySource"], sourceSceneId: "one", title: "Example", code: "ABC-123", date: null, studioName: null, performerCount: 1, tagCount: 0, performers: [], tags: [], imageUrl: null, url: null, inLibrary: false, matchedStashSceneId: null, hasStashSource: false, hasStashBoxSource: true, stashBoxSceneId: "one", stashBoxEndpoint: null, sourceLabels: [], stashIds: [],
      mojiTask: { id: "task-one", stage: TaskStage.Sourcing, stageStatus: TaskStageStatus.Pending, stageLabel: "", stageStatusLabel: "", progress: 0 }
    };
    const view = render(<PerformerSceneCard scene={scene} page={props()} />);
    expect(screen.getByRole("button", { name: i18n.t("taskQueue.pending") })).toBeTruthy();
    view.rerender(<PerformerSceneCard scene={{ ...scene, mojiTask: { ...scene.mojiTask!, stage: TaskStage.Downloading, stageStatus: TaskStageStatus.Blocked, progress: .5 } }} page={props()} />);
    expect(screen.getByRole("button", { name: i18n.t("taskModel.states.blocked") })).toBeTruthy();
    expect(screen.queryByText("Downloading 50%")).toBeNull();
  });
});
