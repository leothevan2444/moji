// @vitest-environment jsdom

import { act, render } from "@testing-library/react";
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

import { TaskEventBridge } from "./TaskEventBridge";

describe("TaskEventBridge", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mocks.taskOptions = null;
    mocks.connectionStatus = "connected";
    mocks.query.mockClear();
  });
  afterEach(() => vi.useRealTimers());

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
});
