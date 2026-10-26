// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { SettingsDraftProvider } from "./SettingsDraftStore";
const mocks=vi.hoisted(()=>({save:vi.fn(),refresh:vi.fn(),toast:vi.fn()}));
vi.mock("react-router",()=>({useOutletContext:()=>({pushToast:mocks.toast})}));
vi.mock("urql",()=>({
 useQuery:()=>[{data:{downloader:{stale:false,storageAvailable:true,downloadLimit:2048,uploadLimit:1024,policy:{enabled:false,ratio:0,seedHours:0,matchAll:false,deleteFiles:true,afterIngest:false}}},fetching:false},mocks.refresh],
 useMutation:()=>[{fetching:false},mocks.save]
}));
import DownloaderSettingsTab from "./DownloaderSettingsTab";
beforeEach(async()=>{vi.clearAllMocks();mocks.save.mockResolvedValue({data:{}});await i18n.changeLanguage("en");});
afterEach(cleanup);
describe("downloader settings",()=>{
 it("defaults to deleting download files and saves immediate cleanup without thresholds",async()=>{
  render(<SettingsDraftProvider><DownloaderSettingsTab/></SettingsDraftProvider>);
  fireEvent.click(screen.getByLabelText("Enable automatic cleanup (off by default)"));
  expect((screen.getByLabelText("Permanently delete downloaded files too") as HTMLInputElement).checked).toBe(true);
  fireEvent.change(screen.getByLabelText("Cleanup trigger"),{target:{value:"AFTER_INGEST"}});
  expect(screen.queryByLabelText("Share ratio threshold")).toBeNull();
  expect(screen.queryByLabelText("Seeding threshold (hours)")).toBeNull();
  fireEvent.click(screen.getByRole("button",{name:"Save cleanup policy"}));
  await waitFor(()=>expect(mocks.save).toHaveBeenCalledWith({input:{enabled:true,afterIngest:true,ratio:0,seedHours:0,matchAll:false,deleteFiles:true}}));
 });
 it("hides inactive cleanup fields and retains their draft when toggled",()=>{
  const view=render(<SettingsDraftProvider><DownloaderSettingsTab/></SettingsDraftProvider>);
  expect(screen.queryByLabelText("Cleanup trigger")).toBeNull();
  expect(screen.queryByLabelText("Permanently delete downloaded files too")).toBeNull();
  expect(screen.queryByText(/Only torrents whose linked/u)).toBeNull();
  const toggle=screen.getByLabelText("Enable automatic cleanup (off by default)");
  expect(toggle.parentElement?.querySelector(".switch__thumb")).toBeTruthy();
  fireEvent.click(toggle);
  fireEvent.change(screen.getByLabelText("Share ratio threshold"),{target:{value:"5"}});
  const files=screen.getByLabelText("Permanently delete downloaded files too") as HTMLInputElement;
  expect(files.parentElement?.querySelector(".switch__thumb")).toBeTruthy();
  const info=view.container.querySelector(".settings-info")!;
  expect(info.getAttribute("aria-label")).toMatch(/also deletes download source files/u);
  fireEvent.click(info);
  expect(files.checked).toBe(true);
  fireEvent.click(toggle);
  expect(screen.queryByLabelText("Cleanup trigger")).toBeNull();
  fireEvent.click(toggle);
  expect((screen.getByLabelText("Share ratio threshold") as HTMLInputElement).value).toBe("5");
 });
 it("preserves policy and speed drafts independently across tab switches",async()=>{
  const view=render(<SettingsDraftProvider><DownloaderSettingsTab/></SettingsDraftProvider>);
  fireEvent.click(screen.getByLabelText("Enable automatic cleanup (off by default)"));
  fireEvent.change(screen.getByLabelText("Share ratio threshold"),{target:{value:"4"}});
  fireEvent.change(screen.getByLabelText("Upload limit (KiB/s)"),{target:{value:"512"}});
  view.rerender(<SettingsDraftProvider><div/></SettingsDraftProvider>);
  view.rerender(<SettingsDraftProvider><DownloaderSettingsTab/></SettingsDraftProvider>);
  expect((screen.getByLabelText("Share ratio threshold") as HTMLInputElement).value).toBe("4");
  expect((screen.getByLabelText("Upload limit (KiB/s)") as HTMLInputElement).value).toBe("512");
  const form=screen.getByLabelText("Upload limit (KiB/s)").closest("form")!;
  fireEvent.click(within(form).getByRole("button",{name:"Save speed limit"}));
  await waitFor(()=>expect(mocks.save).toHaveBeenCalledWith({upload:true,kibPerSecond:512}));
  expect((screen.getByLabelText("Share ratio threshold") as HTMLInputElement).value).toBe("4");
 });
});
