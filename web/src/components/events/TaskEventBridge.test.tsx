// @vitest-environment jsdom
import { act, cleanup, render } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
const mocks = vi.hoisted(() => ({ options: null as any, query: vi.fn() }));
vi.mock("urql", () => ({ useClient: () => ({query:mocks.query}) }));
vi.mock("../../hooks/useTaskEvents", () => ({ useTaskEvents: (options:any) => {mocks.options=options;return {}} }));
import { TaskEventBridge } from "./TaskEventBridge";
import { recordTaskEvent } from "../../graphql/taskRealtime";
import { getTaskSnapshotRecoveryGeneration } from "../../graphql/taskRecovery";
afterEach(cleanup);
it("notifies mounted owners after READY without fetching any task list", () => {
 render(<TaskEventBridge/>);const before=getTaskSnapshotRecoveryGeneration();
 act(() => recordTaskEvent({type:"READY",sequence:12,streamId:"bridge-test",taskId:"",listChanged:false}));
 expect(getTaskSnapshotRecoveryGeneration()).toBe(before+1);
 expect(mocks.query).not.toHaveBeenCalled();
});
it("recovers after a sequence gap without a global full snapshot", () => {
 render(<TaskEventBridge/>);const before=getTaskSnapshotRecoveryGeneration();
 act(() => mocks.options.onSequenceGap());
 expect(getTaskSnapshotRecoveryGeneration()).toBe(before+1);
 expect(mocks.query).not.toHaveBeenCalled();
});
