import { afterEach, describe, expect, it, vi } from "vitest";
import { SettingsAutosave } from "./settingsAutosave";
const advance = async (ms=500) => { await vi.advanceTimersByTimeAsync(ms); };
afterEach(()=>vi.useRealTimers());
describe("settings autosave queue",()=>{
 it("merges rapid nested edits after 500ms and sends only changes, including empty lists and false",async()=>{
  vi.useFakeTimers();const save=vi.fn().mockResolvedValue(undefined);
  const entry=new SettingsAutosave({system:{language:"auto",enabled:true},sites:["a"]},v=>v,save);
  entry.edit(v=>({...v,system:{...v.system,language:"en"}}));await advance(400);expect(save).not.toHaveBeenCalled();
  entry.edit(v=>({...v,system:{...v.system,enabled:false},sites:[]}));await advance(499);expect(save).not.toHaveBeenCalled();await advance(1);
  expect(save).toHaveBeenCalledExactlyOnceWith({system:{language:"en",enabled:false},sites:[]});expect(entry.snapshot.pending).toEqual({});expect(entry.snapshot.state).toBe("saved");
 });
 it("keeps edits made during a slow request, serializes requests and compensates reverted values",async()=>{
  vi.useFakeTimers();let finish!:()=>void;const save=vi.fn().mockImplementationOnce(()=>new Promise<void>(resolve=>{finish=resolve})).mockResolvedValue(undefined);
  const entry=new SettingsAutosave({a:1,b:1},v=>v,save);
  entry.edit({a:2,b:1});await advance();entry.edit({a:1,b:3});await advance();expect(save).toHaveBeenCalledTimes(1);
  finish();await advance(0);expect(save).toHaveBeenCalledTimes(2);expect(save.mock.calls[1][0]).toEqual({a:1,b:3});expect(entry.snapshot.value).toEqual({a:1,b:3});expect(entry.snapshot.pending).toEqual({});
 });
 it("retains failed pending fields, ignores stale query snapshots and supports retry",async()=>{
  vi.useFakeTimers();const save=vi.fn().mockRejectedValueOnce(new Error("offline")).mockResolvedValue(undefined);
  const entry=new SettingsAutosave({a:1},v=>v,save);entry.edit({a:2});entry.initialize({a:9});await advance();
  expect(entry.snapshot.state).toBe("error");expect(entry.snapshot.value.a).toBe(2);expect(entry.snapshot.pending).toEqual({a:2});await advance(5000);expect(save).toHaveBeenCalledTimes(1);
  entry.retry();await advance(0);expect(save).toHaveBeenCalledTimes(2);expect(entry.snapshot.state).toBe("saved");expect(entry.snapshot.pending).toEqual({});
 });
 it("does not save initialization or edits reverted within the debounce window",async()=>{
  vi.useFakeTimers();const save=vi.fn();const entry=new SettingsAutosave({a:1},v=>v,save);entry.initialize({a:2});await advance();expect(save).not.toHaveBeenCalled();entry.edit({a:3});entry.edit({a:2});await advance();expect(save).not.toHaveBeenCalled();expect(entry.snapshot.state).toBe("idle");
 });
 it("continues queued work when UI listeners detach for tab navigation",async()=>{
  vi.useFakeTimers();const save=vi.fn().mockResolvedValue(undefined);const entry=new SettingsAutosave({a:1},v=>v,save);const listener=vi.fn();const unsubscribe=entry.subscribe(listener);entry.edit({a:2});unsubscribe();listener.mockClear();await advance();expect(save).toHaveBeenCalledWith({a:2});expect(listener).not.toHaveBeenCalled();
 });
});
