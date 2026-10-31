// @vitest-environment jsdom
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { LibraryFilter, SceneSourceFilter } from "../graphql/generated/graphql";
const mocks = vi.hoisted(()=>({event:null as {performerId:string;sequence:number}|null, refresh:vi.fn()}));
vi.mock("urql",()=>({useQuery:()=>[{data:undefined,fetching:false},mocks.refresh]}));
vi.mock("./usePerformerSubscriptionEvents",()=>({usePerformerSubscriptionEvents:()=>({event:mocks.event})}));
import { usePerformerScenes } from "./usePerformerScenes";
beforeEach(()=>{vi.useFakeTimers();mocks.event=null;mocks.refresh.mockClear();});
afterEach(()=>{cleanup();vi.useRealTimers();});
it("coalesces matching actor events and cancels pending reads on unmount",()=>{
 const view=renderHook(()=>usePerformerScenes({enabled:true,performerId:"p1",search:null,source:SceneSourceFilter.All,library:LibraryFilter.All,page:2,pageSize:24}));
 mocks.event={performerId:"p2",sequence:1};view.rerender();act(()=>vi.advanceTimersByTime(300));
 expect(mocks.refresh).not.toHaveBeenCalled();
 for(let sequence=2;sequence<=4;sequence++) {mocks.event={performerId:"p1",sequence};view.rerender();act(()=>vi.advanceTimersByTime(50));}
 act(()=>vi.advanceTimersByTime(200));
 expect(mocks.refresh).toHaveBeenCalledTimes(1);
 expect(mocks.refresh).toHaveBeenCalledWith({requestPolicy:"network-only"});
 mocks.event={performerId:"p1",sequence:5};view.rerender();view.unmount();act(()=>vi.advanceTimersByTime(300));
 expect(mocks.refresh).toHaveBeenCalledTimes(1);
});
