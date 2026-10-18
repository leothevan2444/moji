// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { TaskStage } from "../../graphql/generated/graphql";
import type { DashboardTask } from "../../utils";
import { TaskStashScenes } from "./TaskStashScenes";

const mocks = vi.hoisted(() => ({ query: vi.fn(), reload: vi.fn(), result: {} as any }));
vi.mock("urql", () => ({ useQuery: (options: unknown) => { mocks.query(options); return [mocks.result, mocks.reload]; } }));
const task = { id: "task-1", stage: TaskStage.Completed, stashSceneIds: ["1", "2"] } as DashboardTask;
const scene = { id: "1", title: "Example movie", code: "DEMO-001", date: "2026-10-18", studio: "Example studio", performers: ["Actor One", "Actor Two"], imageUrl: "/api/images/cover", url: "https://stash.test/scenes/1" };

beforeEach(async () => { vi.clearAllMocks(); mocks.result = { fetching: false }; await i18n.changeLanguage("en"); });
afterEach(cleanup);

describe("Stash movie information", () => {
  it("shows a quiet placeholder before ingest without fetching Stash", () => {
    render(<TaskStashScenes task={{ ...task, stage: TaskStage.Downloading, stashSceneIds: [] }} />);
    expect(screen.getByText("Movie information appears after ingest")).toBeTruthy();
    expect(mocks.query.mock.calls[0][0].pause).toBe(true);
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("shows multiple movies with metadata and direct Stash links", () => {
    mocks.result = { data: { taskStashScenes: [scene, { ...scene, id: "2", title: "Second movie", imageUrl: null, url: "https://stash.test/scenes/2" }] } };
    const { container } = render(<TaskStashScenes task={task} />);
    expect(screen.getByRole("article", { name: "Stash movie information" })).toBeTruthy();
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
    expect(screen.getByRole("heading", { name: "Example movie" })).toBeTruthy();
    expect(screen.getAllByText("Actor One · Actor Two")).toHaveLength(2);
    const link = screen.getByRole("link", { name: "View in Stash: Example movie" });
    expect(link.getAttribute("href")).toBe(scene.url);
    expect(link.getAttribute("rel")).toContain("noopener");
    const image = container.querySelector("img")!;
    expect(image.getAttribute("src")).toBe(scene.imageUrl);
    fireEvent.error(image);
    expect(container.querySelector("img")).toBeNull();
    expect(screen.queryByText("Files and ingest results")).toBeNull();
  });

  it("offers a retry when Stash is unavailable", async () => {
    mocks.result = { error: new Error("private connection detail") };
    render(<TaskStashScenes task={task} />);
    expect(screen.getByText("Stash movie information is temporarily unavailable")).toBeTruthy();
    expect(screen.queryByText("private connection detail")).toBeNull();
    await userEvent.setup().click(screen.getByRole("button", { name: "Reload" }));
    expect(mocks.reload).toHaveBeenCalledWith({ requestPolicy: "network-only" });
  });

  it("distinguishes loading, removed movies, and historical tasks without links", () => {
    mocks.result = { fetching: true };
    const { rerender } = render(<TaskStashScenes task={task} />);
    expect(screen.getByRole("status").textContent).toBe("Loading movie information…");
    mocks.result = { fetching: false, data: { taskStashScenes: [scene] } };
    rerender(<TaskStashScenes task={task} />);
    expect(screen.getByText("Unavailable linked movies: 1")).toBeTruthy();
    rerender(<TaskStashScenes task={{ ...task, stashSceneIds: [] }} />);
    expect(screen.getByText("No linked Stash movies found")).toBeTruthy();
    expect(screen.queryByRole("link")).toBeNull();
  });
});
