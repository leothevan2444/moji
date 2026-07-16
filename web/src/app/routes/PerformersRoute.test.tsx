// @vitest-environment jsdom

import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { buildPerformerSceneQueueInput, type SceneSnapshotSource, usePerformerSceneSelection } from "./performerSceneSelection";

function scene(key: string): SceneSnapshotSource {
  return {
    key,
    sourceSceneId: `source-${key}`,
    stashBoxSceneId: `stashbox-${key}`,
    stashBoxEndpoint: "https://stashbox.example/graphql",
    code: key.toUpperCase(),
    title: `Scene ${key}`,
    inLibrary: false
  };
}

describe("PerformersRoute cross-page scene selection", () => {
  it("submits selections from two pages exactly once and preserves them across page and filter changes", () => {
    const pageOne = [scene("one"), scene("shared")];
    const pageTwo = [scene("two"), scene("shared")];
    const { result, rerender } = renderHook(
      ({ performerId, visible }) => usePerformerSceneSelection(performerId, visible),
      { initialProps: { performerId: "performer-1", visible: pageOne } }
    );

    act(() => result.current.addVisible(["one", "shared"]));
    rerender({ performerId: "performer-1", visible: pageTwo });
    expect(result.current.selectedKeys).toEqual(["one", "shared"]);

    act(() => result.current.addVisible(["two", "shared"]));
    rerender({ performerId: "performer-1", visible: [] });
    const input = buildPerformerSceneQueueInput("performer-1", result.current.selected);
    expect(input.scenes.map((item) => item.key)).toEqual(["one", "shared", "two"]);
    expect(new Set(input.scenes.map((item) => item.key)).size).toBe(3);
  });

  it("clears queued keys while retaining skipped, failed, and unreported keys", () => {
    const visible = [scene("queued"), scene("skipped"), scene("failed"), scene("unreported")];
    const { result } = renderHook(() => usePerformerSceneSelection("performer-1", visible));
    act(() => result.current.addVisible(visible.map((item) => item.key)));

    act(() => result.current.applyResults([
      { key: "queued", status: "QUEUED" },
      { key: "skipped", status: "SKIPPED" },
      { key: "failed", status: "FAILED" }
    ]));

    expect(result.current.selectedKeys).toEqual(["skipped", "failed", "unreported"]);
  });

  it("clears all scene selections when the performer changes", () => {
    const { result, rerender } = renderHook(
      ({ performerId, visible }) => usePerformerSceneSelection(performerId, visible),
      { initialProps: { performerId: "performer-1", visible: [scene("one")] } }
    );
    act(() => result.current.toggle("one"));
    expect(result.current.selectedKeys).toEqual(["one"]);

    rerender({ performerId: "performer-2", visible: [scene("two")] });
    expect(result.current.selectedKeys).toEqual([]);
  });
});
