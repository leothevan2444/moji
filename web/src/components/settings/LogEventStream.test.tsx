// @vitest-environment jsdom

import { act, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LogLevel } from "../../graphql/generated/graphql";

const mocks = vi.hoisted(() => ({
  options: null as null | {
    onEvent?: (entry: { sequence: number; time: string; level: LogLevel; message: string }) => void;
    onResync?: () => void;
  }
}));

vi.mock("../../hooks/useLogEvents", () => ({
  useLogEvents: (options: typeof mocks.options) => {
    mocks.options = options;
  }
}));

import { LogEventStream } from "./LogEventStream";

const entry = (sequence: number) => ({
  sequence,
  time: `2026-07-13T08:00:${String(sequence).padStart(2, "0")}Z`,
  level: LogLevel.Info,
  message: `entry ${sequence}`
});

describe("LogEventStream", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mocks.options = null;
  });

  afterEach(() => vi.useRealTimers());

  it("batches rapid entries in newest-first order", () => {
    const onEntries = vi.fn();
    render(<LogEventStream pause={false} batchDelay={75} onEntries={onEntries} onResync={vi.fn()} />);
    act(() => {
      mocks.options?.onEvent?.(entry(1));
      mocks.options?.onEvent?.(entry(2));
      vi.advanceTimersByTime(75);
    });
    expect(onEntries).toHaveBeenCalledTimes(1);
    expect(onEntries).toHaveBeenCalledWith([entry(2), entry(1)]);
  });

  it("clears pending entries when unmounted", () => {
    const onEntries = vi.fn();
    const view = render(<LogEventStream pause={false} batchDelay={75} onEntries={onEntries} onResync={vi.fn()} />);
    act(() => mocks.options?.onEvent?.(entry(1)));
    view.unmount();
    act(() => vi.advanceTimersByTime(75));
    expect(onEntries).not.toHaveBeenCalled();
  });
});
