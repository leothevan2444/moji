import { describe, expect, it } from "vitest";
import { safeUpcomingSource, upcomingDays } from "./UpcomingReleases";
describe("upcoming release dates and links", () => {
 it("uses the business calendar at midnight", () => {
  const now = new Date("2026-10-31T16:00:00Z");
  expect(upcomingDays("2026-11-01", "Asia/Shanghai", now)).toBe(0);
  expect(upcomingDays("2026-11-01", "UTC", now)).toBe(1);
  expect(upcomingDays("2026-11-01", "Local", now, "2026-11-01")).toBe(0);
  expect(upcomingDays("invalid", "UTC", now)).toBe(null);
 });
 it("only opens explicit safe source URLs", () => {
  expect(safeUpcomingSource("https://example.com/release/1")).toBe("https://example.com/release/1");
  for (const raw of ["javascript:alert(1)", "/release/1", "https://secret@example.com/", ""]) expect(safeUpcomingSource(raw)).toBe(null);
 });
});
