// @vitest-environment jsdom
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { TaskStage, TaskStageStatus } from "../../graphql/generated/graphql";
import { taskLifecycle, taskFailureSummary, type DashboardTask } from "../../utils";
import { TaskCard } from "./TaskCard";
import { TaskDetail } from "./TaskDetail";

vi.mock("../downloader/DownloadManager", () => ({ DownloadManager: () => null }));
vi.mock("urql", () => ({ useQuery: () => [{ fetching: false }, vi.fn()], useMutation: () => [{ fetching: false }, vi.fn()] }));
const task = {
  id: "task-1", source: "MANUAL", code: "DEMO-001", torrentName: "Example movie",
  stage: "DOWNLOADING", stageStatus: "RUNNING", progress: 0.46,
  createdAt: "2026-10-17T02:00:00Z", updatedAt: "2026-10-17T03:00:00Z",
  files: [{ index: 1, path: "movie/part-1.mp4", kind: "MAIN", part: 1, selected: true, reason: "TARGET_CODE", size: 100 }]
} as DashboardTask;
const props = () => ({ pendingRetry: false, pendingDelete: false,
  onCopy: vi.fn(), onResolveTask: vi.fn(), onRetryTask: vi.fn(), onDeleteTask: vi.fn() });

beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(cleanup);

