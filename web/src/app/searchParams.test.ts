import { describe, expect, it } from "vitest";
import { parseDiscoverSearchParams, parsePerformerSearchParams, parseTaskSearchParams, serializeDiscoverSearchParams, serializePerformerSearchParams, serializeTaskSearchParams } from "./searchParams";

describe("task search params", () => {
  it("uses stable defaults and omits them when serializing", () => {
    const state = parseTaskSearchParams(new URLSearchParams("status=unknown&sort=unknown"));
    expect(state).toEqual({ q: "", status: "all", sort: "createdAt" });
    expect(serializeTaskSearchParams(state).toString()).toBe("");
  });

  it("round trips public URL values", () => {
    const state = parseTaskSearchParams(new URLSearchParams("q=abc&status=failed&sort=updated"));
    expect(serializeTaskSearchParams(state).toString()).toBe("q=abc&status=failed&sort=updated");
  });
});

describe("performer search params", () => {
  it("normalizes invalid pages and page sizes", () => {
    const state = parsePerformerSearchParams(new URLSearchParams("page=-2&pageSize=13&source=nope"));
    expect(state.page).toBe(1);
    expect(state.pageSize).toBe(24);
    expect(state.source).toBe("all");
  });

  it("round trips shareable list and scene state", () => {
    const state = parsePerformerSearchParams(new URLSearchParams("q=a&page=2&sceneQ=b&source=stashbox&library=not-in-library&scenePage=3"));
    expect(parsePerformerSearchParams(serializePerformerSearchParams(state))).toEqual(state);
  });
});

describe("discover search params", () => {
  it("deduplicates trackers and validates source-specific sorting", () => {
    const state = parseDiscoverSearchParams(new URLSearchParams("source=jackett&sort=title-asc&trackers=b,a,b&fastRules=1"));
    expect(state.sort).toBe("relevance");
    expect(state.trackers).toEqual(["a", "b"]);
    expect(state.fastRules).toBe(true);
  });

  it("round trips canonical discovery state", () => {
    const state = parseDiscoverSearchParams(new URLSearchParams("q=abc&source=jackett&sort=seeders-desc&page=2&trackers=b,a&fastRules=1&fileRules=0"));
    expect(parseDiscoverSearchParams(serializeDiscoverSearchParams(state))).toEqual(state);
  });
});
