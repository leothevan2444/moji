// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { UpcomingReleaseCard, type UpcomingRelease } from "./UpcomingReleaseCard";

const release: UpcomingRelease = { key: "future", code: "ABC-123", title: "Future film", date: "2026-11-12", imageUrl: "", sourceUrl: "", state: "WAITING", decision: "DOWNLOADED", errorCode: "", metadataCheckedAt: "2026-10-31T00:00:00Z", performers: [{ id: "p1", name: "Actor" }] };
beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(cleanup);
it("opens with the keyboard while performer links remain independent", async () => {
  const open = vi.fn();
  render(<MemoryRouter><UpcomingReleaseCard release={release} status={null} onOpen={open} /></MemoryRouter>);
  const user = userEvent.setup();
  const button = screen.getByRole("button", { name: "View ABC-123" });
  button.focus(); await user.keyboard("{Enter}");
  expect(open).toHaveBeenCalledTimes(1);
  await user.click(screen.getByRole("link", { name: "Actor" }));
  expect(open).toHaveBeenCalledTimes(1);
  expect(button.querySelector("a")).toBeNull();
});
