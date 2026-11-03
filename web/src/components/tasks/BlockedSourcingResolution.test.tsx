// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CombinedError } from "urql";
import i18n from "../../i18n/i18n";
import type { DashboardTask } from "../../utils";
import { SourcingResolutionDrawer } from "../drawers/SourcingResolutionDrawer";

const mocks = vi.hoisted(() => ({ resolve: vi.fn(), searching: false }));
vi.mock("urql", async importOriginal => ({
  ...await importOriginal<typeof import("urql")>(),
  useMutation: () => [{ fetching: false }, mocks.resolve],

}));

vi.mock("./TaskIndexerSearch",()=>({TaskIndexerSearch:()=> <div data-testid="indexer-search" aria-busy={mocks.searching}/>}));

const task = { id: "waiting", code: "OFJE-657", stage: "SOURCING", stageStatus: "PENDING", sourcingNextCheckAt: "2026-10-21T00:00:00Z" } as DashboardTask;
beforeEach(async () => {
  await i18n.changeLanguage("en");
  vi.clearAllMocks();
  mocks.searching = false;
  mocks.resolve.mockResolvedValue({ data: { resolveBlockedSourcingTask: { id: task.id } } });
});
afterEach(cleanup);

describe("manual resources for waiting tasks", () => {
  it("keeps direct magnet entry independent from indexer work", async () => {
    mocks.searching = true;
    const onResolved = vi.fn();
    render(<SourcingResolutionDrawer task={task} onResolved={onResolved} />);
    expect(screen.getByTestId("indexer-search")).toBeTruthy();
    await userEvent.setup().type(screen.getByRole("textbox", { name: "Magnet link" }), "magnet:?xt=urn:btih:657&dn=OFJE657");
    await userEvent.setup().click(screen.getByRole("button", { name: "Use magnet link" }));
    expect(mocks.resolve).toHaveBeenCalledWith({ id: task.id, input: expect.objectContaining({ torrentUrl: "magnet:?xt=urn:btih:657&dn=OFJE657" }) });
    expect(onResolved).toHaveBeenCalledTimes(1);
  });

  it("opens a file picker from one button and submits the selected file immediately", async () => {
    const onResolved = vi.fn(), onNotify = vi.fn();
    const { container } = render(<SourcingResolutionDrawer task={task} onResolved={onResolved} onNotify={onNotify} />);
    const input = screen.getByLabelText("Upload torrent file") as HTMLInputElement;
    expect(input.hidden).toBe(true);
    const clicked = vi.spyOn(input, "click");
    await userEvent.setup().click(screen.getByRole("button", { name: "Upload torrent file" }));
    expect(clicked).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("button", { name: "Use torrent file" })).toBeNull();
    expect(container.querySelector(".sourcing-resolution__magnet")?.contains(input)).toBe(true);
    const file = new File(["torrent bytes"], "OFJE657.torrent", { type: "application/x-bittorrent" });
    await userEvent.setup().upload(input, file);
    expect(mocks.resolve).toHaveBeenCalledWith({ id: task.id, input: { torrentFile: file } });
    expect(input.value).toBe("");
    expect(onNotify).toHaveBeenCalledWith("tone-success", "Resource verified and submitted for download.");
    expect(onResolved).toHaveBeenCalledTimes(1);
  });




  it("rejects empty, oversized or wrong-extension files before submission", () => {
    const onNotify = vi.fn();
    render(<SourcingResolutionDrawer task={task} onResolved={vi.fn()} onNotify={onNotify} />);
    for (const file of [new File([], "empty.torrent"), new File([new Uint8Array(8*1024*1024+1)], "large.torrent"), new File(["x"], "movie.html")]) {
      fireEvent.change(screen.getByLabelText("Upload torrent file"), { target: { files: [file] } });
      expect(screen.getByRole("alert").textContent).toContain("non-empty .torrent file");
      expect((screen.getByRole("button", { name: "Upload torrent file" }) as HTMLButtonElement).disabled).toBe(false);
    }
    expect(onNotify).toHaveBeenCalledTimes(3);
    expect(mocks.resolve).not.toHaveBeenCalled();
  });

  it("explains a rejected upload and allows retrying the same file", async () => {
    const onResolved = vi.fn(), onNotify = vi.fn();
    mocks.resolve.mockResolvedValueOnce({ error: new CombinedError({ graphQLErrors: [{ message: "request failed", extensions: { code: "CANDIDATE_CODE_MISMATCH" } }] }) });
    render(<SourcingResolutionDrawer task={task} onResolved={onResolved} onNotify={onNotify} />);
    const file = new File(["seed"], "movie.torrent", { type: "application/x-bittorrent" });
    const input = screen.getByLabelText("Upload torrent file");
    await userEvent.setup().upload(input, file);
    expect(screen.getByRole("alert").textContent).toContain("resource code differs");
    expect(onNotify).toHaveBeenCalledWith("tone-danger", expect.stringContaining("resource code differs"));
    expect(onResolved).not.toHaveBeenCalled();
    await userEvent.setup().upload(input, file);
    expect(mocks.resolve).toHaveBeenCalledTimes(2);
    expect(onResolved).toHaveBeenCalledTimes(1);
  });

  it("disables all submit paths during validation and closes only after success", async () => {
    let finish: (value: unknown) => void = () => {};
    mocks.resolve.mockImplementation(() => new Promise(accept => { finish = accept; }));
    const onResolved = vi.fn();
    render(<SourcingResolutionDrawer task={task} onResolved={onResolved} />);
    await userEvent.setup().upload(screen.getByLabelText("Upload torrent file"), new File(["seed"], "movie.torrent"));
    expect(screen.getAllByRole("button").every(button => (button as HTMLButtonElement).disabled)).toBe(true);
    expect(screen.getByRole("status").textContent).toContain("Validating movie.torrent");
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
