// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SettingsDraftProvider } from "./SettingsDraftStore";

const { updateSettings } = vi.hoisted(() => ({ updateSettings: vi.fn().mockResolvedValue({ data: {} }) }));

vi.mock("react-i18next", async (importOriginal) => ({ ...await importOriginal<typeof import("react-i18next")>(), useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock("react-router", () => ({ useOutletContext: () => ({ pushToast: vi.fn(), copyText: vi.fn() }) }));
vi.mock("./LogEventStream", () => ({ LogEventStream: () => null }));
vi.mock("urql", async (importOriginal) => ({
  ...await importOriginal<typeof import("urql")>(),
  useMutation: () => [{ fetching: false }, updateSettings],
  useQuery: ({ query }: { query: { definitions: Array<{ name?: { value: string } }> } }) => {
    const name = query.definitions[0]?.name?.value;
    if (name === "AutomationSettingsTab") return [{ data: { settings: { automation: {
      taskProcessingEnabled: true, taskProgressSyncIntervalSeconds: 60, subscriptionPollIntervalHours: 48, sourcingRetryIntervalHours: 6, stashBoxEndpoints: [],
      subscriptionReleasePolicy: { soloBehavior: "DOWNLOAD", groupBehavior: "REVIEW", compilationBehavior: "BLOCK", vrBehavior: "REVIEW", maxGroupPerformerCount: 3, releaseDateRange: "ALL" },
      torrentSelection: { enabled: true, inspectionCandidateLimit: 5, fastRules: [], torrentRules: [] }
    } }, settingsStatus: { automation: { taskProgressSyncEnabled: true, subscriptionPollEnabled: true }, stashBox: { stashBoxes: [], stashBoxesLoaded: true, stashBoxesLoadError: null } } }, fetching: false, error: null }, vi.fn()];
    if (name === "IngestSettingsTab") return [{ data: { settings: { ingest: { keepSubtitles: true, downloads: { qbRoot: "/q/downloads", mojiRoot: "/moji/downloads" }, library: { mojiRoot: "/moji/library", stashRoot: "/stash/library" }, transfer: { action: "COPY" } }, qbittorrent: { defaultSavePath: "" } }, settingsStatus: { stashLibraries: [{ path: "/stash/library" }] } }, fetching: false, error: null }, vi.fn()];
    if (name === "SystemSettingsTab") return [{ data: { settings: { system: { taskDeletePolicy: "KEEP_ONLY", imageCache: { enabled: true, maxSizeMb: 1024, retentionDays: 30 } } }, settingsStatus: { imageCache: { usedBytes: 0, entryCount: 0, cacheDirectory: "", lastCleanupAt: null, lastError: null } } }, fetching: false, error: null }, vi.fn()];
    return [{ data: { logs: [{ sequence: 2, time: "2026-07-13T10:00:00Z", level: "INFO", message: "second" }, { sequence: 1, time: "2026-07-13T09:00:00Z", level: "WARNING", message: "first" }] }, fetching: false, error: null }, vi.fn()];
  }
}));

import IngestSettingsTab from "./IngestSettingsTab";
import LogsSettingsTab from "./LogsSettingsTab";
import SystemSettingsTab from "./SystemSettingsTab";
import AutomationSettingsTab from "./AutomationSettingsTab";

afterEach(cleanup);

describe("settings tab visual contracts", () => {
  it("saves developer mode through the system settings switch", async () => {
    updateSettings.mockClear();
    render(<SettingsDraftProvider><SystemSettingsTab /></SettingsDraftProvider>);
    const toggle = screen.getByRole("switch", { name: "systemUi.developerMode" });
    expect(toggle.getAttribute("aria-checked")).toBe("false");
    fireEvent.click(toggle);
    expect(toggle.getAttribute("aria-checked")).toBe("true");
    fireEvent.click(screen.getByText("systemUi.save"));
    await waitFor(() => expect(updateSettings).toHaveBeenCalledWith({ input: expect.objectContaining({ developerMode: true }) }));
  });

  it("saves the VR policy and preserves its draft across tabs", async () => {
    updateSettings.mockClear();
    const view = render(<SettingsDraftProvider><AutomationSettingsTab /></SettingsDraftProvider>);
    const vr = screen.getByRole("combobox", { name: /automationUi.vr/ }) as HTMLSelectElement;
    expect(vr.value).toBe("REVIEW");
    expect(Array.from(vr.options, option => option.value)).toEqual(["DOWNLOAD", "REVIEW", "BLOCK"]);
    expect(screen.queryByText("automationUi.groupMax")).toBeNull();
    for (const key of ["policyScope", "behaviorHelp", "priorityHelp", "reviewHelp"]) {
      expect(screen.getByText(`automationUi.${key}`)).toBeTruthy();
    }
    fireEvent.change(vr, { target: { value: "BLOCK" } });
    view.rerender(<SettingsDraftProvider><SystemSettingsTab /></SettingsDraftProvider>);
    view.rerender(<SettingsDraftProvider><AutomationSettingsTab /></SettingsDraftProvider>);
    expect((screen.getByRole("combobox", { name: /automationUi.vr/ }) as HTMLSelectElement).value).toBe("BLOCK");
    fireEvent.click(screen.getByText("automationUi.save"));
    await waitFor(() => expect(updateSettings).toHaveBeenCalledWith({ input: expect.objectContaining({
      subscriptionReleasePolicy: expect.objectContaining({ vrBehavior: "BLOCK", maxGroupPerformerCount: 3 })
    }) }));
  });

  it("loads and saves the resource search interval independently of metadata polling", async () => {
    updateSettings.mockClear();
    render(<SettingsDraftProvider><AutomationSettingsTab /></SettingsDraftProvider>);
    const input = screen.getByLabelText(/resourceTracking.interval/) as HTMLInputElement;
    expect(input.value).toBe("6");
    expect(input.min).toBe("1");
    expect(input.max).toBe("168");
    fireEvent.change(input, { target: { value: "12" } });
    fireEvent.click(screen.getByText("automationUi.save"));
    await waitFor(() => expect(updateSettings).toHaveBeenCalledWith({ input: expect.objectContaining({ sourcingRetryIntervalHours: 12, subscriptionPollIntervalHours: 48 }) }));
  });
  it("offers preset heartbeat intervals and saves the processing switch independently", async () => {
    updateSettings.mockClear();
    render(<SettingsDraftProvider><AutomationSettingsTab /></SettingsDraftProvider>);
    const toggle = screen.getByLabelText("automationUi.processingEnabled") as HTMLInputElement;
    const interval = screen.getByLabelText(/automationUi.syncInterval/) as HTMLSelectElement;
    expect(toggle.closest('[role="switch"]')).toBeTruthy();
    expect(toggle.checked).toBe(true);
    expect(Array.from(interval.options, option => option.value)).toEqual(["30", "60", "120", "300"]);
    fireEvent.change(interval, { target: { value: "120" } });
    fireEvent.click(toggle);
    expect(interval.disabled).toBe(true);
    expect(interval.value).toBe("120");
    expect(screen.getByText("automationUi.processingOffHelp")).toBeTruthy();
    fireEvent.click(screen.getByText("automationUi.save"));
    await waitFor(() => expect(updateSettings).toHaveBeenCalledWith({ input: expect.objectContaining({ taskProcessingEnabled: false, taskProgressSyncIntervalSeconds: 120 }) }));
  });

  it("offers four roots and Copy or Move, and saves without an ingest mode", async () => {
    const view = render(<SettingsDraftProvider><IngestSettingsTab /></SettingsDraftProvider>);
    expect(screen.queryByText("settings.ingest.mode")).toBeNull();
    const actionSelect = view.container.querySelector('select:has(option[value="COPY"])') as HTMLSelectElement;
    expect(Array.from(actionSelect.options, option => option.value)).toEqual(["COPY", "MOVE"]);
    for (const root of ["/q/downloads", "/moji/downloads", "/moji/library", "/stash/library"]) {
      expect(screen.getByDisplayValue(root)).toBeTruthy();
    }
    fireEvent.change(actionSelect, { target: { value: "MOVE" } });
    fireEvent.click(screen.getByText("settings.ingest.save"));
    await waitFor(() => expect(updateSettings).toHaveBeenCalledWith({ input: {
      keepSubtitles: true,
      generateTranscodes: false,
      downloads: { qbRoot: "/q/downloads", mojiRoot: "/moji/downloads" },
      library: { mojiRoot: "/moji/library", stashRoot: "/stash/library" },
      transfer: { action: "MOVE" }
    } }));
  });

  it("defaults transcodes off and preserves the choice across tab changes before saving", async () => {
    updateSettings.mockClear();
    const view = render(<SettingsDraftProvider><IngestSettingsTab /></SettingsDraftProvider>);
    const toggle = screen.getByLabelText("settings.ingest.generateTranscodes") as HTMLInputElement;
    expect(toggle.checked).toBe(false);
    fireEvent.click(toggle);
    view.rerender(<SettingsDraftProvider><SystemSettingsTab /></SettingsDraftProvider>);
    view.rerender(<SettingsDraftProvider><IngestSettingsTab /></SettingsDraftProvider>);
    expect((screen.getByLabelText("settings.ingest.generateTranscodes") as HTMLInputElement).checked).toBe(true);
    fireEvent.click(screen.getByText("settings.ingest.save"));
    await waitFor(() => expect(updateSettings).toHaveBeenCalledWith({ input: expect.objectContaining({ generateTranscodes: true, keepSubtitles: true }) }));
    fireEvent.click(screen.getByLabelText("settings.ingest.generateTranscodes"));
    fireEvent.click(screen.getByText("settings.ingest.save"));
    await waitFor(() => expect(updateSettings).toHaveBeenLastCalledWith({ input: expect.objectContaining({ generateTranscodes: false }) }));
  });

  it("renders image cache as a switch and disables dependent inputs", () => {
    const view = render(<SettingsDraftProvider><SystemSettingsTab /></SettingsDraftProvider>);
    const toggle = screen.getByLabelText("systemUi.enableCache");
    expect(toggle.closest('[role="switch"]')).toBeTruthy();
    expect((screen.getByDisplayValue("1024") as HTMLInputElement).disabled).toBe(false);
    fireEvent.click(toggle);
    expect((screen.getByDisplayValue("1024") as HTMLInputElement).disabled).toBe(true);
    expect((screen.getByDisplayValue("30") as HTMLInputElement).disabled).toBe(true);
    const stashBoxTtl = screen.getByDisplayValue("24") as HTMLInputElement;
    expect(stashBoxTtl.min).toBe("12");
    expect(stashBoxTtl.max).toBe("360");
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
