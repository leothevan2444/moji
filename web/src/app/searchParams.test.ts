import { describe, expect, it } from "vitest";
import { parseDiscoverSearchParams, parsePerformerSearchParams, parseTaskSearchParams, serializeTaskSearchParams } from "./searchParams";

describe("task search params", () => {
  it("uses stable defaults and omits them when serializing", () => {
    const state = parseTaskSearchParams(new URLSearchParams("status=unknown&sort=unknown"));
    expect(state).toEqual({ q: "", status: "全部", sort: "最新" });
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
});

describe("discover search params", () => {
  it("deduplicates trackers and validates source-specific sorting", () => {
    const state = parseDiscoverSearchParams(new URLSearchParams("source=jackett&sort=title-asc&trackers=b,a,b&fastRules=1"));
    expect(state.sort).toBe("relevance");
    expect(state.trackers).toEqual(["a", "b"]);
    expect(state.fastRules).toBe(true);
  });
});
