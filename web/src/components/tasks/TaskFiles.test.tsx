// @vitest-environment jsdom
import { cleanup, render, screen, within } from "@testing-library/react";
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
 it("uses the save directory when the content path is unavailable", async () => {
  const onCopy = vi.fn();
  render(<TaskFiles task={{ ...task, files: [], savePath: "/downloads/library", tags: " " }} onCopy={onCopy} />);
  expect(screen.getByText("Download location")).toBeTruthy();
  expect(screen.queryByText("Tags")).toBeNull();
  await userEvent.setup().click(screen.getByRole("button", { name: "Copy Download location" }));
  expect(onCopy).toHaveBeenCalledWith("/downloads/library", "Download location copied");
 });
 it("lets the user choose and order uncertain videos without selecting extras", async () => {
  const user = userEvent.setup(); render(<TaskFiles task={task} />);
  const checkboxes = screen.getAllByRole("checkbox");
  expect(checkboxes).toHaveLength(2);
  await user.click(checkboxes[0]); await user.click(checkboxes[1]);
  await user.click(screen.getByRole("button", { name: "Move unknown-b.mkv up" }));
  await user.click(screen.getByRole("button", { name: "Confirm and continue download" }));
  expect(resolve).toHaveBeenCalledWith({ id: task.id, videoIndices: [12, 7] });
  expect(await screen.findByRole("status")).toBeTruthy();
 });
 it("always shows the file groups and gives download files priority over skipped extras", () => {
  const { rerender } = render(<TaskFiles task={{ ...task, stageStatus: "RUNNING" as DashboardTask["stageStatus"] }} />);
  expect(screen.queryAllByRole("checkbox")).toHaveLength(0);
  const list = screen.getByRole("article", { name: "Torrent information" });
  expect(list.querySelector("details, summary")).toBeNull();
  const downloads = screen.getByRole("region", { name: "Download files" });
  const skipped = screen.getByRole("region", { name: "Skipped" });
  expect(within(downloads).getByText("movie.en.srt")).toBeTruthy();
  expect(within(skipped).getByText("advert.mp4")).toBeTruthy();
  expect(within(skipped).getByText("unknown-a.mkv")).toBeTruthy();
  expect(screen.queryByText("Only selected movie files and retained subtitles are delivered to Stash. Multiple movie parts are merged in the order below.")).toBeNull();
  rerender(<TaskFiles task={{ ...task, id: "task-2" }} />);
  expect(screen.getByRole("article", { name: "Torrent information" }).querySelector("details, summary")).toBeNull();
 });
 it("shows filenames with complete paths on hover and copies the original path", async () => {
  const user = userEvent.setup();
  const onCopy = vi.fn();
  const path = "release/a-long-directory/movie.en.srt";
  render(<TaskFiles task={{ ...task, files: [{ ...task.files[2], path }] }} onCopy={onCopy} />);
  expect(screen.getByText("movie.en.srt").getAttribute("title")).toBe(path);
  await user.click(screen.getByRole("button", { name: `Copy ${path}` }));
  expect(onCopy).toHaveBeenCalledWith(path, `${path} copied`);
 });
 it("updates selection and merge order in the same list while protecting against empty submissions", async () => {
  const user = userEvent.setup();
  render(<TaskFiles task={task} />);
  const confirm = screen.getByRole("button", { name: "Confirm and continue download" });
  expect((confirm as HTMLButtonElement).disabled).toBe(true);
  await user.click(screen.getByRole("checkbox", { name: "unknown-a.mkv" }));
  await user.click(screen.getByRole("checkbox", { name: "unknown-b.mkv" }));
  await user.click(screen.getByRole("button", { name: "Move unknown-b.mkv up" }));
  const candidates = screen.getByRole("region", { name: "Movie files & merge order" });
  expect(within(candidates).getAllByRole("checkbox").map(input => input.getAttribute("aria-label"))).toEqual(["unknown-b.mkv", "unknown-a.mkv"]);
  expect(document.activeElement).toBe(screen.getByRole("checkbox", { name: "unknown-b.mkv" }));
  await user.click(confirm);
  expect(resolve).toHaveBeenCalledWith({ id: task.id, videoIndices: [12, 7] });
 });
 it("keeps selections available after a failed save and allows retry", async () => {
  const user = userEvent.setup();
  resolve.mockRejectedValueOnce(new Error("Connection interrupted"));
  render(<TaskFiles task={task} />);
  await user.click(screen.getByRole("checkbox", { name: "unknown-a.mkv" }));
  const confirm = screen.getByRole("button", { name: "Confirm and continue download" });
  await user.click(confirm);
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect((screen.getByRole("checkbox", { name: "unknown-a.mkv" }) as HTMLInputElement).checked).toBe(true);
  await user.click(confirm);
  expect(await screen.findByRole("status")).toBeTruthy();
 });
 it("shows torrent metadata even before its files have synced", async () => {
  const user = userEvent.setup();
  const onCopy = vi.fn();
  const torrentName = "DEMO-001 Example movie 1080p";
  const torrentHash = "0123456789abcdef0123456789abcdef01234567";
  const torrentUrl = `magnet:?xt=urn:btih:${torrentHash}`;
  render(<TaskFiles task={{ ...task, files: [], torrentName, torrentHash, torrentUrl }} onCopy={onCopy} />);
  const information = screen.getByRole("article", { name: "Torrent information" });
  expect(within(information).getByText(torrentName).getAttribute("title")).toBe(torrentName);
  expect(within(information).getByText(torrentHash).getAttribute("title")).toBe(torrentHash);
  expect(within(information).getByText(torrentUrl).getAttribute("title")).toBe(torrentUrl);
  expect(within(information).queryByRole("list")).toBeNull();
  await user.click(within(information).getByRole("button", { name: "Copy Torrent name" }));
  expect(onCopy).toHaveBeenCalledWith(torrentName, "Torrent name copied");
  await user.click(within(information).getByRole("button", { name: "Copy Magnet link" }));
  expect(onCopy).toHaveBeenCalledWith(torrentUrl, "Magnet link copied");
 });
 it("does not show an empty torrent panel before a resource is chosen", () => {
  render(<TaskFiles task={{ ...task, files: [] }} />);
  expect(screen.queryByRole("article", { name: "Torrent information" })).toBeNull();
 });

});
