import { describe, expect, it, vi } from "vitest";
import { createTaskEventSequenceObserver } from "./taskEventSequence";

describe("task event transport sequences", () => {
  it("checks every event in a burst even if React later renders only the last value", () => {
    const gap = vi.fn();
    const observe = createTaskEventSequenceObserver(gap);
    for (let sequence = 100; sequence < 1100; sequence++) {
      expect(observe(sequence, 1).accept).toBe(true);
    }
    expect(gap).not.toHaveBeenCalled();
    expect(observe(1101, 1)).toEqual({ accept: true, gap: true, nextSequence: 1101 });
    expect(gap).toHaveBeenCalledTimes(1);
  });

  it("rejects stale events before they overwrite newer cache data and resets after reconnect", () => {
    const gap = vi.fn();
    const observe = createTaskEventSequenceObserver(gap);
    observe(100, 1);
    expect(observe(99, 1).accept).toBe(false);
    expect(observe(100, 1).accept).toBe(false);
    expect(observe(1, 2)).toEqual({ accept: true, gap: false, nextSequence: 1 });
    expect(observe(2, 2).accept).toBe(true);
    expect(gap).not.toHaveBeenCalled();
  });
});
