import { getAppTimezone } from "./appTimezone";
import type { LogsDocumentQuery } from "../graphql/generated/graphql";
import i18n from "../i18n/i18n";

function activeLocale() {
  return i18n.resolvedLanguage?.toLowerCase().startsWith("zh") ? "zh-CN" : "en";
}

const dateTimeFormatters = new Map<string, Intl.DateTimeFormat>();
const relativeTimeFormatters = new Map<string, Intl.RelativeTimeFormat>();

function dateTimeFormatter(style: "full" | "date" | "compact" | "time" = "full", calendarDate = false) {
  const locale = activeLocale();
  const timezone = calendarDate ? "UTC" : getAppTimezone();
  const key = `${locale}:${timezone}:${style}`;
  let formatter = dateTimeFormatters.get(key);
  if (!formatter) {
    formatter = new Intl.DateTimeFormat(locale, { timeZone: timezone, ...(style === "time"
      ? { hour: "2-digit", minute: "2-digit", second: "2-digit" }
      : style === "compact"
      ? { year: "numeric", month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit", hourCycle: "h23" }
      : style === "date"
      ? { month: "short", day: "numeric" }
      : { year: "numeric", month: "short", day: "numeric", hour: "2-digit", minute: "2-digit", second: "2-digit" }) });
    dateTimeFormatters.set(key, formatter);
  }
  return formatter;
}

function relativeTimeFormatter() {
  const locale = activeLocale();
  let formatter = relativeTimeFormatters.get(locale);
  if (!formatter) {
    formatter = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
    relativeTimeFormatters.set(locale, formatter);
  }
  return formatter;
}

export function formatBytes(size: number) {
  if (!size) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(Math.floor(Math.log(size) / Math.log(1024)), units.length - 1);
  const value = size / 1024 ** index;
  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

/**
 * Format a transfer rate in bytes/second as a human-readable string with a
 * trailing `/s`. Negative or NaN inputs collapse to "0 B/s".
 */
export function formatBytesRate(bytesPerSec: number) {
  if (!Number.isFinite(bytesPerSec) || bytesPerSec <= 0) return "0 B/s";
  return `${formatBytes(bytesPerSec)}/s`;
}

export function formatDateTime(value?: string | null, style: "full" | "compact" = "full") {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return dateTimeFormatter(style).format(date);
}

export function formatTime(value?: string | Date | null) {
  if (!value) return "—";
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return typeof value === "string" ? value : "—";
  return dateTimeFormatter("time").format(date);
}

export function formatLogEntries(entries: LogsDocumentQuery["logs"]) {
  return entries
    .map((entry) => `${entry.time} [${entry.level}] ${entry.message}`)
    .join("\n");
}

export function formatRelativeDate(value?: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return dateTimeFormatter("date", /^\d{4}-\d{2}-\d{2}$/.test(value)).format(date);
}

/**
 * 把来源 publishDate 规范成 yyyy-mm-dd：日历日期保留原值，时间戳按
 * Moji 时区转换。无法解析的值保留来源前 10 位；空值显示占位。
 */
export function formatPublishDate(value?: string | null) {
  if (!value) return "—";
  // A calendar date has no offset; timestamps follow the configured timezone.
  if (/^\d{4}-\d{2}-\d{2}$/.test(value)) return value;
  const parsed = new Date(value);
  if (!Number.isNaN(parsed.getTime())) {
    const parts = new Intl.DateTimeFormat("en", { timeZone: getAppTimezone(), year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(parsed);
    const part = (type: Intl.DateTimeFormatPartTypes) => parts.find(item => item.type === type)?.value;
    return `${part("year")}-${part("month")}-${part("day")}`;
  }
  // 解析失败时回退到字符串前 10 位（如 "2025-01-08 12:34" → "2025-01-08"）。
  return value.slice(0, 10);
}

export function formatDurationSeconds(value?: number | null) {
  if (!value || value <= 0) return "";
  const total = Math.floor(value);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = total % 60;

  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
  }
  return `${minutes}:${String(seconds).padStart(2, "0")}`;
}

/**
 * Format an ISO timestamp as a short relative string (e.g. "刚刚", "5s 前",
 * "2m 前", "1h 前", "3d 前"). Returns null when the input is missing or
 * unparseable so callers can decide how to render the absence.
 */
export function formatRelative(iso?: string | null): string | null {
  if (!iso) return null;
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return null;
  const diffMs = Date.now() - then;
  const formatter = relativeTimeFormatter();
  if (diffMs < 0) return formatter.format(0, "second");
  const sec = Math.floor(diffMs / 1000);
  if (sec < 10) return formatter.format(0, "second");
  if (sec < 60) return formatter.format(-sec, "second");
  const min = Math.floor(sec / 60);
  if (min < 60) return formatter.format(-min, "minute");
  const hr = Math.floor(min / 60);
  if (hr < 24) return formatter.format(-hr, "hour");
  const day = Math.floor(hr / 24);
  return formatter.format(-day, "day");
}
