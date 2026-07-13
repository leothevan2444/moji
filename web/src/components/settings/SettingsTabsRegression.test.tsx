// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SettingsDraftProvider } from "./SettingsDraftStore";

vi.mock("react-i18next", async (importOriginal) => ({ ...await importOriginal<typeof import("react-i18next")>(), useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock("react-router", () => ({ useOutletContext: () => ({ pushToast: vi.fn(), copyText: vi.fn() }) }));
vi.mock("./LogEventStream", () => ({ LogEventStream: () => null }));
vi.mock("urql", async (importOriginal) => ({
  ...await importOriginal<typeof import("urql")>(),
  useMutation: () => [{ fetching: false }, vi.fn()],
  useQuery: ({ query }: { query: { definitions: Array<{ name?: { value: string } }> } }) => {
    const name = query.definitions[0]?.name?.value;
    if (name === "SystemSettingsTab") return [{ data: { settings: { system: { taskDeletePolicy: "KEEP_ONLY", imageCache: { enabled: true, maxSizeMb: 1024, retentionDays: 30 } } }, settingsStatus: { imageCache: { usedBytes: 0, entryCount: 0, cacheDirectory: "", lastCleanupAt: null, lastError: null } } }, fetching: false, error: null }, vi.fn()];
    return [{ data: { logs: [{ sequence: 2, time: "2026-07-13T10:00:00Z", level: "INFO", message: "second" }, { sequence: 1, time: "2026-07-13T09:00:00Z", level: "WARNING", message: "first" }] }, fetching: false, error: null }, vi.fn()];
  }
}));

import LogsSettingsTab from "./LogsSettingsTab";
import SystemSettingsTab from "./SystemSettingsTab";

afterEach(cleanup);

describe("settings tab visual contracts", () => {
  it("renders image cache as a switch and disables dependent inputs", () => {
    const view = render(<SettingsDraftProvider><SystemSettingsTab /></SettingsDraftProvider>);
    const toggle = screen.getByLabelText("systemUi.enableCache");
    expect(toggle.closest('[role="switch"]')).toBeTruthy();
    expect((screen.getByDisplayValue("1024") as HTMLInputElement).disabled).toBe(false);
    fireEvent.click(toggle);
    expect((screen.getByDisplayValue("1024") as HTMLInputElement).disabled).toBe(true);
    expect((screen.getByDisplayValue("30") as HTMLInputElement).disabled).toBe(true);
    expect(view.container.querySelector(".image-cache-config.is-disabled")).toBeTruthy();
    expect(screen.getByText("systemUi.disabled")).toBeTruthy();
  });

  it("renders log entries as rows within one shared log viewport", () => {
    const view = render(<SettingsCardWrapper><LogsSettingsTab /></SettingsCardWrapper>);
    expect(view.container.querySelectorAll(".log-stream")).toHaveLength(1);
    expect(view.container.querySelectorAll(".log-line")).toHaveLength(2);
    expect(view.container.querySelectorAll(".log-row")).toHaveLength(0);
    expect(view.container.querySelector(".toolbar-inline--logs select")).toBeTruthy();
  });
});

function SettingsCardWrapper({ children }: { children: React.ReactNode }) {
  return <SettingsDraftProvider>{children}</SettingsDraftProvider>;
}
