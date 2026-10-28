// @vitest-environment jsdom
import { act,cleanup,fireEvent,render,screen } from "@testing-library/react";
import { afterEach,expect,it,vi } from "vitest";
import { SettingsAutosaveProvider } from "./SettingsAutosaveProvider";
import { useSettingsAutosave } from "./useSettingsAutosave";
const save=vi.hoisted(()=>vi.fn().mockResolvedValue({data:{}}));
vi.mock("urql",()=>({useMutation:()=>[{},save]}));
vi.mock("react-i18next",()=>({useTranslation:()=>({t:(key:string)=>key})}));
function Editor(){const a=useSettingsAutosave("system",{system:{language:"auto",developer_mode:false}},v=>v);return <input aria-label="language" value={a.value.system.language} onChange={e=>a.setValue(v=>({...v,system:{...v.system,language:e.target.value}}))}/>;}
afterEach(()=>{cleanup();vi.clearAllTimers();vi.useRealTimers();save.mockClear();});
it("initializes from the query, preserves pending state through navigation, and saves after unmount",async()=>{
 vi.useFakeTimers();const view=render(<SettingsAutosaveProvider><Editor/></SettingsAutosaveProvider>);
 expect(save).not.toHaveBeenCalled();fireEvent.change(screen.getByLabelText("language"),{target:{value:"en"}});
 view.rerender(<SettingsAutosaveProvider><div/></SettingsAutosaveProvider>);view.rerender(<SettingsAutosaveProvider><Editor/></SettingsAutosaveProvider>);
 expect((screen.getByLabelText("language") as HTMLInputElement).value).toBe("en");view.rerender(<SettingsAutosaveProvider><div/></SettingsAutosaveProvider>);
 await act(async()=>{await vi.advanceTimersByTimeAsync(500)});
 expect(save).toHaveBeenCalledExactlyOnceWith({input:{system:{language:"en"}}});
});
