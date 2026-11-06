// @vitest-environment jsdom
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { SearchSiteFailure } from "./SearchSiteFailure";

beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(() => { cleanup(); vi.restoreAllMocks(); });

it("shows the failure on hover and copies the site, reason and stable error code", async () => {
  const user = userEvent.setup();
  const write = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue();
  const notify = vi.fn();
  render(<SearchSiteFailure onNotify={notify} name="Example" label="Failed" code="INDEXER_TIMEOUT" />);
  const button = screen.getByRole("button");
  await user.hover(button);
  expect(screen.getByRole("tooltip").textContent).toContain("The site request timed out.");
  const original = screen.getByRole("tooltip").textContent;
  await user.click(button);
  expect(screen.getByRole("tooltip").textContent).toBe(original);
  await user.unhover(button);
  expect(screen.queryByRole("tooltip")).toBeNull();
  await user.hover(button);
  expect(screen.getByRole("tooltip").textContent).toBe(original);
  expect(write).toHaveBeenCalledWith("Example: The site request timed out. Try again. (INDEXER_TIMEOUT)");
  await waitFor(() => expect(notify).toHaveBeenCalledWith("tone-success", "Error details copied"));
});

it("reports denied clipboard access and lets the user retry with the keyboard", async () => {
  const user = userEvent.setup();
  const write = vi.spyOn(navigator.clipboard, "writeText").mockRejectedValueOnce(new Error("denied")).mockResolvedValue();
  const notify = vi.fn();
  render(<SearchSiteFailure onNotify={notify} name="Example" label="Failed" code="INDEXER_TIMEOUT" />);
  await user.tab();
  await user.keyboard("{Enter}");
  await waitFor(() => expect(notify).toHaveBeenCalledWith("tone-danger", expect.stringContaining("Copy failed.")));
  await user.keyboard(" ");
  await waitFor(() => expect(notify).toHaveBeenCalledWith("tone-success", "Error details copied"));
  expect(write).toHaveBeenCalledTimes(2);
  await user.keyboard("{Escape}");
  expect(screen.queryByRole("tooltip")).toBeNull();
});
