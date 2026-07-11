import { describe, expect, it } from "vitest";
import { matchRoutes } from "react-router";
import { appRoutes } from "./routeConfig";
import { taskCloseTarget } from "./routes/TasksRoute";

describe("SPA route boundaries", () => {
  it("declares every business surface as a lazy child route", () => {
    const children = appRoutes.find((route) => route.children)?.children ?? [];
    expect(children.map((route) => route.index ? "/" : route.path)).toEqual([
      "/", "/tasks", "/tasks/:taskId", "/tasks/:taskId/resolve", "/discover",
      "/performers", "/performers/:performerId", "/stats", "/settings/:section"
    ]);
    expect(children.every((route) => typeof route.lazy === "function")).toBe(true);
  });

  it("returns to the background URL when closing a deep-linked task drawer", () => {
    expect(taskCloseTarget({ backgroundLocation: { pathname: "/discover", search: "?q=abc" } }, new URLSearchParams())).toBe("/discover?q=abc");
    expect(taskCloseTarget(null, new URLSearchParams("status=failed"))).toBe("/tasks?status=failed");
  });

  it.each([
    ["/tasks/task-1", "/tasks/:taskId"],
    ["/tasks/task-1/resolve", "/tasks/:taskId/resolve"],
    ["/performers/performer-1", "/performers/:performerId"],
    ["/settings/automation", "/settings/:section"]
  ])("matches deep link %s to its lazy route", (url, expectedPath) => {
    expect(matchRoutes(appRoutes, url)?.at(-1)?.route.path).toBe(expectedPath);
  });
});
