import type { LogsDocumentQuery } from "../graphql/generated/graphql";
import i18n from "../i18n/i18n";

function activeLocale() {
  return i18n.resolvedLanguage === "en" ? "en" : "zh-CN";
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

export function formatDateTime(value?: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(activeLocale(), {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit"
  }).format(date);
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
  return new Intl.DateTimeFormat(activeLocale(), {
    month: "short",
    day: "numeric"
  }).format(date);
}

/**
 * 把 Jackett/Tracker 返回的 publishDate 规范成 yyyy-mm-dd。Tracker 返回的
 * 日期格式各异（ISO、带时间、本地化字符串等），这里统一抽取前 10 位作为
 * 日期。空字符串返回 "—" 让调用方有占位渲染。
 */
export function formatPublishDate(value?: string | null) {
  if (!value) return "—";
  // 优先尝试 Date 解析，对 ISO 时间戳和本地化字符串都更稳。
  const parsed = new Date(value);
  if (!Number.isNaN(parsed.getTime())) {
    const year = parsed.getFullYear();
    const month = String(parsed.getMonth() + 1).padStart(2, "0");
    const day = String(parsed.getDate()).padStart(2, "0");
    return `${year}-${month}-${day}`;
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
  const formatter = new Intl.RelativeTimeFormat(activeLocale(), { numeric: "auto" });
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