describe("task experience", () => {
  it("combines film identity and progress while keeping supplementary information collapsed", async () => {
    const callbacks = props();
    const film = {endpoint:"https://catalog.test/graphql",sceneId:"film-1",code:"DEMO-001",title:"A film with a clear identity",date:"2026-10-10",studio:"Example studio",performers:["Alice","Beth","Cara","Diana"],imageUrl:"",url:"",confirmationDigest:"fixture"};
    const current = {...task,film,category:"private-category",sourcingMode:"USER_SELECTED"};
    const {rerender} = render(<TaskDetail task={current} {...callbacks} />);
    const overview = screen.getByRole("article",{name:"Task overview"});
    const identity = overview.querySelector(".task-detail-summary")! as HTMLElement;
    expect(within(identity).getByRole("heading",{name:film.title})).toBeTruthy();
    for (const value of [film.date,film.studio,film.performers.join(" / ")]) expect(within(identity).getByText(value)).toBeTruthy();
    expect(within(identity).queryByText(film.code,{exact:true})).toBeNull();
    const details = overview.querySelector("details")!;
    expect(details.open).toBe(false);
    expect(within(overview).getByText(film.endpoint).closest("details")).toBe(details);
    expect(within(overview).getByText("private-category").closest("details")).toBe(details);
    expect(within(overview).getByText("Created").closest("details")).toBe(details);
    expect(within(overview).getByRole("list",{name:"Lifecycle"})).toBeTruthy();
    await userEvent.setup().click(details.querySelector("summary")!);
    expect(details.open).toBe(true);
    await userEvent.setup().click(within(details).getByRole("button",{name:"Copy Original metadata source"}));
    expect(callbacks.onCopy).toHaveBeenCalledWith(film.endpoint,"Original metadata source copied");
    rerender(<TaskDetail task={{...current,id:"task-2"}} {...callbacks} />);
    expect(screen.getByRole("article",{name:"Task overview"}).querySelector("details")!.open).toBe(false);
  });
  it("places waiting state, schedule and actions in the sourcing node", () => {
    render(<TaskDetail task={{...task,stage:TaskStage.Sourcing,stageStatus:TaskStageStatus.Pending,sourcingNextCheckAt:"2026-10-21T02:00:00Z"}} {...props()} />);
    const overview = screen.getByRole("article",{name:"Task overview"});
    expect(overview.querySelector(".task-detail-summary .status-chip")).toBeNull();
    const current = overview.querySelector('.task-timeline > [aria-current="step"]')! as HTMLElement;
    expect(within(current).getByText("Waiting for resources")).toBeTruthy();
    expect(within(current).getByText("Next resource check")).toBeTruthy();
    expect(within(current).getByRole("button",{name:"Check resources now: DEMO-001"})).toBeTruthy();
  });

  it("keeps the blocking reason and next action in the failed node", () => {
    const currentTask={...task,stageStatus:TaskStageStatus.Blocked,stageErrorCode:"FILE_SELECTION_REQUIRED"};
    render(<TaskDetail task={currentTask} {...props()} />);
    const reason=taskFailureSummary(currentTask);
    const overview = screen.getByRole("article",{name:"Task overview"});
    expect(overview.querySelector(".task-detail-hero")).toBeNull();
    expect(overview.querySelectorAll(".task-timeline__failure")).toHaveLength(1);
    const current = overview.querySelector('.task-timeline > [aria-current="step"]')! as HTMLElement;
    expect(within(current).getByText(reason.title,{selector:"strong"})).toBeTruthy();
    expect(within(current).getByRole("button",{name:"Select movie files: DEMO-001"})).toBeTruthy();
  });

  it("marks the completed endpoint as in library without another status badge", () => {
    render(<TaskDetail task={{...task,stage:TaskStage.Completed,stageStatus:TaskStageStatus.Done}} {...props()} />);
    const overview = screen.getByRole("article",{name:"Task overview"});
    expect(overview.querySelector(".task-detail-summary .status-chip")).toBeNull();
    expect(within(overview).getByText("In library").closest("li")?.classList.contains("is-done")).toBe(true);
    expect(overview.querySelector('[aria-current="step"]')).toBeNull();
  });

  it("offers restarting a blocked ingest from selection and disables it while pending", async () => {
    const user = userEvent.setup();
    const onRestartTask = vi.fn();
    const current = { ...task, stage: TaskStage.Scanning, stageStatus: TaskStageStatus.Blocked };
    const { rerender } = render(<TaskDetail task={current} {...props()} onRestartTask={onRestartTask} />);
    await user.click(screen.getByRole("button", { name: "Reselect torrent" }));
    expect(onRestartTask).toHaveBeenCalledWith(task.id);
    rerender(<TaskDetail task={current} {...props()} onRestartTask={onRestartTask} pendingRestart />);
    expect((screen.getByRole("button", { name: "Deleting previous task files…" }) as HTMLButtonElement).disabled).toBe(true);
    rerender(<TaskDetail task={{ ...current, stageStatus: TaskStageStatus.Running }} {...props()} onRestartTask={onRestartTask} />);
    expect(screen.queryByRole("button", { name: "Reselect torrent" })).toBeNull();
    rerender(<TaskDetail task={{ ...current, stage: TaskStage.Completed }} {...props()} onRestartTask={onRestartTask} />);
    expect(screen.queryByRole("button", { name: "Reselect torrent" })).toBeNull();
  });
  it.each([TaskStage.PendingIngest, TaskStage.Transferring, TaskStage.Scanning, TaskStage.Completed] as const)("does not show download percentages as overall progress during %s", stage => {
    const current = { ...task, stage, progress: 1 };
    const { container } = render(<><TaskCard task={current} onOpen={vi.fn()} /><TaskDetail task={current} {...props()} /></>);
    expect(screen.queryAllByRole("progressbar")).toHaveLength(0);
    expect(container.textContent).not.toContain("100%");
    if (stage === TaskStage.Completed) {
      const overview = screen.getByRole("article", { name: "Task overview" });
      expect(within(overview).queryByText("Download and ingest completed.")).toBeNull();
      expect(overview.querySelector(".task-detail-hero")).toBeNull();
    }
  });

  it.each([
    ["downloading", TaskStageStatus.Running, 0.46, true],
    ["forcedDL", TaskStageStatus.Running, 0.46, true],
    ["pausedDL", TaskStageStatus.Running, 0.46, false],
    ["stoppedDL", TaskStageStatus.Running, 0.46, false],
    ["stalledDL", TaskStageStatus.Running, 0.46, false],
    ["queuedDL", TaskStageStatus.Running, 0.46, false],
    ["downloading", TaskStageStatus.Blocked, 0.46, false],
    ["downloading", TaskStageStatus.Running, 1, false]
  ])("animates only active download progress for %s / %s / %s", (qbittorrentState, stageStatus, progress, expected) => {
    const current = { ...task, qbittorrentState, stageStatus, progress };
    const { container } = render(<><TaskCard task={current} onOpen={vi.fn()} /><TaskDetail task={current} {...props()} /></>);
    expect(container.querySelectorAll(".progress-fill.is-animating")).toHaveLength(expected ? 2 : 0);
    for (const bar of container.querySelectorAll('[role="progressbar"]')) {
      expect(bar.getAttribute("aria-valuenow")).toBe(String(Math.round(progress * 100)));
    }
  });

  it("labels download progress and reports its numeric value accessibly", () => {
    render(<TaskCard task={task} onOpen={vi.fn()} />);
    expect(screen.getByText("Download progress 46%")).toBeTruthy();
    expect(screen.getByRole("progressbar").getAttribute("aria-valuenow")).toBe("46");
  });

  it.each([
    [TaskStage.Sourcing, undefined, "Select seed manually", "onResolveTask"],
    [TaskStage.Downloading, "FILE_SELECTION_REQUIRED", "Select movie files", "files"],
    [TaskStage.Transferring, "MEDIA_PREPARATION_FAILED", "Retry current stage", "onRetryTask"]
  ] as const)("uses the same next action in cards and details for %s / %s", async (stage, stageErrorCode, label, handler) => {
    const user = userEvent.setup();
    const current = { ...task, stage, stageErrorCode, stageStatus: TaskStageStatus.Blocked };
    const callbacks = props();
    const open = vi.fn();
    render(<>
      <TaskCard task={current} onOpen={open} onResolve={callbacks.onResolveTask} onRetry={callbacks.onRetryTask} />
      <TaskDetail task={current} {...callbacks} />
    </>);
    const actions = screen.getAllByRole("button", { name: `${label}: DEMO-001` });
    expect(actions).toHaveLength(2);
    await user.click(actions[0]);
    if (handler === "files") {
      expect(open).toHaveBeenCalledWith(task.id);
      const fileDetails = screen.getByRole("article", { name: "Torrent information" });
      expect(fileDetails.querySelector("summary")).toBeNull();
      await user.click(actions[1]);
      expect(document.activeElement).toBe(within(fileDetails).getByRole("checkbox"));
    } else {
      expect(callbacks[handler]).toHaveBeenCalledWith(task.id);
      callbacks[handler].mockClear();
      await user.click(actions[1]);
      expect(callbacks[handler]).toHaveBeenCalledWith(task.id);
      expect(open).not.toHaveBeenCalled();
    }
  });

  it("keeps three resource actions concise and exposes deletion directly", async () => {
    const callbacks = props();
    const open = vi.fn();
    const current = { ...task, stage: TaskStage.Sourcing, stageStatus: TaskStageStatus.Pending, sourcingNextCheckAt: "2026-10-21T02:00:00Z" };
    const { container, rerender } = render(<TaskCard task={current} onOpen={open} onResolve={callbacks.onResolveTask} onRetry={callbacks.onRetryTask} onDelete={callbacks.onDeleteTask} />);
    expect(container.querySelector("details, summary")).toBeNull();
    const choose = screen.getByRole("button", { name: "Select seed manually: DEMO-001" });
    const check = screen.getByRole("button", { name: "Check resources now: DEMO-001" });
    const remove = screen.getByRole("button", { name: "Delete task: DEMO-001" });
    expect(choose.textContent).toBe("Select seed");
    expect(choose.getAttribute("title")).toBe("Search for a seed or provide a magnet link");
    expect(check.getAttribute("title")).toBe("Search for available resources now");
    expect(remove.getAttribute("title")).toBe("Delete this task after confirmation");
    expect(check.textContent).toBe("Check");
    expect(remove.textContent).toBe("Delete");
    const user = userEvent.setup();
    await user.click(choose); await user.click(check); await user.click(remove);
    expect(callbacks.onResolveTask).toHaveBeenCalledWith(task.id);
    expect(callbacks.onRetryTask).toHaveBeenCalledWith(task.id);
    expect(callbacks.onDeleteTask).toHaveBeenCalledWith(task.id);
    expect(open).not.toHaveBeenCalled();
    rerender(<TaskCard task={current} onOpen={open} onDelete={callbacks.onDeleteTask} pendingDeleteId={task.id} />);
    const deleting = screen.getByRole("button", { name: "Delete task: DEMO-001" });
    expect((deleting as HTMLButtonElement).disabled).toBe(true);
    expect(deleting.getAttribute("aria-busy")).toBe("true");
    await user.click(deleting);
    expect(callbacks.onDeleteTask).toHaveBeenCalledTimes(1);
    expect(open).not.toHaveBeenCalled();
  });

  it("moves file locations to their sections and removes the task information section", async () => {
    const user = userEvent.setup();
    const callbacks = props();
    const path = "/stash/library/long-directory/movie.mkv";
    const download = "/qb/downloads/movie.mkv";
    const { container } = render(<TaskDetail task={{ ...task, stashScanPath: path, contentPath: download, savePath: "/qb/downloads", tags: "favorites", mojiTransferPath: "/moji/library/movie.mkv" }} {...callbacks} />);
    expect(screen.queryByRole("article", { name: "Task information" })).toBeNull();
    expect(screen.queryByRole("article",{name:"Stash movie information"})).toBeNull();
    const information = screen.getByRole("article", { name: "Task overview" });
    const torrent = screen.getByRole("article", { name: "Torrent information" });
    await user.click(information.querySelector(".task-detail-extra summary")!);
    expect(within(information).getByText(path).getAttribute("title")).toBe(path);
    expect(within(information).getByText(path).classList.contains("task-inline-value--truncate")).toBe(true);
    expect(within(torrent).getByText(download)).toBeTruthy();
    expect(within(torrent).getByText("favorites")).toBeTruthy();
    expect(within(torrent).getByText("Example movie")).toBeTruthy();
    expect(screen.queryByText("/moji/library/movie.mkv")).toBeNull();
    await user.click(within(information).getByRole("button", { name: "Copy Library location" }));
    expect(callbacks.onCopy).toHaveBeenCalledWith(path, "Library location copied");
    await user.click(torrent.querySelector(".torrent-information summary")!);
    await user.click(within(torrent).getByRole("button", { name: "Copy Content location" }));
    expect(callbacks.onCopy).toHaveBeenCalledWith(download, "Content location copied");
    expect(screen.queryByRole("button", { name: "Sync all task progress" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Scan pending-ingest tasks" })).toBeNull();
    const actions = screen.getByRole("group", { name: "Actions" });
    expect(container.lastElementChild).toBe(actions.closest("article"));
    expect(actions.querySelector("details, summary")).toBeNull();
    await user.click(within(actions).getByRole("button", { name: "Delete task: DEMO-001" }));
    expect(callbacks.onDeleteTask).toHaveBeenCalledWith(task.id);
  });

  it("copies operational diagnostics without adding persistent technical fields", async () => {
    const callbacks = props();
    const current = { ...task, stage: TaskStage.Scanning, stageStatus: TaskStageStatus.Blocked, stashIngestPhase: "GENERATE", stashGenerateJobId: "generate-1", stageErrorCode: "STASH_GENERATE_FAILED", stashScanError: "raw failure", mojiSourcePath: "/moji/downloads/movie", torrentUrl: "https://example.test/download?token=secret" };
    render(<TaskDetail task={current} {...callbacks} />);
    expect(screen.queryByText("generate-1")).toBeNull();
    expect(screen.queryByText("raw failure")).toBeNull();
    await userEvent.setup().click(screen.getByRole("button", { name: "Copy diagnostic information" }));
    const [value, message] = callbacks.onCopy.mock.calls[0];
    expect(JSON.parse(value)).toMatchObject({ id: task.id, stashGenerateJobId: "generate-1", stashScanError: "raw failure", mojiSourcePath: "/moji/downloads/movie" });
    expect(value).not.toContain("token=secret");
    expect(message).toBe("Diagnostic information copied");
    const currentStep = screen.getByText("Generate").closest("li")!;
    expect(currentStep.getAttribute("aria-current")).toBe("step");
    expect(within(currentStep).getByText(/then retry/)).toBeTruthy();
  });

  it("shows transfer details only in the active transfer substep", async () => {
    const callbacks = props();
    const current = { ...task, stage: TaskStage.Transferring, transferAction: "COPY", mojiTransferPath: "/moji/library/movie.mkv" };
    const { rerender } = render(<TaskDetail task={current} {...callbacks} />);
    const step = screen.getByText("Transfer").closest("li")!;
    expect(within(step).getByText("Copy")).toBeTruthy();
    expect(within(step).getByText(current.mojiTransferPath).getAttribute("title")).toBe(current.mojiTransferPath);
    await userEvent.setup().click(within(step).getByRole("button", { name: "Copy Destination" }));
    expect(callbacks.onCopy).toHaveBeenCalledWith(current.mojiTransferPath, "Destination copied");
    rerender(<TaskDetail task={{ ...current, stage: TaskStage.Completed, stageStatus: TaskStageStatus.Done }} {...callbacks} />);
    expect(screen.queryByText(current.mojiTransferPath)).toBeNull();
  });

  it("keeps all three stages visible and embeds compact substeps in the active ingest stage", () => {
    render(<TaskDetail task={{ ...task, stage: TaskStage.Scanning, stashIngestPhase: "IDENTIFY", progress: 1 }} {...props()} />);
    const overview = screen.getByRole("article", { name: "Task overview" });
    expect(within(overview).getByRole("list", { name: "Lifecycle" }).children).toHaveLength(3);
    expect(within(overview).getByRole("region", { name: "Ingest progress" })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Lifecycle" })).toBeNull();
    expect(screen.getByRole("region", { name: "Ingest progress" })).toBeTruthy();
    expect(within(screen.getByRole("list", { name: "Ingest steps" })).getAllByRole("listitem")).toHaveLength(6);
    expect(screen.getByText("Identify").closest("li")?.getAttribute("aria-current")).toBe("step");
    expect(screen.queryByText("Expand ingest steps")).toBeNull();
  });

  it("places download progress, stage details, and key dates inside the overview", () => {
    const { container } = render(<TaskDetail task={task} {...props()} />);
    const overview = screen.getByRole("article", { name: "Task overview" });
    expect(container.firstElementChild).toBe(overview);
    expect(within(overview).getByRole("heading", { name: "Task overview" })).toBeTruthy();
    expect(within(overview).queryByRole("heading", { name: "DEMO-001" })).toBeNull();
    const progress = within(overview).getByRole("progressbar", { name: "Downloading" });
    expect(progress.closest("li")?.getAttribute("aria-current")).toBe("step");
    expect(progress.getAttribute("aria-valuenow")).toBe("46");
    expect(within(overview).getByText("Created")).toBeTruthy();
    expect(within(overview).getByText("Last updated")).toBeTruthy();
    expect(within(overview).getAllByText("Download progress 46%")).toHaveLength(1);
    expect(screen.queryByRole("heading", { name: "Lifecycle" })).toBeNull();
  });

  it("never fabricates milestone timestamps from updatedAt", () => {
    const current = { ...task, stage: TaskStage.Completed, sourcingCompletedAt: "2026-10-17T02:10:00Z", downloadCompletedAt: "2026-10-17T02:30:00Z", stashScanStartedAt: "2026-10-17T02:35:00Z", ingestCompletedAt: "2026-10-17T03:00:00Z" };
    const steps = taskLifecycle(current, taskFailureSummary(current));
    expect(steps.filter(step => step.time).map(step => [step.key, step.time])).toEqual([
      ["sourcing", current.sourcingCompletedAt], ["downloading", current.downloadCompletedAt], ["scanning", current.stashScanStartedAt], ["completed", current.ingestCompletedAt]
    ]);
  });
});

const selectionReport: NonNullable<DashboardTask["selectionReport"]> = {
  groupName: "Daily", end: "PREFERRED", partial: false,
  evaluations: [{ selected: true, title: "DEMO-001 preferred resource", status: "ACCEPTABLE",
    size: 1024, seeders: 0, tierName: "First choice", reasons: ["TIER_MATCHED"], excludedRules: [] }]
};

describe("selection explanation in the sourcing stage", () => {
  it.each([TaskStage.Sourcing, TaskStage.Downloading, TaskStage.Completed])("opens a report dialog from sourcing when the task is %s", async stage => {
    const user = userEvent.setup();
    const { rerender } = render(<TaskDetail task={{ ...task, stage, selectionReport }} {...props()} />);
    const overview = screen.getByRole("article", { name: "Task overview" });
    const sourcing = overview.querySelector(".task-timeline > li")! as HTMLElement;
    const entry = within(sourcing).getByRole("button", { name: "View selection report" });
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(entry.querySelector("svg")).toBeNull();
    expect(sourcing.querySelector(".task-timeline__actions button")).toBe(entry);
    expect(sourcing.querySelector(".task-selection-report")).toBeNull();
    await user.click(entry);
    const report = screen.getByRole("dialog", { name: "Selection report · DEMO-001" });
    expect(report.contains(document.activeElement)).toBe(true);
    expect(within(report).getByText("First-priority resource found")).toBeTruthy();
    const candidate = report.querySelector<HTMLElement>(".task-selection-report__candidate")!;
    expect(candidate.querySelector("details")).toBeNull();
    expect(within(candidate).getByText("Priority tier matched: First choice")).toBeTruthy();
    expect(within(candidate).queryByText("Match reasons")).toBeNull();
    expect(within(candidate).queryByText("Match evidence")).toBeNull();
    expect(within(candidate).queryByText("Priority tier matched")).toBeNull();
    expect(within(report).getAllByText("Daily")).toHaveLength(1);
    expect(within(candidate).queryByText("Daily")).toBeNull();
    expect(within(report).getByRole("img", {name:"Meets selection conditions"})).toBeTruthy();
    expect(within(report).getAllByText("DEMO-001 preferred resource")).toHaveLength(1);
    expect(within(candidate).getByText("0", {exact:true})).toBeTruthy();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(document.activeElement).toBe(entry);
    await user.click(entry);
    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(document.activeElement).toBe(entry);
    await user.click(entry);
    rerender(<TaskDetail task={{ ...task, id: "another-task", stage, selectionReport }} {...props()} />);
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("omits the entry when the runtime report is absent", () => {
    render(<TaskDetail task={{ ...task, selectionReport: null }} {...props()} />);
    expect(screen.queryByRole("button", { name: "View selection report" })).toBeNull();
  });

  it("shows an incomplete empty result without an empty selected-resource field", async () => {
    render(<TaskDetail task={{ ...task, selectionReport: { groupName: "Daily", end: "NO_ACCEPTABLE_RESOURCE", partial: true, evaluations: [] } }} {...props()} />);
    await userEvent.setup().click(screen.getByRole("button", { name: "View selection report" }));
    const report = screen.getByRole("dialog", { name: "Selection report · DEMO-001" });
    expect(within(report).getByText("No acceptable resource found")).toBeTruthy();
    expect(within(report).queryByText("Selected resource")).toBeNull();
    expect(report.querySelector("ol")).toBeNull();
  });
});


describe("selection report eligibility and historical context",()=>{
 it("keeps user selection distinct from automatic eligibility",async()=>{
  const report={...selectionReport,end:"USER_SELECTED",evaluations:selectionReport.evaluations.map(e=>({...e,status:"REJECTED",tierName:"",reasons:["EXCLUDED_RULE_MATCHED"],excludedRules:["Unwanted labels"]}))};
  render(<TaskDetail task={{...task,selectionReport:report}} {...props()} />);
  await userEvent.setup().click(screen.getByRole("button",{name:"View selection report"}));
  const dialog=screen.getByRole("dialog");
  expect(within(dialog).getByText("User-selected resource")).toBeTruthy();
  expect(within(dialog).queryByText("Selected resource",{exact:true})).toBeNull();
  expect(dialog.querySelector(".task-selection-report__candidate.is-selected")).toBeTruthy();
  expect(within(dialog).getByRole("img",{name:"Does not meet selection conditions"})).toBeTruthy();
  const candidate=dialog.querySelector(".task-selection-report__candidate")! as HTMLElement;
  expect(within(candidate).queryByText("Priority tier",{exact:true})).toBeNull();
  expect(within(candidate).getByText("Excluded by: Unwanted labels")).toBeTruthy();
 });
 it("shows explicitly unused groups without relying on candidate presence",async()=>{
  render(<TaskDetail task={{...task,selectionReport:{...selectionReport,groupName:"",evaluations:[]}}} {...props()} />);
  await userEvent.setup().click(screen.getByRole("button",{name:"View selection report"}));
  expect(within(screen.getByRole("dialog")).getByText("No rule group used")).toBeTruthy();
  expect(within(screen.getByRole("dialog")).queryByText("First-priority resource found")).toBeNull();
 });
});


describe("concise selection report reasons",()=>{
 it.each([
  ["FALLBACK_ALLOWED",[],"Fallback"],
  ["EXCLUDED_RULE_MATCHED",[],"Excluded by: Rule name not recorded"]
 ] as const)("shows %s without per-rule results",async(reason,excludedRules,label)=>{
  render(<TaskDetail task={{...task,selectionReport:{...selectionReport,evaluations:selectionReport.evaluations.map(e=>({...e,tierName:"",reasons:[reason],excludedRules:[...excludedRules]}))}}} {...props()}/>);
  await userEvent.setup().click(screen.getByRole("button",{name:"View selection report"}));
  const dialog=screen.getByRole("dialog");expect(within(dialog).getByText(label)).toBeTruthy();
  expect(dialog.querySelector("details")).toBeNull();expect(within(dialog).queryByText("Match evidence")).toBeNull();
 });
});


describe("selection report candidate emphasis",()=>{
 it("highlights only the server-selected item even when titles repeat, keeping eligibility independent",async()=>{
  const original=selectionReport.evaluations[0];
  render(<TaskDetail task={{...task,selectionReport:{...selectionReport,evaluations:[{...original,selected:false},{...original,selected:true,status:"REJECTED",tierName:"",reasons:["EXCLUDED_RULE_MATCHED"],excludedRules:["Label"]}]}}} {...props()}/>);
  await userEvent.setup().click(screen.getByRole("button",{name:"View selection report"}));
  const dialog=screen.getByRole("dialog"),candidates=dialog.querySelectorAll(".task-selection-report__candidate");
  expect(candidates[0].classList.contains("is-selected")).toBe(false);expect(candidates[1].classList.contains("is-selected")).toBe(true);
  expect(within(dialog).queryByText("Selected resource")).toBeNull();
  expect(candidates[1].querySelector(".task-selection-report__watermark")?.getAttribute("aria-hidden")).toBe("true");
  expect(within(candidates[0] as HTMLElement).getByRole("img",{name:"Meets selection conditions"}).textContent).toBe("");
  expect(within(candidates[1] as HTMLElement).getByRole("img",{name:"Does not meet selection conditions"}).textContent).toBe("");
 });
});
