// @vitest-environment jsdom
import { gql } from "urql";
import { expect, it, vi } from "vitest";

const transport = vi.hoisted(() => ({
  sinks: [] as Array<{ next: (payload: unknown) => void }>,
  connected: null as null | (() => void)
}));
vi.mock("graphql-ws", () => ({
  createClient: (options: { on: { connected: () => void } }) => {
    transport.connected = options.on.connected;
    return { subscribe: (_input: unknown, sink: { next: (payload: unknown) => void }) => {
      transport.sinks.push(sink);
      return () => {};
    } };
  }
}));

import { client } from "./client";
import { getTaskEventGapGeneration } from "./taskEventSequence";

it("observes a burst before delivering results and only recovers for an actual transport gap", async () => {
  const results: number[] = [];
  const query = gql`subscription TransportSequenceRegression { taskEvents { sequence } }`;
  const subscription = client.subscription<{ taskEvents: { sequence: number } }>(query, {}).subscribe(result => {
    if (result.data) results.push(result.data.taskEvents.sequence);
  });
  try {
    await vi.waitFor(() => expect(transport.sinks).toHaveLength(1));
    transport.connected?.();
    const sink = transport.sinks.at(-1)!;
    const before = getTaskEventGapGeneration();
    const deliver = (sequence: number) => sink.next({ data: { taskEvents: { __typename: "TaskEvent", sequence } } });
    for (let n = 100; n <= 200; n++) deliver(n);
    await vi.waitFor(() => expect(results.at(-1)).toBe(200));
    expect(getTaskEventGapGeneration()).toBe(before);
    deliver(203);
    expect(getTaskEventGapGeneration()).toBe(before + 1);
    deliver(201); // A stale replay must not regress the normalized cache.
    await vi.waitFor(() => expect(results.at(-1)).toBe(203));
    transport.connected?.();
    deliver(1);
    await vi.waitFor(() => expect(results.at(-1)).toBe(1));
    expect(getTaskEventGapGeneration()).toBe(before + 1);
  } finally { subscription.unsubscribe(); }
});

it("coalesces each task while delivering different tasks in sequence order",async()=>{
 const results:Array<{sequence:number;total:number}>=[];
 const query=gql`subscription BatchOrderRegression {taskEvents {sequence type taskId listChanged task:summary{id version} dashboardStats{total}}}`;
 const before=transport.sinks.length;
 const subscription=client.subscription<any>(query,{}).subscribe(result=>{if(result.data)results.push({sequence:result.data.taskEvents.sequence,total:result.data.taskEvents.dashboardStats.total})});
 try{
  await vi.waitFor(()=>expect(transport.sinks.length).toBe(before+1));transport.connected?.();const sink=transport.sinks.at(-1)!;
  const send=(taskId:string,sequence:number)=>sink.next({data:{taskEvents:{__typename:"TaskEvent",streamId:"batch",type:"UPDATED",taskId,sequence,listChanged:false,task:{__typename:"TaskSummary",id:taskId,version:String(sequence)},dashboardStats:{__typename:"DashboardStats",total:sequence}}}});
  send("a",1);send("b",2);send("a",3);
  await vi.waitFor(()=>expect(results.at(-1)?.sequence).toBe(3));
  expect(results).toEqual([{sequence:2,total:2},{sequence:3,total:3}]);
 }finally{subscription.unsubscribe()}
});
