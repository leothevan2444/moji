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

  it("opens a performer normally and selects it in multi-select mode", () => {
    const onOpenPerformer = vi.fn();
    const onTogglePerformerSelection = vi.fn();
    const view = render(<PerformersPage {...props({ onOpenPerformer, onTogglePerformerSelection })} />);
    fireEvent.click(screen.getByText("Alpha").closest("article")!);
    expect(onOpenPerformer).toHaveBeenCalledWith("p1");

    view.rerender(<PerformersPage {...props({ performerSelectionMode: true, onOpenPerformer, onTogglePerformerSelection })} />);
    expect(screen.queryByRole("button", { name: "Select performer Alpha" })).toBeNull();
    fireEvent.click(screen.getByText("Alpha").closest("article")!);
    expect(onTogglePerformerSelection).toHaveBeenCalledWith("p1");
    expect(onOpenPerformer).toHaveBeenCalledTimes(1);
  });

  it("shows subscription eligibility actions for selected performers", () => {
    render(<PerformersPage {...props({ performerSelectionMode: true, selectedPerformerIds: ["p1"] })} />);
    const actions = within(screen.getByRole("region", { name: "Selected performer actions" }));
    expect((actions.getByRole("button", { name: "Subscribe (1)" }) as HTMLButtonElement).disabled).toBe(false);
    expect((actions.getByRole("button", { name: "Check selected (0)" }) as HTMLButtonElement).disabled).toBe(true);
    expect((screen.getByRole("button", { name: "Check Alpha now" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("keeps reload icon-only and moves check-all into more actions", () => {
    render(<PerformersPage {...props()} />);
    const reload = screen.getByRole("button", { name: "Reload from Stash" });
    expect(reload.textContent).toBe("");
    expect(screen.queryByRole("button", { name: "Check all" })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "More actions" }));
    expect(screen.getByRole("dialog", { name: "More actions" })).toBeTruthy();
    expect((screen.getByRole("button", { name: "Check all" }) as HTMLButtonElement).disabled).toBe(true);
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
