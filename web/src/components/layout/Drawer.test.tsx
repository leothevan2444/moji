// @vitest-environment jsdom
import { useState } from "react";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import i18n from "../../i18n/i18n";
import { Drawer } from "./Drawer";
import { ConfirmDeleteDrawer } from "../drawers/ConfirmDeleteDrawer";

function Example() {
  const [open, setOpen] = useState(false);
  const [nested, setNested] = useState(false);
  return <>
    <button onClick={() => setOpen(true)}>Open task</button>
    {open ? <Drawer visibleDrawer="task" closing={false} title="Task details" onClose={() => setOpen(false)}>
      <details><summary>Technical details</summary><button>Hidden copy</button></details>
      <button onClick={() => setNested(true)}>Delete task</button>
    </Drawer> : null}
    {nested ? <Drawer visibleDrawer="confirm" closing={false} title="Confirm deletion" onClose={() => setNested(false)}>
      <ConfirmDeleteDrawer taskLabel="Example movie" deletePolicy="KEEP_ONLY" pending={false}
        onConfirm={() => setNested(false)} onCancel={() => setNested(false)} />
    </Drawer> : null}
  </>;
}

beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(cleanup);

describe("modal keyboard navigation", () => {
  it("focuses the dialog, cycles Tab in both directions, skips collapsed fields, and restores its opener", async () => {
    const user = userEvent.setup();
    render(<Example />);
    const opener = screen.getByRole("button", { name: "Open task" });
    await user.click(opener);
    const dialog = screen.getByRole("dialog", { name: "Task details" });
    const close = screen.getByRole("button", { name: "Close" });
    expect(document.activeElement).toBe(close);
    expect(dialog.getAttribute("aria-modal")).toBe("true");
    expect(document.body.style.overflow).toBe("hidden");
    await user.tab({ shift: true });
    expect(document.activeElement).toBe(screen.getByRole("button", { name: "Delete task" }));
    await user.tab();
    expect(document.activeElement).toBe(close);
    await user.tab();
    expect(document.activeElement).toBe(screen.getByText("Technical details"));
    await user.tab();
    expect(document.activeElement).toBe(screen.getByRole("button", { name: "Delete task" }));
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(document.activeElement).toBe(opener);
    expect(document.body.style.overflow).toBe("");
  });

  it("closes only the topmost dialog and restores focus inside its parent", async () => {
    const user = userEvent.setup();
    render(<Example />);
    await user.click(screen.getByText("Open task"));
    const deleteButton = screen.getByText("Delete task");
    await user.click(deleteButton);
    const confirmation = screen.getByRole("dialog", { name: "Confirm deletion" });
    expect(confirmation.classList.contains("drawer--confirm")).toBe(true);
    expect(confirmation.querySelector(".drawer-card")).toBeNull();
    expect(screen.getByText("Example movie")).toBeTruthy();
    expect(screen.getByText("Only the Moji task records will be deleted.")).toBeTruthy();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog", { name: "Confirm deletion" })).toBeNull();
    expect(screen.getByRole("dialog", { name: "Task details" })).toBeTruthy();
    expect(document.activeElement).toBe(deleteButton);
    expect(document.body.style.overflow).toBe("hidden");
    await user.keyboard("{Escape}");
    expect(document.activeElement).toBe(screen.getByText("Open task"));
  });
});
