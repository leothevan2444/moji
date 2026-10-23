// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createMemoryRouter, Outlet, RouterProvider } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";

const mocks = vi.hoisted(() => ({ process: vi.fn(), refresh: vi.fn(), pushToast: vi.fn() }));
vi.mock("../../hooks/usePerformerSubscriptionEvents", () => ({ usePerformerSubscriptionEvents: () => ({ event: null }) }));
vi.mock("../../hooks/useTaskEvents", () => ({ useTaskEvents: () => ({}) }));
vi.mock("urql", () => ({
  useMutation: () => [{ fetching: false }, mocks.process],
  useQuery: ({ variables }: { variables: { input: { page: number; status: string } } }) => {
    const { page, status } = variables.input;
    const ids = page === 1 ? ["one", "two"] : ["three", "four"];
    return [{ fetching: false, error: null, data: { releaseReviews: {
      totalCount: 4, pendingCount: 4, failedCount: 0, page, pageSize: 2, totalPages: 2,
      performers: [{ id: "p1", name: "Actor" }],
      items: ids.map(id => ({ key: id, code: id, title: `Movie ${id}`, status, vr: false, source: "stash-box:example", reasonCode: "group_behavior_review", seenAt: "2026-10-23T00:00:00Z", performers: [{ id: "p1", name: "Actor" }], performerNames: ["Actor"] }))
    } } }, mocks.refresh];
  }
}));
import { Component } from "./ReleaseReviewsRoute";

function setup() {
  const router = createMemoryRouter([{ element: <Outlet context={{ pushToast: mocks.pushToast }} />, children: [{ path: "/performers/review", element: <Component /> }] }], { initialEntries: ["/performers/review"] });
  return render(<RouterProvider router={router} />);
}
beforeEach(async () => { vi.clearAllMocks(); await i18n.changeLanguage("en"); });
afterEach(cleanup);

describe("release review actions", () => {
  it("submits the full cross-page selection and retains failed items", async () => {
    mocks.process.mockResolvedValue({ data: { processReleaseReviews: { succeededCount: 1, skippedCount: 0, failedCount: 1, results: [
      { key: "one", outcome: "SUCCEEDED", reasonCode: "UPDATED", release: { status: "DOWNLOADED", taskId: "task-1" } },
      { key: "three", outcome: "FAILED", reasonCode: "TASK_CREATION_FAILED", release: { status: "FAILED", taskId: null } }
    ] } } });
    setup();
    fireEvent.click(await screen.findByRole("checkbox", { name: "Select one" }));
    fireEvent.click(screen.getByRole("button", { name: "Next page" }));
    fireEvent.click(await screen.findByRole("checkbox", { name: "Select three" }));
    fireEvent.click(screen.getByRole("button", { name: "Download selected (2)" }));
    await waitFor(() => expect(mocks.process).toHaveBeenCalledWith({ input: { keys: ["one", "three"], action: "DOWNLOAD" } }));
    await waitFor(() => expect(screen.getByText("1 selected across pages")).toBeTruthy());
    expect((screen.getByRole("checkbox", { name: "Select three" }) as HTMLInputElement).checked).toBe(true);
    expect(screen.getByRole("link", { name: "View task" }).getAttribute("href")).toBe("/tasks/task-1");
  });

  it("rejects without navigating to tasks and restores historical decisions explicitly", async () => {
    mocks.process.mockResolvedValue({ data: { processReleaseReviews: { succeededCount: 1, skippedCount: 0, failedCount: 0, results: [{ key: "one", outcome: "SUCCEEDED", reasonCode: "UPDATED", release: { status: "REJECTED", taskId: null } }] } } });
    setup();
    fireEvent.click((await screen.findAllByRole("button", { name: "Do not download" }))[0]);
    await waitFor(() => expect(mocks.process).toHaveBeenLastCalledWith({ input: { keys: ["one"], action: "REJECT" } }));
    expect(screen.queryByRole("link", { name: "View task" })).toBeNull();
    fireEvent.change(screen.getByLabelText("Review status"), { target: { value: "REJECTED" } });
    fireEvent.click((await screen.findAllByRole("button", { name: "Reconsider" }))[0]);
    await waitFor(() => expect(mocks.process).toHaveBeenLastCalledWith({ input: { keys: ["one"], action: "RECONSIDER" } }));
  });
});
