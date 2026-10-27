import { describe, expect, it } from "vitest";
import { performerImageURL, performerThumbnailURL } from "./performerUtils";

describe("performer image variants", () => {
  it("uses a thumbnail in the list and retains the original detail URL", () => {
    const path = `/api/images/${"a".repeat(64)}`;
    expect(performerThumbnailURL(path)).toBe(`${path}?variant=avatar`);
    expect(performerImageURL(path)).toBe(path);
    expect(performerThumbnailURL(`${path}?variant=old&other=1`)).toBe(`${path}?variant=avatar&other=1`);
  });

  it("keeps absent and non-proxied images compatible", () => {
    expect(performerThumbnailURL(null)).toBeNull();
    expect(performerThumbnailURL("/portrait.jpg", "https://stash.test")).toBe("https://stash.test/portrait.jpg");
    expect(performerThumbnailURL("https://image.test/portrait.jpg")).toBe("https://image.test/portrait.jpg");
  });
});
