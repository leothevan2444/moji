// @vitest-environment jsdom

import { act, cleanup, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  taskOptions: null as null | { onSequenceGap?: () => void; onReconnect?: () => void },
  connectionStatus: "connected",
  query: vi.fn(() => ({ toPromise: () => Promise.resolve({ error: undefined }) }))
}));

vi.mock("urql", () => ({ useClient: () => ({ query: mocks.query }) }));
vi.mock("../../hooks/useTaskEvents", () => ({
  useTaskEvents: (options: typeof mocks.taskOptions) => {
    mocks.taskOptions = options;
    return { connectionStatus: mocks.connectionStatus };
  }
}));

import { StrictMode } from "react";
import { HomePageDocumentDocument } from "../../graphql/generated/graphql";
import { TaskEventBridge } from "./TaskEventBridge";

describe("TaskEventBridge", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mocks.taskOptions = null;
    mocks.connectionStatus = "connected";
    mocks.query.mockReset();
    mocks.query.mockReturnValue({ toPromise: () => Promise.resolve({ error: undefined }) });
  });
  afterEach(() => { cleanup(); vi.useRealTimers(); });

  it("coalesces initial, gap and reconnect calibration into one network query", () => {
    render(<TaskEventBridge delay={200} />);
    act(() => {
      mocks.taskOptions?.onSequenceGap?.();
      mocks.taskOptions?.onReconnect?.();
      vi.advanceTimersByTime(200);
    });
    expect(mocks.query).toHaveBeenCalledTimes(1);
    expect(mocks.query).toHaveBeenCalledWith(expect.anything(), {}, { requestPolicy: "network-only" });
  });

  it("clears pending calibration on unmount", () => {
    const view = render(<TaskEventBridge delay={200} />);
    view.unmount();
    act(() => vi.advanceTimersByTime(200));
    expect(mocks.query).not.toHaveBeenCalled();
  });
  it("does not overlap recovery queries and follows a gap during an in-flight snapshot once", async () => {
    let complete!: (value: { error: undefined }) => void;
    mocks.query.mockReturnValueOnce({ toPromise: () => new Promise(resolve => { complete = resolve; }) });
    render(<TaskEventBridge delay={200} />);
    await act(async () => { vi.advanceTimersByTime(200); });
    expect(mocks.query).toHaveBeenCalledTimes(1);
    await act(async () => {
      for (let i = 0; i < 20; i++) mocks.taskOptions?.onSequenceGap?.();
      vi.advanceTimersByTime(1000);
    });
    expect(mocks.query).toHaveBeenCalledTimes(1);
    await act(async () => { complete({ error: undefined }); });
    await act(async () => { vi.advanceTimersByTime(200); });
    expect(mocks.query).toHaveBeenCalledTimes(2);
  });

  it("keeps initial calibration under StrictMode effect replay", async () => {
    render(<StrictMode><TaskEventBridge delay={200} /></StrictMode>);
    await act(async () => { vi.advanceTimersByTime(200); });
    expect(mocks.query).toHaveBeenCalledTimes(1);
  });

  it("uses the current route scope for a recovery queued behind another query", async () => {
    let complete!: (value: { error: undefined }) => void;
    mocks.query.mockReturnValueOnce({ toPromise: () => new Promise(resolve => { complete = resolve; }) });
    const view = render(<TaskEventBridge delay={200} scope="tasks" />);
    await act(async () => { vi.advanceTimersByTime(200); });
    view.rerender(<TaskEventBridge delay={200} scope="home" />);
    act(() => { mocks.taskOptions?.onSequenceGap?.(); });
    await act(async () => { complete({ error: undefined }); });
    await act(async () => { vi.advanceTimersByTime(200); });
    expect(mocks.query).toHaveBeenLastCalledWith(HomePageDocumentDocument, {}, { requestPolicy: "network-only" });
  });

  it("does not start a queued recovery after unmount", async () => {
    let complete!: (value: { error: undefined }) => void;
    mocks.query.mockReturnValueOnce({ toPromise: () => new Promise(resolve => { complete = resolve; }) });
    const view = render(<TaskEventBridge delay={200} />);
    await act(async () => { vi.advanceTimersByTime(200); });
    act(() => { mocks.taskOptions?.onSequenceGap?.(); });
    view.unmount();
    await act(async () => { complete({ error: undefined }); });
    act(() => { vi.advanceTimersByTime(200); });
    expect(mocks.query).toHaveBeenCalledTimes(1);
  });

});
