import { describe, expect, it } from "vitest";
import { mergePerformerSelection, performerBatchEligibility } from "./performerUtils";

describe("performer batch helpers", () => {
  it("merges unique performer ids while respecting the limit", () => {
    expect(mergePerformerSelection(["p1", "p2"], ["p2", "p3", "p4"], 3)).toEqual(["p1", "p2", "p3"]);
  });

  it("separates selected performers by current subscription truth", () => {
    expect(performerBatchEligibility(["p1", "p2", "p3"], ["p2", "p3"])).toEqual({
      subscribedIDs: ["p2", "p3"],
      unsubscribedIDs: ["p1"]
    });
  });
});
