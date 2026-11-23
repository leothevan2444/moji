import { getAppTimezone, setAppTimezone } from "../../utils/appTimezone";
// @vitest-environment jsdom

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SettingsAutosaveProvider } from "./SettingsAutosaveProvider";

const { updateSettings, languageChanged } = vi.hoisted(() => ({ updateSettings: vi.fn().mockResolvedValue({ data: {} }), languageChanged: vi.fn() }));

vi.mock("react-i18next", async (importOriginal) => ({ ...await importOriginal<typeof import("react-i18next")>(), useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock("react-router", () => ({ useOutletContext: () => ({ pushToast: vi.fn(), copyText: vi.fn(), onLanguageChange: languageChanged, onTimezoneChange: setAppTimezone }) }));
vi.mock("./LogEventStream", () => ({ LogEventStream: () => null }));
vi.mock("urql", async (importOriginal) => ({
  ...await importOriginal<typeof import("urql")>(),
  useMutation: () => [{ fetching: false }, updateSettings],
  useQuery: ({ query }: { query: { definitions: Array<{ name?: { value: string } }> } }) => {
    const name = query.definitions[0]?.name?.value;
    if (name === "TaskDeletionSettings") return [{data:{settings:{system:{taskDeletePolicy:"KEEP_ONLY"}}},fetching:false,error:null},vi.fn()];
    if (name === "SettingsPolicySection") return [{ data: { settings: { policies: {
      taskProcessingFrequency: "MEDIUM", sourcingRetryIntervalHours: 6, stashBoxEndpoints: [],
      subscriptionReleasePolicy: { soloBehavior: "DOWNLOAD", groupBehavior: "REVIEW", compilationBehavior: "BLOCK", vrBehavior: "REVIEW", maxGroupPerformerCount: 3, releaseDateRange: "ALL" },
      torrentSelection: {version:7,activeGroup:"",rules:[],groups:[]}
    } }, settingsStatus: { metadataSources: { stashBoxes: [], stashBoxesLoaded: true, stashBoxesLoadError: null } } }, fetching: false, error: null }, vi.fn()];
    if (name === "LibraryDeliverySettings") return [{ data: { settings: { delivery: { keepSubtitles: true, downloads: { qbRoot: "/q/downloads", mojiRoot: "/moji/downloads" }, library: { mojiRoot: "/moji/library", stashRoot: "/stash/library" }, transfer: { action: "COPY" } }, qbittorrent: { defaultSavePath: "" } }, settingsStatus: { stashLibraries: [{ path: "/stash/library" }] } }, fetching: false, error: null }, vi.fn()];
    if (name === "SystemSettingsTab") return [{ data: { settings: { system: { taskDeletePolicy: "KEEP_ONLY", imageCache: { enabled: true, maxSizeMb: 1024, retentionDays: 30 } } }, settingsStatus: { imageCache: { usedBytes: 0, entryCount: 0, cacheDirectory: "", lastCleanupAt: null, lastError: null } } }, fetching: false, error: null }, vi.fn()];
    return [{ data: { logs: [{ sequence: 2, time: "2026-07-13T10:00:00Z", level: "INFO", message: "second" }, { sequence: 1, time: "2026-07-13T09:00:00Z", level: "WARNING", message: "first" }] }, fetching: false, error: null }, vi.fn()];
  }
}));

import LibraryDeliverySettings from "./LibraryDeliverySettings";
import LogsSettingsTab from "./LogsSettingsTab";
import SystemSettingsTab from "./SystemSettingsTab";
import TaskSubscriptionSettingsTab from "./TaskSubscriptionSettingsTab";

const flush = async () => { await act(async () => { await vi.advanceTimersByTimeAsync(500); }); };
beforeEach(()=>{ setAppTimezone("UTC");vi.useFakeTimers();updateSettings.mockReset().mockResolvedValue({data:{}});languageChanged.mockClear(); });
afterEach(()=>{cleanup();vi.clearAllTimers();vi.useRealTimers();});

describe("settings tab visual contracts", () => {
  it("saves language only with the system form and applies it after success", async () => {
    updateSettings.mockClear(); languageChanged.mockClear();
    render(<SettingsAutosaveProvider><SystemSettingsTab /></SettingsAutosaveProvider>);
    expect(screen.queryByLabelText(/taskDeletion.title/)).toBeNull();
    fireEvent.change(screen.getByRole("combobox",{name:"common.language"}),{target:{value:"en"}});
    expect(languageChanged).not.toHaveBeenCalled();
    updateSettings.mockResolvedValueOnce({error:new Error("save failed")});
    await flush();
    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(languageChanged).not.toHaveBeenCalled();
    expect((screen.getByRole("combobox",{name:"common.language"}) as HTMLSelectElement).value).toBe("en");
    fireEvent.click(screen.getByText("common.retry"));
    await flush();
    expect(languageChanged).toHaveBeenCalledWith("en");
    expect(updateSettings.mock.calls[1][0].input.system.language).toBe("en");
    expect(updateSettings.mock.calls[1][0].input).not.toHaveProperty("taskDeletePolicy");
  });

  it("uses a timezone select and applies only successful saves", async () => {
    render(<SettingsAutosaveProvider><SystemSettingsTab /></SettingsAutosaveProvider>);
    const select = screen.getByRole("combobox", { name: "systemSettings.timezone" }) as HTMLSelectElement;
    expect(select.querySelector('option[value="Asia/Tokyo"]')).toBeTruthy();
    expect(screen.queryByRole("textbox", { name: "systemSettings.timezone" })).toBeNull();
    updateSettings.mockResolvedValueOnce({error: new Error("save failed")});
    fireEvent.change(select, {target:{value:"Asia/Tokyo"}});
    await flush();
    expect(getAppTimezone()).toBe("UTC");
    expect(select.value).toBe("Asia/Tokyo");
    fireEvent.click(screen.getByText("common.retry"));
    await flush();
    expect(getAppTimezone()).toBe("Asia/Tokyo");
    expect(updateSettings).toHaveBeenLastCalledWith({input:{system:{timezone:"Asia/Tokyo"}}});
  });

  it("saves task deletion policy independently in tasks and subscriptions", async () => {
    updateSettings.mockClear();
    render(<SettingsAutosaveProvider><TaskSubscriptionSettingsTab /></SettingsAutosaveProvider>);
    const policy=screen.getByRole("combobox",{name:/taskDeletion.title/});
    fireEvent.change(policy,{target:{value:"REMOVE_TORRENT"}});
    await flush();
    expect(updateSettings).toHaveBeenCalledWith({input:{tasks:{task_delete_policy:"REMOVE_TORRENT"}}});
    expect(screen.queryByRole("combobox",{name:"common.language"})).toBeNull();
  });

  it("saves developer mode through the system settings switch", async () => {
    updateSettings.mockClear();
    render(<SettingsAutosaveProvider><SystemSettingsTab /></SettingsAutosaveProvider>);
    const toggle = screen.getByRole("switch", { name: "systemSettings.developerMode" });
    expect(toggle.getAttribute("aria-checked")).toBe("false");
    fireEvent.click(toggle);
    expect(toggle.getAttribute("aria-checked")).toBe("true");
    await flush();
    expect(updateSettings).toHaveBeenCalledWith({input:{system:{developer_mode:true}}});
  });

  it("saves the VR policy and preserves its draft across tabs", async () => {
    updateSettings.mockClear();
    const view = render(<SettingsAutosaveProvider><TaskSubscriptionSettingsTab /></SettingsAutosaveProvider>);
    const vr = screen.getByRole("combobox", { name: /taskSettings.vr/ }) as HTMLSelectElement;
    expect(vr.value).toBe("REVIEW");
    expect(Array.from(vr.options, option => option.value)).toEqual(["DOWNLOAD", "REVIEW", "BLOCK"]);
    expect(screen.queryByText("taskSettings.groupMax")).toBeNull();
    for (const key of ["policyScope", "behaviorHelp", "priorityHelp", "reviewHelp"]) {
      expect(screen.getByText(`taskSettings.${key}`)).toBeTruthy();
    }
    fireEvent.change(vr, { target: { value: "BLOCK" } });
    view.rerender(<SettingsAutosaveProvider><SystemSettingsTab /></SettingsAutosaveProvider>);
    view.rerender(<SettingsAutosaveProvider><TaskSubscriptionSettingsTab /></SettingsAutosaveProvider>);
    expect((screen.getByRole("combobox", { name: /taskSettings.vr/ }) as HTMLSelectElement).value).toBe("BLOCK");
    await flush();
    expect(updateSettings).toHaveBeenCalledWith({input:{tasks:{subscription_release_policy:{vr_behavior:"BLOCK"}}}});
  });

  it("retains a failed save as an unsaved draft across tabs", async () => {
    updateSettings.mockClear();
    updateSettings.mockResolvedValueOnce({ error: new Error("unavailable") });
    const view = render(<SettingsAutosaveProvider><TaskSubscriptionSettingsTab /></SettingsAutosaveProvider>);
    fireEvent.change(screen.getByLabelText(/resourceTracking.interval/), { target: { value: "18" } });
    await flush();
    expect(updateSettings).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("alert").textContent).toContain("settings.autosave.error");
    view.rerender(<SettingsAutosaveProvider><SystemSettingsTab /></SettingsAutosaveProvider>);
    view.rerender(<SettingsAutosaveProvider><TaskSubscriptionSettingsTab /></SettingsAutosaveProvider>);
    expect((screen.getByLabelText(/resourceTracking.interval/) as HTMLInputElement).value).toBe("18");
  });

  it("saves reusable rules and group tiers only in the resource section", async () => {
    updateSettings.mockClear();
    render(<SettingsAutosaveProvider><TaskSubscriptionSettingsTab /></SettingsAutosaveProvider>);
    fireEvent.click(screen.getByText("resourceSelection.addRule"));
    fireEvent.change(screen.getByRole("textbox",{name:"resourceSelection.expression"}),{target:{value:"user-label"}});
    fireEvent.click(screen.getByRole("button",{name:"common.close"}));
    fireEvent.click(screen.getByText("resourceSelection.addGroup"));
    fireEvent.click(screen.getByRole("switch",{name:"resourceSelection.allowFallback"}));
    await flush();
    const input=updateSettings.mock.calls.at(-1)?.[0].input;
    expect(Object.keys(input)).toEqual(["tasks"]);
    const p=input.tasks.torrent_selection;
    expect(p.rules[0]).not.toHaveProperty("action");
    expect(p.groups[0].tiers[0].rules).toEqual([]);
    expect(p.groups[0].fallback).toBe(true);
    expect(p.groups[0].tiers).toHaveLength(1);
    expect(p.groups[0]).not.toHaveProperty("requireMode");
    expect(p.groups[0].tiers[0]).not.toHaveProperty("mode");
    expect(p.groups[0].tiers[0]).not.toHaveProperty("matchAll");
    expect(p).not.toHaveProperty("original");expect(p).not.toHaveProperty("order");
  });

  it("edits torrent size in GiB while storing exact bytes",async()=>{
    updateSettings.mockClear();render(<SettingsAutosaveProvider><TaskSubscriptionSettingsTab /></SettingsAutosaveProvider>);
    fireEvent.click(screen.getByText("resourceSelection.addRule"));
    fireEvent.change(screen.getByRole("combobox",{name:"resourceSelection.scope"}),{target:{value:"SIZE"}});
    const size=screen.getByRole("textbox",{name:"resourceSelection.minimumGiB"});
    fireEvent.change(size,{target:{value:"1."}});expect((size as HTMLInputElement).value).toBe("1.");
    fireEvent.change(size,{target:{value:"1.5"}});await flush();
    expect(updateSettings.mock.calls.at(-1)?.[0].input.tasks.torrent_selection.rules[0].conditions[0]).toMatchObject({scope:"SIZE",minimum:"1610612736",maximum:"",value:""});
  });

  it("omits empty conditions from autosave while keeping rows editable", async () => {
    render(<SettingsAutosaveProvider><TaskSubscriptionSettingsTab /></SettingsAutosaveProvider>);
    fireEvent.click(screen.getByText("resourceSelection.addRule"));
    await flush();
    expect(updateSettings.mock.calls.at(-1)?.[0].input.tasks.torrent_selection.rules[0].conditions).toEqual([]);
    fireEvent.change(screen.getByRole("textbox", { name: "resourceSelection.expression" }), { target: { value: "label" } });
    await flush();
    fireEvent.click(screen.getByRole("button", { name: "resourceSelection.addCondition" }));
    fireEvent.change(screen.getAllByRole("combobox", { name: "resourceSelection.scope" })[1], { target: { value: "SIZE" } });
    await flush();
    expect(updateSettings.mock.calls.at(-1)?.[0].input.tasks.torrent_selection.rules[0].conditions).toHaveLength(1);
    expect(screen.getAllByRole("combobox", { name: "resourceSelection.scope" })).toHaveLength(2);
    fireEvent.change(screen.getByRole("textbox", { name: "resourceSelection.expression" }), { target: { value: "   " } });
    await flush();
    expect(updateSettings.mock.calls.at(-1)?.[0].input.tasks.torrent_selection.rules[0].conditions).toEqual([]);
    fireEvent.change(screen.getByRole("textbox", { name: "resourceSelection.maximumGiB" }), { target: { value: "2" } });
    await flush();
    expect(updateSettings.mock.calls.at(-1)?.[0].input.tasks.torrent_selection.rules[0].conditions).toEqual([{scope:"SIZE",value:"",minimum:"",maximum:"2147483648"}]);
  });

  it("loads and saves the resource search interval independently of metadata polling", async () => {
    updateSettings.mockClear();
    render(<SettingsAutosaveProvider><TaskSubscriptionSettingsTab /></SettingsAutosaveProvider>);
    const input = screen.getByLabelText(/resourceTracking.interval/) as HTMLInputElement;
    expect(input.value).toBe("6");
    expect(input.min).toBe("1");
    expect(input.max).toBe("168");
    fireEvent.change(input, { target: { value: "12" } });
    await flush();
    expect(updateSettings).toHaveBeenCalledWith({input:{tasks:{sourcing_retry_interval_hours:12}}});
  });
  it("offers three processing frequencies without a subscription switch", async () => {
    updateSettings.mockClear();
    render(<SettingsAutosaveProvider><TaskSubscriptionSettingsTab /></SettingsAutosaveProvider>);
    const interval = screen.getByLabelText(/taskSettings.frequency/) as HTMLSelectElement;
    expect(Array.from(interval.options, option => option.value)).toEqual(["HIGH", "MEDIUM", "LOW"]);
    expect(interval.disabled).toBe(false);
    expect(screen.queryByText("performerActivity.automaticChecks")).toBeNull();
    fireEvent.change(interval, { target: { value: "LOW" } });
    await flush();
    expect(updateSettings).toHaveBeenCalledWith({input:{tasks:{task_processing_frequency:"LOW"}}});
  });

  it("offers four roots and Copy or Move, and saves without an ingest mode", async () => {
    const view = render(<SettingsAutosaveProvider><LibraryDeliverySettings /></SettingsAutosaveProvider>);
    expect(screen.queryByText("libraryDelivery.mode")).toBeNull();
    const actionSelect = view.container.querySelector('select:has(option[value="COPY"])') as HTMLSelectElement;
    expect(Array.from(actionSelect.options, option => option.value)).toEqual(["COPY", "MOVE"]);
    for (const root of ["/q/downloads", "/moji/downloads", "/moji/library", "/stash/library"]) {
      expect(screen.getByDisplayValue(root)).toBeTruthy();
    }
    fireEvent.change(actionSelect, { target: { value: "MOVE" } });
    await flush();
    expect(updateSettings).toHaveBeenCalledWith({input:{stash:{ingest:{transfer:{action:"MOVE"}}}}});
  });

  it("defaults transcodes off and preserves the choice across tab changes before saving", async () => {
    updateSettings.mockClear();
    const view = render(<SettingsAutosaveProvider><LibraryDeliverySettings /></SettingsAutosaveProvider>);
    const toggle = screen.getByLabelText("libraryDelivery.generateTranscodes") as HTMLInputElement;
    expect(toggle.checked).toBe(false);
    fireEvent.click(toggle);
    view.rerender(<SettingsAutosaveProvider><SystemSettingsTab /></SettingsAutosaveProvider>);
    view.rerender(<SettingsAutosaveProvider><LibraryDeliverySettings /></SettingsAutosaveProvider>);
    expect((screen.getByLabelText("libraryDelivery.generateTranscodes") as HTMLInputElement).checked).toBe(true);
    await flush();
    expect(updateSettings).toHaveBeenCalledWith({input:{stash:{ingest:{generate_transcodes:true}}}});
    fireEvent.click(screen.getByLabelText("libraryDelivery.generateTranscodes"));
    await flush();
    expect(updateSettings).toHaveBeenLastCalledWith({input:{stash:{ingest:{generate_transcodes:false}}}});
  });

  it("keeps existing-cache cleanup limits editable while image writes are disabled", () => {
    const view = render(<SettingsAutosaveProvider><SystemSettingsTab /></SettingsAutosaveProvider>);
    const toggle = screen.getByLabelText("systemSettings.enableCache");
    expect(toggle.closest('[role="switch"]')).toBeTruthy();
    expect((screen.getByDisplayValue("1024") as HTMLInputElement).disabled).toBe(false);
    fireEvent.click(toggle);
    expect((screen.getByDisplayValue("1024") as HTMLInputElement).disabled).toBe(false);
    expect((screen.getByDisplayValue("30") as HTMLInputElement).disabled).toBe(false);
    expect(screen.queryByText("stashBoxCacheUi.title")).toBeNull();
    expect(view.container.querySelector(".image-cache-config.is-disabled")).toBeNull();
    expect(screen.queryByText("systemSettings.disabled")).toBeNull();
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
  return <SettingsAutosaveProvider>{children}</SettingsAutosaveProvider>;
}
