// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { SettingsAutosaveProvider } from "./SettingsAutosaveProvider";
const mocks=vi.hoisted(()=>({save:vi.fn(),refresh:vi.fn(),toast:vi.fn()}));
vi.mock("react-router",()=>({useOutletContext:()=>({pushToast:mocks.toast})}));
vi.mock("./ServiceConnectionSettings",()=>({default:()=>null}));
vi.mock("urql",()=>({
 useQuery:()=>[{data:{downloader:{stale:false,storageAvailable:true,downloadLimit:2048,uploadLimit:1024,policy:{enabled:false,ratio:0,seedHours:0,matchAll:false,deleteFiles:true,afterIngest:false}}},fetching:false},mocks.refresh],
 useMutation:()=>[{fetching:false},mocks.save]
}));
import DownloaderSettingsTab from "./DownloaderSettingsTab";
const flush=async()=>{await act(async()=>{await vi.advanceTimersByTimeAsync(500);});};
beforeEach(async()=>{vi.useFakeTimers();vi.clearAllMocks();mocks.save.mockResolvedValue({data:{}});await i18n.changeLanguage("en");});
afterEach(()=>{cleanup();vi.clearAllTimers();vi.useRealTimers();});
describe("downloader settings",()=>{
 it("defaults to deleting download files and saves immediate cleanup without thresholds",async()=>{
  render(<SettingsAutosaveProvider><DownloaderSettingsTab/></SettingsAutosaveProvider>);
  fireEvent.click(screen.getByLabelText("Enable automatic cleanup"));
  expect((screen.getByLabelText("Permanently delete downloaded files too") as HTMLInputElement).checked).toBe(true);
  fireEvent.change(screen.getByLabelText("Cleanup trigger"),{target:{value:"AFTER_INGEST"}});
  expect(screen.queryByLabelText("Share ratio threshold")).toBeNull();
  expect(screen.queryByLabelText("Seeding threshold (hours)")).toBeNull();
  await flush();
  expect(mocks.save).toHaveBeenCalledWith({input:{enabled:true,afterIngest:true}});
 });
 it("hides inactive cleanup fields and retains their draft when toggled",()=>{
  const view=render(<SettingsAutosaveProvider><DownloaderSettingsTab/></SettingsAutosaveProvider>);
  expect(screen.queryByLabelText("Cleanup trigger")).toBeNull();
  expect(screen.queryByLabelText("Permanently delete downloaded files too")).toBeNull();
  expect(screen.getByText(/Only torrents whose linked/u)).toBeTruthy();
  const toggle=screen.getByLabelText("Enable automatic cleanup");
  expect(toggle.parentElement?.querySelector(".switch__thumb")).toBeTruthy();
  fireEvent.click(toggle);
  fireEvent.change(screen.getByLabelText("Share ratio threshold"),{target:{value:"5"}});
  const files=screen.getByLabelText("Permanently delete downloaded files too") as HTMLInputElement;
  expect(files.parentElement?.querySelector(".switch__thumb")).toBeTruthy();
  const info=view.container.querySelectorAll(".settings-info").item(1)!;
  expect(info.getAttribute("aria-label")).toMatch(/download source files/u);
  fireEvent.click(info);
  expect(files.checked).toBe(true);
  fireEvent.click(toggle);
  expect(screen.queryByLabelText("Cleanup trigger")).toBeNull();
  fireEvent.click(toggle);
  expect((screen.getByLabelText("Share ratio threshold") as HTMLInputElement).value).toBe("5");
 });
 it("preserves policy and speed drafts independently across tab switches",async()=>{
  const view=render(<SettingsAutosaveProvider><DownloaderSettingsTab/></SettingsAutosaveProvider>);
  fireEvent.click(screen.getByLabelText("Enable automatic cleanup"));
  fireEvent.change(screen.getByLabelText("Share ratio threshold"),{target:{value:"4"}});
  fireEvent.change(screen.getByLabelText("Upload limit (KiB/s)"),{target:{value:"512"}});
  view.rerender(<SettingsAutosaveProvider><div/></SettingsAutosaveProvider>);
  view.rerender(<SettingsAutosaveProvider><DownloaderSettingsTab/></SettingsAutosaveProvider>);
  expect((screen.getByLabelText("Share ratio threshold") as HTMLInputElement).value).toBe("4");
  expect((screen.getByLabelText("Upload limit (KiB/s)") as HTMLInputElement).value).toBe("512");
  await flush();
  expect(mocks.save).toHaveBeenCalledWith({upload:true,kibPerSecond:512});
  expect((screen.getByLabelText("Share ratio threshold") as HTMLInputElement).value).toBe("4");
 });
});
