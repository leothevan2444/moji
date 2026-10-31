// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import type { UpcomingRelease } from "./UpcomingReleaseCard";
import { UpcomingReleaseDetail } from "./UpcomingReleaseDetail";

beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(cleanup);
it.each(["QUEUED", "DOWNLOADED", "SKIPPED"])("offers review only for a handed-off review decision (%s)", decision => {
  const release: UpcomingRelease = { key: "future", code: "ABC-123", title: "Future film", date: "2026-11-12", imageUrl: "", sourceUrl: "", state: "HANDED_OFF", decision, errorCode: "", metadataCheckedAt: "", performers: [] };
  render(<MemoryRouter><UpcomingReleaseDetail release={release} status="Sent to release processing" sourceUrl={null} refreshing={false} feedback="" onRefresh={vi.fn()} onClose={vi.fn()} /></MemoryRouter>);
  expect(screen.queryByRole("link", {name:"Release review"}) !== null).toBe(decision === "QUEUED");
  expect((screen.getByRole("button", {name:"Refresh"}) as HTMLButtonElement).disabled).toBe(true);
});
