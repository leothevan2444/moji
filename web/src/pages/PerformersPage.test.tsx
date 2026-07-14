// @vitest-environment jsdom

import { fireEvent, render, screen, within } from "@testing-library/react";
import type { ComponentProps } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LibraryFilter, SceneSourceFilter } from "../graphql/generated/graphql";
import i18n from "../i18n/i18n";
import { PerformersPage } from "./PerformersPage";

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

  it("opens a performer normally and selects it in multi-select mode", () => {
    const onOpenPerformer = vi.fn();
    const onTogglePerformerSelection = vi.fn();
    const view = render(<PerformersPage {...props({ onOpenPerformer, onTogglePerformerSelection })} />);
    fireEvent.click(screen.getByText("Alpha").closest("article")!);
    expect(onOpenPerformer).toHaveBeenCalledWith("p1");

    view.rerender(<PerformersPage {...props({ performerSelectionMode: true, onOpenPerformer, onTogglePerformerSelection })} />);
    fireEvent.click(screen.getByText("Alpha").closest("article")!);
    expect(onTogglePerformerSelection).toHaveBeenCalledWith("p1");
    expect(onOpenPerformer).toHaveBeenCalledTimes(1);
  });

  it("shows subscription eligibility actions for selected performers", () => {
    render(<PerformersPage {...props({ performerSelectionMode: true, selectedPerformerIds: ["p1"] })} />);
    const actions = within(screen.getByRole("region", { name: "Selected performer actions" }));
    expect((actions.getByRole("button", { name: "Subscribe (1)" }) as HTMLButtonElement).disabled).toBe(false);
    expect((actions.getByRole("button", { name: "Check selected (0)" }) as HTMLButtonElement).disabled).toBe(true);
  });
});
