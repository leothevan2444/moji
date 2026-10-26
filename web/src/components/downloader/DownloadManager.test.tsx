// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
const mocks = vi.hoisted(() => ({ command: vi.fn(), requery: vi.fn(), snapshot: { current: null as unknown } }));
vi.mock("../../graphql/client", () => ({ settingsClient: {} }));
vi.mock("urql", () => ({
  Provider: ({ children }: { children: unknown }) => children,
  useQuery: () => [{ data: { downloader: mocks.snapshot.current }, fetching: false }, mocks.requery],
  useMutation: () => [{ fetching: false }, mocks.command]
}));
import { DownloadManager } from "./DownloadManager";
const torrent = { hash: "a".repeat(40), name: "ABCD-123", state: "stoppedUP", present: true, taskIds: ["task"], taskCodes: ["ABCD-123"], allCompleted: true, canDeleteFiles: true, progress: 1, downloadSpeed: 0, uploadSpeed: 0, size: 1024, completedBytes: 1024, downloaded: 1024, uploaded: 2048, ratio: 2, seedingSeconds: 7200, eta: 0, seeds: 1, peers: 2, availability: 1, savePath: "/downloads", category: "moji", tags: "", lastSeenAt: "2026-10-26T00:00:00Z", removedAt: "", removalReason: "", cleanupError: false, needsAttention: false };
function setup(taskId?: string, onCountChange?: (count:number)=>void) { return render(<MemoryRouter><DownloadManager taskId={taskId} onCountChange={onCountChange} /></MemoryRouter>); }
beforeEach(async () => {
  vi.clearAllMocks(); await i18n.changeLanguage("en"); mocks.command.mockResolvedValue({ data: { controlTorrent: true } });
  mocks.snapshot.current = { checkedAt: "2026-10-26T00:00:00Z", connected: true, stale: false, freeSpace: 4096, storageAvailable: true, downloadSpeed: 0, uploadSpeed: 0, downloadLimit: 0, uploadLimit: 0, completedBytes: 1024, remainingBytes: 0, connectionStatus: "connected", dhtNodes: 10, total: 1, attentionCount: 0, torrents: [{ ...torrent }], policy: { enabled: false, ratio: 0, seedHours: 0, matchAll: false, deleteFiles: false } };
});
afterEach(() => {cleanup();vi.restoreAllMocks();});
describe("download management", () => {
  it("starts a stopped torrent and requires explicit confirmation before file removal", async () => {
    setup();
    fireEvent.click(await screen.findByRole("button", { name: "Start" }));
    await waitFor(() => expect(mocks.command).toHaveBeenCalledWith({ hash: torrent.hash, action: "RESUME" }));
    await waitFor(() => expect((screen.getByRole("button", { name: "Start" }) as HTMLButtonElement).disabled).toBe(false));
    mocks.command.mockClear();
    fireEvent.click(screen.getByRole("button",{name:"More actions"}));
    fireEvent.click(screen.getByRole("menuitem",{name:"Remove torrent and download files"}));
    expect(mocks.command).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
    await waitFor(() => expect(mocks.command).toHaveBeenCalledWith({ hash: torrent.hash, action: "REMOVE_FILES" }));
  });
  it("disables commands for stale observations", async () => {
    (mocks.snapshot.current as { stale: boolean }).stale = true; setup();
    expect((await screen.findByRole("button", { name: "Start" }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByText(/Data may be stale/u)).toBeTruthy();
  });
  it("keeps settings out of the torrent list and uses a three second poll with an icon refresh", async () => {
    const timer = vi.spyOn(window, "setInterval");
    setup();
    expect(screen.queryByRole("button", {name: "Save cleanup policy"})).toBeNull();
    expect(screen.queryByText("Global speed limits")).toBeNull();
    expect(timer).toHaveBeenCalledWith(expect.any(Function), 3000);
    const refresh = screen.getByRole("button", {name: "Refresh qB status"});
    expect(refresh.title).toBe("Refresh qB status");
    expect(refresh.textContent).toBe("");
  });
  it("shows completed missing source data as neutral information", async () => {
    const snap = mocks.snapshot.current as { torrents: Array<typeof torrent> };
    snap.torrents = [{...torrent, present:false, state:"missingFiles", needsAttention:false}];
    const view=setup("task");
    expect(await screen.findByText(/library copy is unaffected/u)).toBeTruthy();
    expect(view.container.querySelector(".download-torrent--attention")).toBeNull();
    expect(screen.queryByText(/absent from qB/u)).toBeNull();
  });
  it("protects unfinished ingest and displays missing torrents without transfer controls", async () => {
    const snap = mocks.snapshot.current as { torrents: Array<typeof torrent> };
    snap.torrents = [{ ...torrent, allCompleted: false, canDeleteFiles: false, present: false, state: "missing", needsAttention: true }];
    setup("task");
    expect(await screen.findByText(/absent from qB/u)).toBeTruthy();
    expect((screen.getByRole("button", { name: "Remove torrent" }) as HTMLButtonElement).disabled).toBe(true);
    expect((screen.getByRole("button", { name: "Pause" }) as HTMLButtonElement).disabled).toBe(true);
  });
  it("uses separate torrent and linked-task columns and excludes absent sources",()=>{
    const snap=mocks.snapshot.current as {torrents:Array<typeof torrent>};
    snap.torrents=[{...torrent},{...torrent,hash:"b".repeat(40),name:"Independent",taskIds:[],taskCodes:[]},{...torrent,hash:"c".repeat(40),name:"Absent",present:false}];
    setup();
    expect(screen.getByRole("columnheader",{name:/^Torrent$/})).toBeTruthy();
    expect(screen.getByRole("columnheader",{name:/^Linked tasks$/})).toBeTruthy();
    const row=screen.getByRole("row",{name:/ABCD-123/});
    const cells=within(row).getAllByRole("cell");
    expect(within(cells[0]).queryByRole("link")).toBeNull();
    expect(within(cells[1]).getByRole("link").getAttribute("href")).toBe("/tasks/task");
    expect(screen.getByRole("row",{name:/Independent/})).toBeTruthy();
    expect(screen.queryByText("Absent")).toBeNull();
  });
  it("closes the action menu on Escape and restores its trigger",()=>{
    setup();const trigger=screen.getByRole("button",{name:"More actions"});
    fireEvent.click(trigger);
    const menu=screen.getByRole("menu");
    fireEvent.keyDown(menu,{key:"Escape"});
    expect(screen.queryByRole("menu")).toBeNull();
    expect(document.activeElement).toBe(trigger);
  });

  it("uses a placeholder, reports title counts and only shows pagination for multiple pages",()=>{
    const count=vi.fn();const view=setup(undefined,count);
    expect(screen.getByPlaceholderText("Search name, code, hash, category or tags")).toBeTruthy();
    expect(screen.queryByText("Search name, code, hash, category or tags")).toBeNull();
    expect(screen.queryByText(/Last sync:/)).toBeNull();
    expect(screen.queryByText("1 torrents")).toBeNull();
    expect(screen.queryByRole("button",{name:"Next"})).toBeNull();
    expect(count).toHaveBeenLastCalledWith(1);
    (mocks.snapshot.current as {total:number}).total=31;
    view.rerender(<MemoryRouter><DownloadManager onCountChange={count}/></MemoryRouter>);
    expect(screen.getByRole("button",{name:"Next"})).toBeTruthy();
    expect(count).toHaveBeenLastCalledWith(31);
  });

});
