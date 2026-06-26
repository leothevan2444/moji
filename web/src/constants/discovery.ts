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
  { value: DiscoverSortBy.Relevance, label: "默认（相关度）" },
  { value: DiscoverSortBy.DateDesc, label: "日期从新到旧" },
  { value: DiscoverSortBy.DateAsc, label: "日期从旧到新" },
  { value: DiscoverSortBy.DurationDesc, label: "时长从长到短" },
  { value: DiscoverSortBy.TitleAsc, label: "标题 A → Z" }
];

/** Jackett 排序选项。 */
export const JACKETT_SORT_OPTIONS: ReadonlyArray<{ value: JackettSortBy; label: string }> = [
  { value: JackettSortBy.Relevance, label: "默认（相关度）" },
  { value: JackettSortBy.SeedersDesc, label: "种子数从多到少" },
  { value: JackettSortBy.SizeDesc, label: "体积从大到小" },
  { value: JackettSortBy.DateDesc, label: "发布日期从新到旧" }
];

/** placeholder 轮播文案。聚焦输入框时会停止轮播，避免抖动。 */
export const SEARCH_PLACEHOLDERS: readonly string[] = [
  "输入番号、标题、演员或关键词",
  "试试 STARS-001 / IPX-789",
  "按 / 聚焦搜索框",
  "已订阅演员会自动出现在推荐里"
];

/**
 * 单页大小。后端每次返回 50 条，前端按 PAGE_SIZE 切片显示。
 * limit 与 pageSize 配合可同时支持前 N 页的快速翻页。
 */
export const DISCOVERY_PAGE_SIZE = 10;
export const DISCOVERY_RESULT_LIMIT = 50;