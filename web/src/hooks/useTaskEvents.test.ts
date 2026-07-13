// @vitest-environment jsdom

import { describe, expect, it } from "vitest";
import { evaluateTaskEventSequence } from "./useTaskEvents";

describe("evaluateTaskEventSequence", () => {
  it("accepts the first sequence without reporting a gap", () => {
    expect(evaluateTaskEventSequence(null, 42)).toEqual({
      accept: true,
      gap: false,
      nextSequence: 42
    });
  });

  it("accepts a consecutive sequence", () => {
    expect(evaluateTaskEventSequence(42, 43)).toEqual({
      accept: true,
      gap: false,
      nextSequence: 43
    });
  });

  it("ignores duplicate and old sequences", () => {
    expect(evaluateTaskEventSequence(42, 42)).toEqual({ accept: false, gap: false, nextSequence: 42 });
    expect(evaluateTaskEventSequence(42, 40)).toEqual({ accept: false, gap: false, nextSequence: 42 });
  });

  it("accepts a newer sequence and reports a gap", () => {
    expect(evaluateTaskEventSequence(42, 45)).toEqual({
      accept: true,
      gap: true,
      nextSequence: 45
    });
  });
});
