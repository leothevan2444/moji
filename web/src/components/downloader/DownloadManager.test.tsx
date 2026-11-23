// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { DocumentNode } from "graphql";
import i18n from "../../i18n/i18n";
import official from "../../utils/__fixtures__/qbittorrent-statuses.json";
const mocks = vi.hoisted(() => ({ command: vi.fn(), requery: vi.fn(), pieces:{hash:"a".repeat(40),total:3,observedAt:"2026-10-26T00:00:00Z",buckets:[{start:0,count:3,missing:1,downloading:1,downloaded:1}]}, queryError: undefined as Error|undefined, dataAvailable:true, snapshot: { current: null as unknown } }));
vi.mock("../../graphql/client", () => ({ settingsClient: {} }));
vi.mock("urql", () => ({
  Provider: ({ children }: { children: unknown }) => children,
  useQuery: (options:{query:DocumentNode;pause?:boolean}) => options.query.definitions.some(def=>def.kind==="OperationDefinition"&&def.name?.value==="TorrentPiecesDocument")
    ? [{data:options.pause?undefined:{torrentPieces:mocks.pieces},fetching:false},mocks.requery]
    : [{ data: mocks.dataAvailable ? { downloader: mocks.snapshot.current } : undefined, error:mocks.queryError, fetching: false }, mocks.requery],
  useMutation: () => [{ fetching: false }, mocks.command]
}));
import { TaskSource,TaskStage,TaskStageStatus } from "../../graphql/generated/graphql";
import { TaskFiles } from "../tasks/TaskFiles";
import type { DashboardTask } from "../../utils";
import type { Torrent } from "./TorrentDetails";
import { DownloadManager } from "./DownloadManager";
const torrent:Torrent = { hash: "a".repeat(40), name: "ABCD-123", state: "stoppedUP", present: true, taskIds: ["task"], taskCodes: ["ABCD-123"], allCompleted: true, canDeleteFiles: true, progress: 1, downloadSpeed: 0, uploadSpeed: 0, size: 1024, completedBytes: 1024, downloaded: 1024, uploaded: 2048, ratio: 2, seedingSeconds: 7200, eta: 0, seeds: 1, peers: 2, seedsTotal:5,peersTotal:9,activeSeconds:10800,lastSeenCompleteAt:"2026-10-25T23:00:00Z",availability: 1, savePath: "/downloads", category: "moji", tags: "", lastSeenAt: "2026-10-26T00:00:00Z", removedAt: "", removalReason: "", cleanupError: false, needsAttention: false };
function setup(taskId?: string, onCountChange?: (count:number)=>void) { return render(<MemoryRouter><DownloadManager taskId={taskId} onCountChange={onCountChange} /></MemoryRouter>); }
beforeEach(async () => {
  vi.clearAllMocks();mocks.queryError=undefined;mocks.dataAvailable=true; await i18n.changeLanguage("en"); mocks.command.mockResolvedValue({ data: { controlTorrent: true } });
  mocks.snapshot.current = { checkedAt: "2026-10-26T00:00:00Z", connected: true, stale: false, freeSpace: 4096, storageAvailable: true, downloadSpeed: 0, uploadSpeed: 0, downloadLimit: 0, uploadLimit: 0, completedBytes: 1024, remainingBytes: 0, connectionStatus: "connected", dhtNodes: 10, total: 1, attentionCount: 0, torrents: [{ ...torrent }], policy: { enabled: false, ratio: 0, seedHours: 0, matchAll: false, deleteFiles: false } };
});
afterEach(() => {cleanup();vi.restoreAllMocks();});
describe("download management", () => {
  it.each(["en", "zh-CN"] as const)("uses official %s status copy in both the global table and task card", async locale => {
    await i18n.changeLanguage(locale);
    const snap = mocks.snapshot.current as { torrents: Array<typeof torrent> };
    snap.torrents = [{ ...torrent, state: "forcedMetaDL" }];
    const expected = official.releases[0].states.forcedMetaDL[locale === "en" ? 0 : 1];
    const table = setup();
    expect(screen.getByText(expected)).toBeTruthy();
    expect(screen.queryByText("forcedMetaDL")).toBeNull();
    table.unmount();
    setup("task");
    expect(screen.getByText(expected)).toBeTruthy();
    expect(screen.queryByText("forcedMetaDL")).toBeNull();
  });
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
  it("embeds qB controls in torrent information with one polling loop and retains removal confirmation", async () => {
    const timer = vi.spyOn(window, "setInterval");
    const { container } = render(<MemoryRouter><TaskFiles task={{ id: "task", torrentHash: torrent.hash, torrentName: torrent.name, files: [], stage: "COMPLETED", stageStatus: "DONE" } as unknown as DashboardTask} torrentControls={<DownloadManager taskId="task" embedded />} /></MemoryRouter>);
    const section = container.querySelector(".task-files")!;
    expect(within(section as HTMLElement).getByRole("heading", { name: "Torrent information" })).toBeTruthy();
    expect(section.querySelectorAll(".download-manager")).toHaveLength(1);
    expect(section.querySelector(".download-torrent.drawer-card")).toBeNull();
    expect(timer).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("link")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Refresh qB status" }));
    await waitFor(() => expect(mocks.command).toHaveBeenCalledWith({}));
    await waitFor(() => expect((screen.getByRole("button", { name: "Refresh qB status" }) as HTMLButtonElement).disabled).toBe(false));
    expect((screen.getByRole("button",{name:"Remove torrent and download files"}) as HTMLButtonElement).disabled).toBe(false);
    mocks.command.mockClear();
    fireEvent.click(screen.getByRole("button", { name: "Remove torrent and download files" }));
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
  it.each([
    ["downloading",["downloader.download","downloader.upload","downloader.eta","torrentCard.completedSize","torrentCard.seeders","torrentCard.peers"]],
    ["forcedDL",["downloader.download","downloader.upload","downloader.eta","torrentCard.completedSize","torrentCard.seeders","torrentCard.peers"]],
    ["uploading",["downloader.upload","downloader.ratio","downloader.seedTime","torrentCard.seeders","torrentCard.peers"]],
    ["stalledUP",["downloader.upload","downloader.ratio","downloader.seedTime","torrentCard.seeders","torrentCard.peers"]],
    ["stoppedDL",["torrentCard.completedSize","downloader.progress","torrentCard.seeders","torrentCard.peers"]],
    ["checkingDL",["torrentCard.completedSize","downloader.progress","torrentCard.seeders","torrentCard.peers"]],
  ] as const)("shows relevant metrics for %s without another progress bar",(state,labels)=>{
    (mocks.snapshot.current as {torrents:Array<typeof torrent>}).torrents=[{...torrent,state}];
    const {container}=setup("task");
    const metrics=container.querySelector(".download-metrics")!;
    expect(Array.from(metrics.querySelectorAll(".muted")).map(node=>node.textContent)).toEqual(labels.map(key=>i18n.t(key)));
    expect(screen.queryByRole("progressbar")).toBeNull();
  });

  it("keeps recent torrent observations on query failure and clears a confirmed empty result",()=>{
    const view=setup("task");
    mocks.queryError=new Error("offline");mocks.dataAvailable=false;
    view.rerender(<MemoryRouter><DownloadManager taskId="task" /></MemoryRouter>);
    expect(screen.getByText(torrent.name)).toBeTruthy();
    expect(screen.getByText(/Data may be stale/)).toBeTruthy();
    expect((screen.getByRole("button",{name:"Start"}) as HTMLButtonElement).disabled).toBe(true);
    mocks.queryError=undefined;mocks.dataAvailable=true;
    mocks.snapshot.current={...(mocks.snapshot.current as object),torrents:[],total:0};
    view.rerender(<MemoryRouter><DownloadManager taskId="task" /></MemoryRouter>);
    expect(screen.queryByText(torrent.name)).toBeNull();
  });

  it("does not reuse retained observations when switching tasks",()=>{
    const view=setup("task");
    mocks.queryError=new Error("offline");mocks.dataAvailable=false;
    view.rerender(<MemoryRouter><DownloadManager taskId="different-task" /></MemoryRouter>);
    expect(screen.queryByText(torrent.name)).toBeNull();
    expect(screen.queryByRole("button",{name:"Start"})).toBeNull();
  });

  it("shows the current qB name once and keeps technical fields collapsed and copyable",()=>{
    const copy=vi.fn();
    const saved:DashboardTask={id:"task",stashSceneIds:[],source:TaskSource.Manual,code:"ABCD-123",stageLabel:"",stageStatusLabel:"",progress:1,createdAt:"2026-10-26T00:00:00Z",updatedAt:"2026-10-26T00:00:00Z",torrentName:"Earlier qB name",torrentHash:torrent.hash,torrentUrl:"magnet:?xt=urn:btih:"+torrent.hash,contentPath:"/downloads/actual.mp4",files:[],stage:TaskStage.Completed,stageStatus:TaskStageStatus.Done};
    const {container}=render(<MemoryRouter><TaskFiles task={saved} onCopy={copy} torrentControls={<DownloadManager taskId="task" embedded task={saved} onCopy={copy}/>} /></MemoryRouter>);
    expect(screen.getAllByText(torrent.name)).toHaveLength(1);
    expect(screen.queryByText(saved.torrentName!)).toBeNull();
    const details=container.querySelector<HTMLDetailsElement>(".torrent-information")!;
    expect(details.open).toBe(false);
    fireEvent.click(details.querySelector("summary")!);
    expect(details.open).toBe(true);
    fireEvent.click(screen.getByRole("button",{name:"Copy Content location"}));
    expect(copy).toHaveBeenCalledWith(saved.contentPath,"Content location copied");
    expect(container.querySelectorAll(".download-metrics > div")).toHaveLength(4);
  });

  it("keeps task actions expanded and restores focus after confirmation",()=>{
    setup("task");
    expect(screen.queryByRole("button",{name:"More actions"})).toBeNull();
    const recheck=screen.getByRole("button",{name:"Recheck"});
    fireEvent.click(recheck);
    expect(document.activeElement).toBe(screen.getByRole("button",{name:"Confirm"}));
    fireEvent.click(screen.getByRole("button",{name:"Cancel"}));
    expect(document.activeElement).toBe(recheck);
    expect(mocks.command).not.toHaveBeenCalled();
  });

  it("rechecks deletion eligibility if observations change while confirming",()=>{
    const view=setup("task");
    fireEvent.click(screen.getByRole("button",{name:"Remove torrent and download files"}));
    expect(mocks.command).not.toHaveBeenCalled();
    (mocks.snapshot.current as {torrents:Array<typeof torrent>}).torrents=[{...torrent,canDeleteFiles:false}];
    view.rerender(<MemoryRouter><DownloadManager taskId="task" /></MemoryRouter>);
    expect((screen.getByRole("button",{name:"Confirm"}) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.click(screen.getByRole("button",{name:"Confirm"}));
    expect(mocks.command).not.toHaveBeenCalled();
  });

  it("keeps the name label hidden and removes transfer speeds from technical details",()=>{
    const {container}=setup("task");
    expect(container.querySelector<HTMLDListElement>(".torrent-identity dt")?.hasAttribute("hidden")).toBe(true);
    const details=container.querySelector(".torrent-information")! as HTMLElement;
    expect(within(details).queryByText(i18n.t("downloader.download"))).toBeNull();
    expect(within(details).queryByText(i18n.t("downloader.upload"))).toBeNull();
    expect(within(details).getByText("Last observed").closest("div")?.title).toContain("Moji");
  });

  it("shows negative availability and connection counts as unknown rather than real values",()=>{
    (mocks.snapshot.current as {torrents:Array<typeof torrent>}).torrents=[{...torrent,availability:-1,seeds:-1,peers:-1}];
    const {container}=setup("task");
    const label=container.querySelector(".torrent-details")! as HTMLElement;
    const field=within(label).getByText("Available copies").closest("div")!;
    expect(field.querySelector("dd")?.textContent).toBe("Unknown");
    expect(field.title).toContain("Distributed copies");
    const metrics=container.querySelector(".download-metrics")! as HTMLElement;
    expect(within(metrics).getAllByText(/^Unknown \(/)).toHaveLength(2);
  });

  it("explains ingest protection without enabling removal",()=>{
    (mocks.snapshot.current as {torrents:Array<typeof torrent>}).torrents=[{...torrent,allCompleted:false,canDeleteFiles:false}];
    setup("task");
    const menu=screen.getByRole("button",{name:"Remove torrent"}).closest(".torrent-actions")! as HTMLElement;
    expect(within(menu).queryByText(/Linked tasks have not all finished ingest/)).toBeNull();
    for(const name of ["Remove torrent","Remove torrent and download files"]) {
      const button=within(menu).getByRole("button",{name}) as HTMLButtonElement;
      expect(button.disabled).toBe(true);expect(button.title).toContain("Linked tasks");
    }
    expect(mocks.command).not.toHaveBeenCalled();
  });

  it("distinguishes unsafe file deletion from torrent-only removal",()=>{
    (mocks.snapshot.current as {torrents:Array<typeof torrent>}).torrents=[{...torrent,canDeleteFiles:false}];
    setup("task");
    expect((screen.getByRole("button",{name:"Remove torrent"}) as HTMLButtonElement).disabled).toBe(false);
    expect(screen.queryByText(/independent library copies or shared-file checks/)).toBeNull();
    expect((screen.getByRole("button",{name:"Remove torrent and download files"}) as HTMLButtonElement).title).toContain("independent library copies or shared-file checks");
    expect((screen.getByRole("button",{name:"Remove torrent and download files"}) as HTMLButtonElement).disabled).toBe(true);
  });

  it("shows connected counts with tracker totals and both upstream time fields",()=>{
    const {container}=setup("task");
    expect(screen.getByText("1 (5)")).toBeTruthy();expect(screen.getByText("2 (9)")).toBeTruthy();
    const details=container.querySelector(".torrent-information")! as HTMLElement;
    expect(within(details).getByText("Active time").closest("div")?.querySelector("dd")?.textContent).toBe("3:00:00");
    expect(within(details).getByText("Last seen complete").closest("div")?.querySelector("dd")?.textContent).not.toBe("Unknown");
    expect(within(details).queryByText("Selected file size")).toBeNull();
  });
  it("keeps unknown tracker totals and upstream timestamps unknown while preserving real zero activity",()=>{
    (mocks.snapshot.current as {torrents:Torrent[]}).torrents=[{...torrent,seedsTotal:null,peersTotal:null,activeSeconds:0,lastSeenCompleteAt:null}];
    const {container}=setup("task");expect(screen.getByText("1 (Unknown)")).toBeTruthy();expect(screen.getByText("2 (Unknown)")).toBeTruthy();
    const details=container.querySelector(".torrent-information")! as HTMLElement;
    expect(within(details).getByText("Active time").closest("div")?.querySelector("dd")?.textContent).toBe("0:00");
    expect(within(details).getByText("Last seen complete").closest("div")?.querySelector("dd")?.textContent).toBe("Unknown");
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
