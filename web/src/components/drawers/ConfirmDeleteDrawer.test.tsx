// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { ConfirmDeleteDrawer } from "./ConfirmDeleteDrawer";

beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(cleanup);

it.each(["KEEP_ONLY", "REMOVE_TORRENT", "REMOVE_TORRENT_AND_FILES"] as const)("keeps the one-time Stash choice separate from %s", async deletePolicy => {
  const user = userEvent.setup();
  const onConfirm = vi.fn();
  const props = { deletePolicy, pending: false, hasStashScenes: true, onConfirm, onCancel: vi.fn(), onConfigure: vi.fn() };
  const view = render(<ConfirmDeleteDrawer {...props} />);
  const choice = screen.getByRole("checkbox") as HTMLInputElement;
  expect(choice.checked).toBe(false);
  await user.click(screen.getByRole("button", { name: "Confirm deletion" }));
  expect(onConfirm).toHaveBeenLastCalledWith(false);
  await user.click(choice);
  await user.click(screen.getByRole("button", { name: "Confirm deletion" }));
  expect(onConfirm).toHaveBeenLastCalledWith(true);
  view.rerender(<ConfirmDeleteDrawer {...props} pending />);
  expect((screen.getByRole("checkbox") as HTMLInputElement).disabled).toBe(true);
  view.unmount();
  render(<ConfirmDeleteDrawer {...props} />);
  expect((screen.getByRole("checkbox") as HTMLInputElement).checked).toBe(false);
});

it("disables unlinked scenes and uses the browser's default tooltip", async () => {
  const user = userEvent.setup();
  const onConfirm = vi.fn();
  render(<ConfirmDeleteDrawer deletePolicy="KEEP_ONLY" pending={false} hasStashScenes={false}
    onConfirm={onConfirm} onCancel={vi.fn()} onConfigure={vi.fn()} />);
  expect((screen.getByRole("checkbox") as HTMLInputElement).disabled).toBe(true);
  const reason = screen.getByLabelText("No Stash scene linked yet");
  expect(reason.getAttribute("title")).toBe("No Stash scene linked yet");
  await user.hover(reason);
  expect(screen.queryByRole("tooltip")).toBeNull();
  await user.unhover(reason);
  await user.tab();
  expect(document.activeElement).toBe(reason);
  expect(screen.queryByRole("tooltip")).toBeNull();
  await user.click(screen.getByRole("button", { name: "Confirm deletion" }));
  expect(onConfirm).toHaveBeenCalledExactlyOnceWith(false);
});

it("clears a checked choice when the last scene link disappears", async () => {
  const user = userEvent.setup();
  const props = { deletePolicy: "KEEP_ONLY" as const, pending: false, onConfirm: vi.fn(), onCancel: vi.fn(), onConfigure: vi.fn() };
  const view = render(<ConfirmDeleteDrawer {...props} hasStashScenes />);
  await user.click(screen.getByRole("checkbox"));
  view.rerender(<ConfirmDeleteDrawer {...props} hasStashScenes={false} />);
  await user.click(screen.getByRole("button", { name: "Confirm deletion" }));
  expect(props.onConfirm).toHaveBeenLastCalledWith(false);
  view.rerender(<ConfirmDeleteDrawer {...props} hasStashScenes />);
  expect((screen.getByRole("checkbox") as HTMLInputElement).checked).toBe(false);
});
