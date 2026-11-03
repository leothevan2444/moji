// @vitest-environment jsdom
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { useTorrentSelectionPreview } from "./useTorrentSelectionPreview";
const state=vi.hoisted(()=>({mutation:vi.fn()}));
vi.mock("urql",()=>({useMutation:()=>[{fetching:false},state.mutation]}));
beforeEach(()=>{state.mutation.mockReset();});afterEach(cleanup);
it("mount and rerender do not inspect files; an explicitly empty result stays empty",async()=>{
 const hook=renderHook(()=>useTorrentSelectionPreview());hook.rerender();expect(state.mutation).not.toHaveBeenCalled();state.mutation.mockResolvedValue({data:{previewTorrentSelection:{searchId:"a",candidates:[],total:0}}});
 await act(async()=>{expect(await hook.result.current.preview({searchId:"a",applyFileRules:true})).toBe(true)});expect(hook.result.current.value?.candidates).toEqual([]);
 act(()=>hook.result.current.clear());expect(hook.result.current.value).toBeNull();expect(state.mutation).toHaveBeenCalledTimes(1);
});
it("clearing a run rejects old preview acknowledgement and blocks duplicate pending inspection",async()=>{
 let finish!:(value:{data:{previewTorrentSelection:{searchId:string}}})=>void;state.mutation.mockImplementation(()=>new Promise(resolve=>{finish=resolve}));
 const hook=renderHook(()=>useTorrentSelectionPreview());let pending!:Promise<boolean>;
 act(()=>{pending=hook.result.current.preview({searchId:"a",applyFileRules:true})});expect(await hook.result.current.preview({searchId:"a"})).toBe(false);act(()=>hook.result.current.clear());
 await act(async()=>{finish({data:{previewTorrentSelection:{searchId:"a"}}});expect(await pending).toBe(false)});expect(hook.result.current.value).toBeNull();expect(hook.result.current.error).toBeNull();
});
it("a failed later preview retains the last confirmed preview",async()=>{
 state.mutation.mockResolvedValueOnce({data:{previewTorrentSelection:{searchId:"a",total:1}}}).mockResolvedValueOnce({error:new Error("failed")});const hook=renderHook(()=>useTorrentSelectionPreview());
 await act(async()=>{await hook.result.current.preview({searchId:"a"})});await act(async()=>{expect(await hook.result.current.preview({searchId:"a"})).toBe(false)});expect(hook.result.current.value?.total).toBe(1);expect(hook.result.current.error).toBeInstanceOf(Error);
});
