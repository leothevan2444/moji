// @vitest-environment jsdom
import {act,cleanup,fireEvent,render,screen,within} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {CombinedError} from "urql";
import i18n from "../../i18n/i18n";
import {SettingsAutosaveProvider} from "./SettingsAutosaveProvider";
import {IndexerDefinitionPolicy} from "./IndexerDefinitionPolicy";
const state=vi.hoisted(()=>({snapshot:{revision:"config-one",source:"prowlarr",schemaVersion:11,checkIntervalHours:24,autoUpdate:false},update:vi.fn(),reload:vi.fn(),refresh:vi.fn(),fallback:vi.fn(),error:null as unknown,loading:false}));
vi.mock("../../hooks/useIndexerDefinitionPolicy",()=>({useIndexerDefinitionPolicy:()=>state}));
vi.mock("urql",async original=>({...await original<typeof import("urql")>(),useMutation:()=>[{},state.fallback]}));
beforeEach(async()=>{await i18n.changeLanguage("en");vi.useFakeTimers();state.snapshot={revision:"config-one",source:"prowlarr",schemaVersion:11,checkIntervalHours:24,autoUpdate:false};state.update.mockReset().mockResolvedValue("config-two");state.reload.mockReset();state.refresh.mockClear();state.fallback.mockClear();state.error=null;state.loading=false});
afterEach(()=>{cleanup();vi.useRealTimers()});
const mount=()=>render(<SettingsAutosaveProvider><IndexerDefinitionPolicy/></SettingsAutosaveProvider>);
const flush=async()=>{await act(async()=>vi.advanceTimersByTimeAsync(501))};
it("saves only edited fields and carries the exact acknowledged revision into explicit false",async()=>{
 state.update.mockResolvedValueOnce("config-two").mockResolvedValueOnce("config-three");mount();expect(state.update).not.toHaveBeenCalled();fireEvent.click(screen.getByRole("checkbox"));await flush();expect(state.update).toHaveBeenNthCalledWith(1,{autoUpdate:true},"config-one");fireEvent.click(screen.getByRole("checkbox"));await flush();expect(state.update).toHaveBeenNthCalledWith(2,{autoUpdate:false},"config-two");expect(state.fallback).not.toHaveBeenCalled();
});
it("changes the check interval without replacing automatic apply or fixed source fields",async()=>{
 mount();fireEvent.change(screen.getByRole("spinbutton"),{target:{value:"48"}});await flush();expect(state.update).toHaveBeenCalledWith({checkIntervalHours:48},"config-one");expect((screen.getByRole("checkbox") as HTMLInputElement).checked).toBe(false);
});
it.each(["","0","169","1.5"])("keeps invalid interval %s editable without sending it",async value=>{
 mount();fireEvent.change(screen.getByRole("spinbutton"),{target:{value}});await flush();expect(state.update).not.toHaveBeenCalled();expect((screen.getByRole("spinbutton") as HTMLInputElement).value).toBe(value);expect(screen.getByRole("button",{name:i18n.t("common.retry")})).toBeTruthy();
});
it("preserves failed edits across external snapshots and reloads only after confirmation",async()=>{
 state.update.mockRejectedValue(new CombinedError({graphQLErrors:[{message:"redacted",extensions:{code:"CONFIG_REVISION_CONFLICT"}}]}));const view=mount();fireEvent.click(screen.getByRole("checkbox"));await flush();expect((screen.getByRole("checkbox") as HTMLInputElement).checked).toBe(true);
 state.snapshot={...state.snapshot,revision:"external",checkIntervalHours:72};view.rerender(<SettingsAutosaveProvider><IndexerDefinitionPolicy/></SettingsAutosaveProvider>);expect((screen.getByRole("checkbox") as HTMLInputElement).checked).toBe(true);expect((screen.getByRole("spinbutton") as HTMLInputElement).value).toBe("24");
 state.reload.mockResolvedValue(state.snapshot);fireEvent.click(screen.getByRole("button",{name:i18n.t("indexerCache.reload")}));expect(state.reload).not.toHaveBeenCalled();const dialog=screen.getByRole("dialog");await act(async()=>fireEvent.click(within(dialog).getByRole("button",{name:i18n.t("indexerCache.reload")})));expect((screen.getByRole("checkbox") as HTMLInputElement).checked).toBe(false);expect((screen.getByRole("spinbutton") as HTMLInputElement).value).toBe("72");expect(state.update).toHaveBeenCalledTimes(1);
});
