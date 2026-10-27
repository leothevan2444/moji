// @vitest-environment jsdom
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
const mocks=vi.hoisted(()=>({state:{} as any,refresh:vi.fn()}));
vi.mock("urql",()=>({useQuery:()=>[mocks.state,mocks.refresh]}));
import { useTaskDetail } from "./useTaskDetail";
import { publishTaskBatch, recordTaskEvent, type TaskRealtimeEvent } from "../graphql/taskRealtime";

const full=(revision=1)=>({id:"detail",revision,version:String(revision),progress:.2,stageErrorMessage:"full diagnostic message",files:[{index:7,path:"movie.mp4"}]});
const update=(revision:number):TaskRealtimeEvent=>({type:"UPDATED",sequence:11,streamId:"detail-test",taskId:"detail",listChanged:false,task:{id:"detail",revision,version:String(revision)+"a",progress:.7,stageErrorMessage:"summary"}});
beforeEach(()=>{mocks.refresh.mockClear();recordTaskEvent({type:"READY",sequence:10,streamId:"detail-test",taskId:"",listChanged:false});mocks.state={fetching:false,data:{taskDetailSnapshot:{sequence:10,streamId:"detail-test",task:full()}}}});
afterEach(cleanup);
it("patches progress without requesting files again or clearing them",()=>{
 const {result}=renderHook(()=>useTaskDetail("detail"));
 act(()=>{const event=update(1);recordTaskEvent(event);publishTaskBatch([event])});
 expect(result.current[0].data?.task?.progress).toBe(.7);
 expect(result.current[0].data?.task?.files).toEqual([{index:7,path:"movie.mp4"}]);
 expect(result.current[0].data?.task?.stageErrorMessage).toBe("full diagnostic message");
 expect(mocks.refresh).not.toHaveBeenCalled();
});
it("waits for an in-flight detail before refetching a newer durable revision once",()=>{
 mocks.state={fetching:true};const {result,rerender}=renderHook(()=>useTaskDetail("detail"));
 act(()=>{const event=update(2);recordTaskEvent(event);publishTaskBatch([event])});
 expect(mocks.refresh).not.toHaveBeenCalled();
 mocks.state={fetching:false,data:{taskDetailSnapshot:{sequence:10,streamId:"detail-test",task:full()}}};rerender();
 expect(result.current[0].data?.task).toBeNull();expect(mocks.refresh).toHaveBeenCalledTimes(1);
 mocks.state={fetching:false,data:{taskDetailSnapshot:{sequence:11,streamId:"detail-test",task:full(2)}}};rerender();
 expect(result.current[0].data?.task?.revision).toBe(2);expect(mocks.refresh).toHaveBeenCalledTimes(1);
});
it("removes a deleted detail immediately without refetching it",()=>{
 const {result}=renderHook(()=>useTaskDetail("detail"));
 act(()=>{const event={...update(1),type:"DELETED",task:null};recordTaskEvent(event);publishTaskBatch([event])});
 expect(result.current[0].data?.task).toBeNull();expect(mocks.refresh).not.toHaveBeenCalled();
});
