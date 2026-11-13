// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { SettingsAutosaveProvider } from "./SettingsAutosaveProvider";
const mocks=vi.hoisted(()=>({save:vi.fn(),refresh:vi.fn(),toast:vi.fn(),preferencesError:null as Error|null}));
vi.mock("react-router",()=>({useOutletContext:()=>({pushToast:mocks.toast})}));
vi.mock("./ServiceConnectionSettings",()=>({default:()=>null}));
vi.mock("urql",()=>({
 useQuery:({query}:{query:{definitions:Array<{name?:{value:string}}>}})=>query.definitions.some(d=>d.name?.value==="DownloaderTransferPreferencesDocument")?
 [{data:{downloaderTransferPreferences:{queueingEnabled:true,maxActiveDownloads:3,maxActiveUploads:3,maxActiveTorrents:5,dontCountSlowTorrents:true,slowTorrentDlRateThreshold:2,slowTorrentUlRateThreshold:2,slowTorrentInactiveTimer:60,maxRatioEnabled:false,maxRatio:-1,maxRatioAction:0,maxSeedingTimeEnabled:false,maxSeedingTime:-1,maxInactiveSeedingTimeEnabled:false,maxInactiveSeedingTime:-1,shareLimitMode:"MatchAny"}},error:mocks.preferencesError,fetching:false},mocks.refresh]:
 [{data:{downloader:{stale:false,storageAvailable:true,downloadLimit:2048,uploadLimit:1024,policy:{enabled:false,capacityEnabled:false,minFreeGb:50,capacityOrder:"DISK_USAGE",deleteFiles:true,afterIngest:false}}},fetching:false},mocks.refresh],
 useMutation:()=>[{fetching:false},mocks.save]
}));
import DownloaderSettingsTab from "./DownloaderSettingsTab";
const flush=async()=>{await act(async()=>{await vi.advanceTimersByTimeAsync(500);});};
beforeEach(async()=>{vi.useFakeTimers();vi.clearAllMocks();mocks.preferencesError=null;mocks.save.mockResolvedValue({data:{}});await i18n.changeLanguage("en");});
afterEach(()=>{cleanup();vi.clearAllTimers();vi.useRealTimers();});
const show=()=>render(<SettingsAutosaveProvider><DownloaderSettingsTab/></SettingsAutosaveProvider>);
describe("downloader settings",()=>{
 it("enables immediate import cleanup without Moji seeding conditions",async()=>{
  show();fireEvent.click(screen.getByLabelText("Enable automatic cleanup"));
  expect((screen.getByLabelText("Cleanup trigger") as HTMLSelectElement).value).toBe("AFTER_INGEST");
  expect(screen.queryByLabelText("Share ratio threshold")).toBeNull();
  await flush();expect(mocks.save).toHaveBeenCalledWith({input:{enabled:true,afterIngest:true}});
 });
 it("selects one trigger and requires file deletion for capacity cleanup",async()=>{
  show();fireEvent.click(screen.getByLabelText("Enable automatic cleanup"));
  fireEvent.click(screen.getByLabelText("Permanently delete downloaded files too"));
  fireEvent.change(screen.getByLabelText("Cleanup trigger"),{target:{value:"CAPACITY"}});
  expect((screen.getByLabelText("Permanently delete downloaded files too") as HTMLInputElement).checked).toBe(true);
  expect((screen.getByLabelText("Permanently delete downloaded files too") as HTMLInputElement).disabled).toBe(true);
  fireEvent.change(screen.getByLabelText("Free space threshold (GB)"),{target:{value:"25.5"}});
  fireEvent.change(screen.getByLabelText("Cleanup order"),{target:{value:"RATIO"}});
  await flush();expect(mocks.save).toHaveBeenCalledWith({input:{enabled:true,capacityEnabled:true,minFreeGb:25.5,capacityOrder:"RATIO"}});
 });
 it("preserves policy and speed drafts across tab switches",async()=>{
  const view=show();fireEvent.click(screen.getByLabelText("Enable automatic cleanup"));
  fireEvent.change(screen.getByLabelText("Cleanup trigger"),{target:{value:"CAPACITY"}});
  fireEvent.change(screen.getByLabelText("Free space threshold (GB)"),{target:{value:"75"}});
  fireEvent.change(screen.getByLabelText("Upload limit (KiB/s)"),{target:{value:"512"}});
  view.rerender(<SettingsAutosaveProvider><div/></SettingsAutosaveProvider>);
  view.rerender(<SettingsAutosaveProvider><DownloaderSettingsTab/></SettingsAutosaveProvider>);
  expect((screen.getByLabelText("Free space threshold (GB)") as HTMLInputElement).value).toBe("75");
  expect((screen.getByLabelText("Upload limit (KiB/s)") as HTMLInputElement).value).toBe("512");
  await flush();expect(mocks.save).toHaveBeenCalledWith({upload:true,kibPerSecond:512});
 });

 it("switches triggers atomically and retains capacity settings",async()=>{
  show();fireEvent.click(screen.getByLabelText("Enable automatic cleanup"));await flush();mocks.save.mockClear();
  fireEvent.change(screen.getByLabelText("Cleanup trigger"),{target:{value:"CAPACITY"}});
  fireEvent.change(screen.getByLabelText("Free space threshold (GB)"),{target:{value:"80"}});await flush();
  expect(mocks.save).toHaveBeenCalledWith({input:{afterIngest:false,capacityEnabled:true,minFreeGb:80}});
  mocks.save.mockClear();fireEvent.change(screen.getByLabelText("Cleanup trigger"),{target:{value:"AFTER_INGEST"}});await flush();
  expect(mocks.save).toHaveBeenCalledWith({input:{afterIngest:true,capacityEnabled:false}});
  expect(screen.queryByLabelText("Free space threshold (GB)")).toBeNull();
  fireEvent.change(screen.getByLabelText("Cleanup trigger"),{target:{value:"CAPACITY"}});
  expect((screen.getByLabelText("Free space threshold (GB)") as HTMLInputElement).value).toBe("80");
 });
 it("patches qB seeding and queue settings without overwriting unrelated values",async()=>{
  show();fireEvent.click(screen.getByLabelText("Enable share ratio limit"));
  fireEvent.change(screen.getByLabelText("When ratio reaches"),{target:{value:"2.5"}});
  fireEvent.change(screen.getByLabelText("Maximum active downloads (-1: unlimited)"),{target:{value:"6"}});
  await flush();expect(mocks.save).toHaveBeenCalledWith({input:{maxRatioEnabled:true,maxRatio:2.5,maxActiveDownloads:6}});
 });
 it("keeps speed and cleanup available when qB preferences fail",()=>{
  mocks.preferencesError=new Error("qB offline");show();
  expect(screen.getByLabelText("Upload limit (KiB/s)")).toBeTruthy();
  expect(screen.getByLabelText("Enable automatic cleanup")).toBeTruthy();
  expect(screen.queryByLabelText("Enable share ratio limit")).toBeNull();
 });
});
