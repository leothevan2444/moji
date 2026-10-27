import { afterEach, expect, it, vi } from "vitest";
import i18n from "../i18n/i18n";
import { formatDateTime, formatRelative, formatRelativeDate } from "./formatters";

afterEach(() => { vi.restoreAllMocks(); vi.useRealTimers(); });
it("reuses locale-specific formatters without caching the date or relative-time result", async () => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2026-10-27T12:00:00Z"));
  await i18n.changeLanguage("en");
  const OriginalDateTimeFormat = Intl.DateTimeFormat;
  const OriginalRelativeTimeFormat = Intl.RelativeTimeFormat;
  const date = vi.spyOn(Intl, "DateTimeFormat").mockImplementation(function (...args: ConstructorParameters<typeof Intl.DateTimeFormat>) { return new OriginalDateTimeFormat(...args); });
  const relative = vi.spyOn(Intl, "RelativeTimeFormat").mockImplementation(function (...args: ConstructorParameters<typeof Intl.RelativeTimeFormat>) { return new OriginalRelativeTimeFormat(...args); });
  const at = "2026-10-27T11:59:30Z";
  const english = formatDateTime(at);
  const before = formatRelative(at);
  for (let i = 0; i < 100; i++) { formatDateTime(at); formatRelative(at); formatRelativeDate(at); }
  expect(date).toHaveBeenCalledTimes(2);
  expect(relative).toHaveBeenCalledTimes(1);
  vi.setSystemTime(new Date("2026-10-27T12:05:00Z"));
  expect(formatRelative(at)).not.toBe(before);
  await i18n.changeLanguage("zh-CN");
  expect(formatDateTime(at)).not.toBe(english);
  formatRelative(at);
  expect(date).toHaveBeenCalledTimes(3);
  expect(relative).toHaveBeenCalledTimes(2);
  await i18n.changeLanguage("en");
  expect(formatDateTime(at)).toBe(english);
  expect(date).toHaveBeenCalledTimes(3);
});
