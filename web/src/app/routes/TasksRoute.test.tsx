// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createMemoryRouter, Outlet, RouterProvider } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  restart: vi.fn(), toast: vi.fn(), refresh: vi.fn(),
  overview: {} as any, detail: {} as any
}));
vi.mock("urql", () => ({
  useQuery: ({ variables }: { variables?: { id: string } }) => [{ data: variables ? mocks.detail : mocks.overview, fetching: false }, mocks.refresh]
}));
vi.mock("../../hooks/useTaskMutations", () => ({
  useTaskMutations: () => ({ restartTaskSourcing: mocks.restart, restartingTaskSourcing: false })
}));
vi.mock("../../pages/TasksPage", () => ({ TasksPage: () => null }));
vi.mock("../../components/drawers/TaskDrawer", () => ({
  TaskDrawer: ({ task, onRestartTask }: any) => <button onClick={() => onRestartTask(task.id)}>Restart test task</button>
}));
vi.mock("../../components/drawers/SourcingResolutionDrawer", () => ({
  SourcingResolutionDrawer: () => <p>Manual torrent selection</p>
}));

import i18n from "../../i18n/i18n";
import { Component } from "./TasksRoute";

beforeEach(async () => {
  vi.clearAllMocks();
  await i18n.changeLanguage("en");
  const task = { id: "task-184", code: "SNOS-184", stage: "SCANNING", stageStatus: "BLOCKED" };
  mocks.overview = { tasks: [task], settingsStatus: { automation: {} }, settings: { system: { taskDeletePolicy: "REMOVE_TORRENT_AND_FILES" } } };
  mocks.detail = { task };
});
afterEach(cleanup);

function renderRoute() {
  const router = createMemoryRouter([{
    element: <Outlet context={{ pushToast: mocks.toast, copyText: vi.fn() }} />,
    children: [{ path: "/tasks/:taskId", element: <Component /> }, { path: "/tasks/:taskId/resolve", element: <Component /> }]
  }], { initialEntries: ["/tasks/task-184"] });
  render(<RouterProvider router={router} />);
  return router;
}

it("requires confirmation and opens manual selection only after restart succeeds", async () => {
  mocks.restart.mockResolvedValue({ data: { restartTaskSourcing: { id: "task-184", code: "SNOS-184" } } });
  const router = renderRoute();
  fireEvent.click(await screen.findByRole("button", { name: "Restart test task" }));
  expect(mocks.restart).not.toHaveBeenCalled();
  expect(screen.getByText(/corresponding qBittorrent download task and downloaded files/)).toBeTruthy();
  expect(screen.getByText(/without a backup/)).toBeTruthy();
  expect(screen.getByText(/cannot be recovered/)).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Delete files and reselect torrent" }));
  await waitFor(() => expect(mocks.restart).toHaveBeenCalledExactlyOnceWith({ id: "task-184" }));
  expect(mocks.toast).toHaveBeenCalledWith("tone-success", "SNOS-184 has been reset. Choose a torrent to start again.");
  await waitFor(() => expect(router.state.location.pathname).toBe("/tasks/task-184/resolve"));
  expect(await screen.findByText("Manual torrent selection")).toBeTruthy();
});

it("keeps the task and restart confirmation open if cleanup fails", async () => {
  mocks.restart.mockResolvedValue({ error: { graphQLErrors: [{ extensions: { code: "TASK_RESTART_CLEANUP_PENDING" } }] } });
  const router = renderRoute();
  fireEvent.click(await screen.findByRole("button", { name: "Restart test task" }));
  fireEvent.click(screen.getByRole("button", { name: "Delete files and reselect torrent" }));
  await waitFor(() => expect(mocks.toast).toHaveBeenCalledWith("tone-danger", expect.stringContaining("Torrent reselection cleanup is incomplete")));
  expect(router.state.location.pathname).toBe("/tasks/task-184");
  expect(screen.getByRole("button", { name: "Delete files and reselect torrent" })).toBeTruthy();
});
