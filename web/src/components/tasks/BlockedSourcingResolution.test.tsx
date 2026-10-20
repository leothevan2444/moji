// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CombinedError } from "urql";
import i18n from "../../i18n/i18n";
import type { DashboardTask } from "../../utils";
import { SourcingResolutionDrawer } from "../drawers/SourcingResolutionDrawer";

const { resolve, search, queryCalls } = vi.hoisted(() => ({ resolve: vi.fn(), search: vi.fn(), queryCalls: vi.fn() }));
vi.mock("urql", async importOriginal => ({
  ...await importOriginal<typeof import("urql")>(),
  useMutation: () => [{ fetching: false }, resolve],
  useQuery: (options: { pause: boolean }) => {
    queryCalls(options);
    return [{ fetching: false, data: options.pause ? undefined : { blockedTaskTorrentCandidates: [{
      title: "OFJE657 indexed", tracker: "indexer", link: "http://indexer.invalid/file", magnetUri: "magnet:?xt=urn:btih:657&dn=OFJE657", infoHash: "657", size: 100, seeders: 5, peers: 1
    }] } }, search];
  }
}));

const task = { id: "waiting", code: "OFJE-657", stage: "SOURCING", stageStatus: "PENDING", sourcingNextCheckAt: "2026-10-21T00:00:00Z" } as DashboardTask;
beforeEach(async () => {
  await i18n.changeLanguage("en");
  vi.clearAllMocks();
  resolve.mockResolvedValue({ data: { resolveBlockedSourcingTask: { id: task.id } } });
});
afterEach(cleanup);

describe("manual resources for waiting tasks", () => {
  it("submits a manual magnet on the existing task without searching first", async () => {
    const onResolved = vi.fn();
    render(<SourcingResolutionDrawer task={task} onResolved={onResolved} />);
    await userEvent.setup().type(screen.getByRole("textbox", { name: "Magnet link" }), "magnet:?xt=urn:btih:657&dn=OFJE657");
    await userEvent.setup().click(screen.getByRole("button", { name: "Use magnet link" }));
    expect(resolve).toHaveBeenCalledWith({ id: task.id, input: expect.objectContaining({ torrentUrl: "magnet:?xt=urn:btih:657&dn=OFJE657" }) });
    expect(queryCalls.mock.calls.every(([options]) => options.pause)).toBe(true);
    expect(onResolved).toHaveBeenCalledTimes(1);
  });

  it("uploads the File as a multipart variable without sending the magnet field", async () => {
    const onResolved = vi.fn();
    render(<SourcingResolutionDrawer task={task} onResolved={onResolved} />);
    const file = new File(["torrent bytes"], "OFJE657.torrent", { type: "application/x-bittorrent" });
    await userEvent.setup().upload(screen.getByLabelText(/Upload torrent file/), file);
    await userEvent.setup().click(screen.getByRole("button", { name: "Use torrent file" }));
    expect(resolve).toHaveBeenCalledWith({ id: task.id, input: { torrentFile: file } });
    expect(onResolved).toHaveBeenCalledTimes(1);
  });

  it("lets the user search and select an indexer result", async () => {
    render(<SourcingResolutionDrawer task={task} onResolved={vi.fn()} />);
    await userEvent.setup().click(screen.getByRole("button", { name: "Search candidates" }));
    expect(screen.getByText("OFJE657 indexed")).toBeTruthy();
    await userEvent.setup().click(screen.getByRole("button", { name: "Select" }));
    expect(resolve).toHaveBeenCalledWith({ id: task.id, input: expect.objectContaining({ torrentUrl: "magnet:?xt=urn:btih:657&dn=OFJE657", title: "OFJE657 indexed", tracker: "indexer" }) });
  });

  it("rejects empty, oversized or wrong-extension files before submission", async () => {
    render(<SourcingResolutionDrawer task={task} onResolved={vi.fn()} />);
    for (const file of [new File([], "empty.torrent"), new File([new Uint8Array(8*1024*1024+1)], "large.torrent"), new File(["x"], "movie.html")]) {
      fireEvent.change(screen.getByLabelText(/Upload torrent file/), { target: { files: [file] } });
      expect(screen.getByRole("alert").textContent).toContain("non-empty .torrent file");
      expect((screen.getByRole("button", { name: "Use torrent file" }) as HTMLButtonElement).disabled).toBe(true);
    }
    expect(resolve).not.toHaveBeenCalled();
  });

  it("keeps a rejected resource in the panel and explains a code mismatch", async () => {
    const onResolved = vi.fn();
    resolve.mockResolvedValue({ error: new CombinedError({ graphQLErrors: [{ message: "request failed", extensions: { code: "CANDIDATE_CODE_MISMATCH" } }] }) });
    render(<SourcingResolutionDrawer task={task} onResolved={onResolved} />);
    fireEvent.change(screen.getByRole("textbox", { name: "Magnet link" }), { target: { value: "magnet:?xt=urn:btih:667&dn=OFJE667" } });
    await userEvent.setup().click(screen.getByRole("button", { name: "Use magnet link" }));
    expect(screen.getByRole("alert").textContent).toContain("resource code differs");
    expect(onResolved).not.toHaveBeenCalled();
  });

  it("disables all submit paths during an upload and closes only after success", async () => {
    let finish: (value: unknown) => void = () => {};
    resolve.mockImplementation(() => new Promise(accept => { finish = accept; }));
    const onResolved = vi.fn();
    render(<SourcingResolutionDrawer task={task} onResolved={onResolved} />);
    await userEvent.setup().upload(screen.getByLabelText(/Upload torrent file/), new File(["seed"], "movie.torrent"));
    await userEvent.setup().click(screen.getByRole("button", { name: "Use torrent file" }));
    expect(screen.getAllByRole("button").every(button => (button as HTMLButtonElement).disabled)).toBe(true);
    expect(onResolved).not.toHaveBeenCalled();
    finish({ data: { resolveBlockedSourcingTask: { id: task.id } } });
    await waitFor(() => expect(onResolved).toHaveBeenCalledTimes(1));
  });

  it("does not allow resource replacement after downloading starts", () => {
    render(<SourcingResolutionDrawer task={{ ...task, stage: "DOWNLOADING" } as DashboardTask} onResolved={vi.fn()} />);
    expect(screen.queryByRole("textbox")).toBeNull();
    expect(screen.getByText("This task cannot be resolved manually")).toBeTruthy();
  });
});
