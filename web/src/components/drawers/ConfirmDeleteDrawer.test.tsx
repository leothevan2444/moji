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
  const props = { deletePolicy, pending: false, onConfirm, onCancel: vi.fn(), onConfigure: vi.fn() };
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
