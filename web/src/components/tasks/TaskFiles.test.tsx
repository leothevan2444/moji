// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import type { DashboardTask } from "../../utils";
import { TaskFiles } from "./TaskFiles";

const { resolve } = vi.hoisted(() => ({ resolve: vi.fn() }));
vi.mock("urql", () => ({ useMutation: () => [{ fetching: false }, resolve] }));
const task = {
 id: "task-1", stage: "DOWNLOADING", stageStatus: "BLOCKED",
 files: [
  { index: 7, path: "unknown-a.mkv", kind: "UNKNOWN_VIDEO", selected: false, part: 0, reason: "IDENTITY_UNCERTAIN", size: 100 },
  { index: 12, path: "unknown-b.mkv", kind: "UNKNOWN_VIDEO", selected: false, part: 0, reason: "IDENTITY_UNCERTAIN", size: 100 },
  { index: 18, path: "movie.en.srt", kind: "SUBTITLE", selected: true, part: 0, reason: "SUBTITLE_POLICY", size: 10 },
  { index: 22, path: "advert.mp4", kind: "DISCARD", selected: false, part: 0, reason: "ADVERTISEMENT_OR_SAMPLE", size: 10 }
 ]
} as DashboardTask;
describe("TaskFiles", () => {
 beforeEach(async () => { await i18n.changeLanguage("en"); resolve.mockReset(); resolve.mockResolvedValue({ data: { resolveTaskFiles: { id: task.id } } }); });
 afterEach(cleanup);
 it("lets the user choose and order uncertain videos without selecting extras", async () => {
  const user = userEvent.setup(); render(<TaskFiles task={task} />);
  const checkboxes = screen.getAllByRole("checkbox");
  expect(checkboxes).toHaveLength(2);
  await user.click(checkboxes[0]); await user.click(checkboxes[1]);
  await user.click(screen.getByRole("button", { name: "Move unknown-b.mkv up" }));
  await user.click(screen.getByRole("button", { name: "Confirm files and merge order" }));
  expect(resolve).toHaveBeenCalledWith({ id: task.id, videoIndices: [12, 7] });
  expect(await screen.findByRole("status")).toBeTruthy();
 });
 it("shows the plan without editable controls while downloading", () => {
  render(<TaskFiles task={{ ...task, stageStatus: "RUNNING" as DashboardTask["stageStatus"] }} />);
  expect(screen.queryAllByRole("checkbox")).toHaveLength(0);
  expect(screen.getByText("unknown-a.mkv")).toBeTruthy();
  expect(screen.getByText("advert.mp4")).toBeTruthy();
 });
});
