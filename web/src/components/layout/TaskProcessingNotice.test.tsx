// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { TaskProcessingNotice } from "./TaskProcessingNotice";

const state = vi.hoisted(() => ({ enabled: undefined as boolean | undefined }));
vi.mock("urql", () => ({ useQuery: () => [{ data: state.enabled === undefined ? undefined : { settings: { automation: { taskProcessingEnabled: state.enabled } } } }] }));
beforeEach(async () => { state.enabled = undefined; await i18n.changeLanguage("en"); });
afterEach(cleanup);

it("shows a persistent pause notice with a settings link and clears it after enabling", () => {
  const view = render(<MemoryRouter><TaskProcessingNotice /></MemoryRouter>);
  expect(screen.queryByRole("status")).toBeNull();
  state.enabled = false;
  view.rerender(<MemoryRouter><TaskProcessingNotice /></MemoryRouter>);
  expect(screen.getByRole("status").textContent).toContain("Automatic task processing is paused");
  expect(screen.getByRole("status").textContent).toContain("Resource searches and subscription checks are unaffected");
  expect(screen.getByRole("link", { name: "Open automation settings" }).getAttribute("href")).toBe("/settings/automation");
  state.enabled = true;
  view.rerender(<MemoryRouter><TaskProcessingNotice /></MemoryRouter>);
  expect(screen.queryByRole("status")).toBeNull();
});
