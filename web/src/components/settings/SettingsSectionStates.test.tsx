// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { SettingsAutosaveProvider } from "./SettingsAutosaveProvider";

const { state } = vi.hoisted(()=>({state:{fetching:true,error:undefined as Error|undefined}}));
vi.mock("react-i18next",async(importOriginal)=>({...await importOriginal<typeof import("react-i18next")>(),useTranslation:()=>({t:(key:string)=>key})}));
vi.mock("react-router",()=>({useOutletContext:()=>({pushToast:vi.fn()})}));
vi.mock("../../hooks/useServiceStatusEvents",()=>({useServiceStatusEvents:()=>{}}));
vi.mock("urql",async(importOriginal)=>({...await importOriginal<typeof import("urql")>(),useQuery:()=>[state,vi.fn()],useMutation:()=>[{fetching:false},vi.fn()]}));
import ServiceConnectionSettings from "./ServiceConnectionSettings";
import SettingsPolicySection from "./SettingsPolicySection";

beforeEach(()=>{state.fetching=true;state.error=undefined;});
afterEach(cleanup);
for(const service of ["stash","jackett","qbittorrent"] as const) {
 for(const failed of [false,true])it(`${service} ${failed?"failure":"loading"} uses its actual service title`,()=>{
  state.fetching=!failed;state.error=failed?new Error("offline"):undefined;
  render(<SettingsAutosaveProvider><ServiceConnectionSettings service={service}/></SettingsAutosaveProvider>);
  expect(screen.getByRole("heading",{name:service==="stash"?"Stash":service==="jackett"?"Jackett":"qBittorrent"})).toBeTruthy();
  expect(screen.queryByText("settings.tabs.connections")).toBeNull();
 });
}
for(const [area,title]of [["tasks","taskSettings.title"],["resources","resourceRules.title"],["metadata","metadataSources.title"]] as const) {
 for(const failed of [false,true])it(`${area} ${failed?"failure":"loading"} uses its current section title`,()=>{
  state.fetching=!failed;state.error=failed?new Error("offline"):undefined;
  render(<SettingsPolicySection area={area}/>);
  expect(screen.getByRole("heading",{name:title})).toBeTruthy();
  expect(screen.queryByText("settings.tabs.automation")).toBeNull();
 });
}
