// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import type { DashboardTask } from "../../utils";
import { TaskCard } from "./TaskCard";

const task = {
 id:"time",source:"MANUAL",code:"ABCD-123",stage:"DOWNLOADING",stageStatus:"RUNNING",progress:.5,
 createdAt:"2026-10-18T01:00:00Z",updatedAt:"2026-10-19T02:00:00Z",lastProgressAt:"2026-10-18T02:00:00Z"
} as DashboardTask;
beforeEach(async()=>{await i18n.changeLanguage("en");});
afterEach(cleanup);

describe("task card progress time",()=>{
 it("shows a normal resource wait, its next check and an immediate check action",()=>{
  const onRetry=vi.fn();
  const onResolve=vi.fn();
  const waiting={...task,stage:"SOURCING",stageStatus:"PENDING",progress:0,sourcingLastCheckedAt:"2026-10-20T00:00:00Z",sourcingNextCheckAt:"2026-10-20T06:00:00Z"} as DashboardTask;
  const {container}=render(<TaskCard task={waiting} onOpen={vi.fn()} onRetry={onRetry} onResolve={onResolve}/>);
  expect(container.querySelector(".task-status-error")).toBeNull();
  expect(container.querySelector(".task-card__tracking")?.textContent).toContain("Next check:");
  fireEvent.click(screen.getByRole("button",{name:"Check resources now: ABCD-123"}));
  expect(onRetry).toHaveBeenCalledWith(task.id);
  fireEvent.click(screen.getByRole("button",{name:"Choose or add a resource: ABCD-123"}));
  expect(onResolve).toHaveBeenCalledWith(task.id);
 });
 it("keeps the visible time stable during polling and advances it on actual progress",()=>{
  const {container,rerender}=render(<TaskCard task={task} onOpen={vi.fn()} />);
  const time=()=>container.querySelector("time")!;
  const text=time().textContent;
  expect(time().dateTime).toBe(task.lastProgressAt);
  rerender(<TaskCard task={{...task,updatedAt:"2026-10-19T03:00:00Z"}} onOpen={vi.fn()} />);
  expect(time().dateTime).toBe(task.lastProgressAt);
  expect(time().textContent).toBe(text);
  expect(time().title).toMatch(/^Updated /);
  rerender(<TaskCard task={{...task,progress:.6,lastProgressAt:"2026-10-19T03:00:00Z"}} onOpen={vi.fn()} />);
  expect(time().dateTime).toBe("2026-10-19T03:00:00Z");
 });
 it("uses creation wording for untracked history and real milestones when available",()=>{
  const {container,rerender}=render(<TaskCard task={{...task,lastProgressAt:null}} onOpen={vi.fn()} />);
  expect(container.querySelector("time")?.dateTime).toBe(task.createdAt);
  expect(container.querySelector("time")?.textContent).toMatch(/^Created /);
  rerender(<TaskCard task={{...task,lastProgressAt:null,sourcingCompletedAt:"2026-10-18T01:10:00Z",downloadCompletedAt:"2026-10-18T02:30:00Z",ingestCompletedAt:"2026-10-18T02:50:00Z"}} onOpen={vi.fn()} />);
  expect(container.querySelector("time")?.dateTime).toBe("2026-10-18T02:50:00Z");
  expect(container.querySelector("time")?.textContent).toMatch(/^Updated /);
 });
});
