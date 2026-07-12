import { DiscoverSortBy, JackettSortBy } from "../graphql/generated/graphql";

/**
 * 搜索模式分段控件的选项。
 * value 与后端订阅状态使用的字符串保持一致（"stashbox" / "jackett"）。
 */
export const DISCOVERY_MODE_OPTIONS = [
  { value: "stashbox", label: "StashBox" },
  { value: "jackett", label: "Jackett" }
] as const;

export type DiscoveryMode = (typeof DISCOVERY_MODE_OPTIONS)[number]["value"];

/** StashBox 排序选项。前端只展示用户可理解的标签，value 透传给后端 enum。 */
export const DISCOVER_SORT_OPTIONS: ReadonlyArray<{ value: DiscoverSortBy; label: string }> = [
  { value: DiscoverSortBy.Relevance, label: "discoverUi.sorts.relevance" },
  { value: DiscoverSortBy.DateDesc, label: "discoverUi.sorts.dateDesc" },
  { value: DiscoverSortBy.DateAsc, label: "discoverUi.sorts.dateAsc" },
  { value: DiscoverSortBy.DurationDesc, label: "discoverUi.sorts.durationDesc" },
  { value: DiscoverSortBy.TitleAsc, label: "discoverUi.sorts.titleAsc" }
];

/** Jackett 排序选项。 */
export const JACKETT_SORT_OPTIONS: ReadonlyArray<{ value: JackettSortBy; label: string }> = [
  { value: JackettSortBy.Relevance, label: "discoverUi.sorts.relevance" },
  { value: JackettSortBy.SeedersDesc, label: "discoverUi.sorts.seedersDesc" },
  { value: JackettSortBy.SizeDesc, label: "discoverUi.sorts.sizeDesc" },
  { value: JackettSortBy.DateDesc, label: "discoverUi.sorts.publishDesc" }
];

/** placeholder 轮播文案。聚焦输入框时会停止轮播，避免抖动。 */
export const SEARCH_PLACEHOLDERS: readonly string[] = [
  "discoverUi.placeholders.first", "discoverUi.placeholders.second", "discoverUi.placeholders.third", "discoverUi.placeholders.fourth"
];

/**
 * 单页大小。后端每次返回 50 条，前端按 PAGE_SIZE 切片显示。
 * limit 与 pageSize 配合可同时支持前 N 页的快速翻页。
 */
export const DISCOVERY_PAGE_SIZE = 10;
export const DISCOVERY_RESULT_LIMIT = 50;
