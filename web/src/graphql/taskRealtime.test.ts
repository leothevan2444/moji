import { beforeEach, expect, it } from "vitest";
import { latestTaskEvent, protectTaskSnapshot, recordTaskEvent, taskStreamCursor } from "./taskRealtime";

beforeEach(() => recordTaskEvent({ type:"READY",streamId:"test",sequence:10,taskId:"",listChanged:false }));
it("replays newer task summaries over a delayed HTTP snapshot", () => {
 recordTaskEvent({type:"UPDATED",streamId:"test",sequence:11,taskId:"one",listChanged:false,task:{__typename:"TaskSummary",id:"one",version:"002",progress:.7}});
 const result=protectTaskSnapshot({taskPage:{sequence:10,items:[{__typename:"TaskSummary",id:"one",version:"001",progress:.2},{__typename:"TaskSummary",id:"two",version:"003",progress:.3}]}});
 expect(result.taskPage.items.map((t:any)=>t.progress)).toEqual([.7,.3]);
 // Full details are intentionally separate, so summary defaults cannot clear files.
 expect(protectTaskSnapshot({__typename:"Task",id:"one",version:"001",files:["main.mp4"]}).files).toEqual(["main.mp4"]);
});
it("keeps a newer snapshot and bounds the transport journal", () => {
 for(let n=11;n<650;n++)recordTaskEvent({type:"UPDATED",streamId:"test",sequence:n,taskId:String(n),listChanged:n===12,task:{id:String(n),version:"001"}});
 expect(latestTaskEvent("11")).toBeUndefined();expect(taskStreamCursor().membershipSequence).toBe(12);
 expect(protectTaskSnapshot({__typename:"TaskSummary",id:"649",version:"002",progress:1}).progress).toBe(1);
});
it("discards the old stream journal at a ready/reconnect boundary", () => {
 recordTaskEvent({type:"UPDATED",streamId:"test",sequence:11,taskId:"one",listChanged:true,task:{id:"one",version:"999"}});
 recordTaskEvent({type:"READY",streamId:"new",sequence:0,taskId:"",listChanged:false});
 expect(latestTaskEvent("one")).toBeUndefined();expect(taskStreamCursor().streamId).toBe("new");
});
it("preserves newer counts and deletions when an older page finishes loading",()=>{
 recordTaskEvent({type:"DELETED",streamId:"test",sequence:11,taskId:"one",listChanged:true,dashboardStats:{__typename:"DashboardStats",total:1,completed:0,attention:0,waiting:0}});
 const result=protectTaskSnapshot({__typename:"TaskPage",streamId:"test",sequence:10,stats:{total:2},items:[{__typename:"TaskSummary",id:"one",version:"001"},{__typename:"TaskSummary",id:"two",version:"001"}]});
 expect(result.stats.total).toBe(1);expect(result.items.map((t:any)=>t.id)).toEqual(["two"]);
});
