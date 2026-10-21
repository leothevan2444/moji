import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { expect, it } from "vitest";
import { APP_VERSION } from "./version";

it("exposes the Git-derived build version to the frontend", () => {
  const expected = process.env.MOJI_BUILD_VERSION?.trim() || execFileSync(
    "sh", [fileURLToPath(new URL("../../scripts/version.sh", import.meta.url))],
    { encoding: "utf8" },
  ).trim();
  expect(APP_VERSION).toBe(expected);
});
