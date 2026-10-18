import { beforeEach, describe, expect, it } from "vitest";
import i18n from "../i18n/i18n";
import { taskMetaLine, taskQueryLabel, taskSummary, type DashboardTask } from "./taskUtils";

const task = { id: "task-1", code: "DEMO-001", torrentName: "Release title", torrentUrl: "magnet:?xt=urn:btih:123", source: "MANUAL" } as DashboardTask;
beforeEach(async () => { await i18n.changeLanguage("en"); });

describe("stable task titles", () => {
  it("uses the code for magnet tasks and keeps it after the torrent name syncs", () => {
    expect(taskQueryLabel(task)).toBe("DEMO-001");
    expect(taskSummary({ ...task, torrentName: null })).toBe("DEMO-001");
    expect(taskSummary(task)).toBe("DEMO-001");
  });
  it.each([
    [" DEMO-001 ", "Release title", "DEMO-001"],
    ["", " Release title ", "Release title"],
    ["   ", "Release title", "Release title"],
    ["", "", "Manual magnet task"],
    ["", "   ", "Manual magnet task"]
  ])("falls back only when the code is missing (%s / %s)", (code, torrentName, expected) => {
    expect(taskSummary({ ...task, code, torrentName })).toBe(expected);
  });
  it("uses the task ID when neither a code, name, nor magnet label is available", () => {
    expect(taskSummary({ ...task, code: "", torrentName: null, torrentUrl: "" })).toBe(task.id);
  });
  it("does not repeat the title or torrent name in the subtitle", () => {
    expect(taskMetaLine(task)).toBe("");
    expect(taskMetaLine({ ...task, category: " movies " })).toBe("movies");
  });
});
