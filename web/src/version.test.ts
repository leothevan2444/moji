import { readFileSync } from "node:fs";
import { expect, it } from "vitest";
import { APP_VERSION } from "./version";

it("uses the root VERSION in the frontend build", () => {
  expect(APP_VERSION).toBe(readFileSync(new URL("../../VERSION", import.meta.url), "utf8").trim());
});
