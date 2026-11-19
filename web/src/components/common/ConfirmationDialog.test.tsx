// @vitest-environment jsdom
import { useState } from "react";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { ConfirmationDialog } from "./ConfirmationDialog";

beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(cleanup);

it("opens outside its form, traps focus and cancels without submitting the operation", async () => {
  const confirm = vi.fn();
  function Example() {
    const [open, setOpen] = useState(false);
    return <form onSubmit={event => { event.preventDefault(); confirm(); }}>
      <button type="button" onClick={() => setOpen(true)}>Clear cached files</button>
      {open ? <ConfirmationDialog title="Clear cache?" confirmLabel="Clear" cancelLabel="Cancel" pending={false} onConfirm={confirm} onCancel={() => setOpen(false)}>
        <p>Original files stay in place.</p>
      </ConfirmationDialog> : null}
    </form>;
  }
  const user = userEvent.setup();
  render(<Example />);
  const opener = screen.getByRole("button", { name: "Clear cached files" });
  await user.click(opener);
  const dialog = screen.getByRole("dialog", { name: "Clear cache?" });
  expect(dialog.closest("form")).toBeNull();
  await user.tab({ shift: true });
  expect(document.activeElement).toBe(within(dialog).getByRole("button", { name: "Clear" }));
  await user.keyboard("{Escape}");
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(document.activeElement).toBe(opener);
  expect(confirm).not.toHaveBeenCalled();
});

it("blocks dismissal and duplicate confirmation while pending, then keeps errors available for retry", async () => {
  const confirm = vi.fn(), cancel = vi.fn();
  const user = userEvent.setup();
  const props = { title: "Clear cache?", confirmLabel: "Clear", cancelLabel: "Cancel", onConfirm: confirm, onCancel: cancel };
  const view = render(<ConfirmationDialog {...props} pending><p>Cached files only.</p></ConfirmationDialog>);
  await user.keyboard("{Escape}");
  await user.click(screen.getByRole("button", { name: "Clear" }));
  await user.click(document.querySelector(".drawer-scrim")!);
  expect(cancel).not.toHaveBeenCalled();
  expect(confirm).not.toHaveBeenCalled();
  view.rerender(<ConfirmationDialog {...props} pending={false} error="Cache could not be cleared."><p>Cached files only.</p></ConfirmationDialog>);
  expect(screen.getByRole("alert").textContent).toBe("Cache could not be cleared.");
  await user.click(screen.getByRole("button", { name: "Clear" }));
  expect(confirm).toHaveBeenCalledOnce();
});
