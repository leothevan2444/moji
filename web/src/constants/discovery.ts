import { DiscoverSortBy, TorrentSearchSort } from "../graphql/generated/graphql";

/**
 * 搜索模式分段控件的选项。
 * value 是发现页 URL 的来源标识（"stashbox" / "indexers"）。
 */
export const DISCOVERY_MODE_OPTIONS = [
  { value: "stashbox", label: "StashBox" },
  { value: "indexers", label: "torrentSearch.label" }
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

/** 资源搜索由服务器排序、分页。 */
export const INDEXER_SORT_OPTIONS: ReadonlyArray<{ value: TorrentSearchSort; label: string }> = [
  { value: TorrentSearchSort.Relevance, label: "discoverUi.sorts.relevance" },
  { value: TorrentSearchSort.SeedersDesc, label: "discoverUi.sorts.seedersDesc" },
  { value: TorrentSearchSort.SizeDesc, label: "discoverUi.sorts.sizeDesc" },
  { value: TorrentSearchSort.DateDesc, label: "discoverUi.sorts.publishDesc" }
];

/** placeholder 轮播文案。聚焦输入框时会停止轮播，避免抖动。 */
export const SEARCH_PLACEHOLDERS: readonly string[] = [
  "discoverUi.placeholders.first", "discoverUi.placeholders.second", "discoverUi.placeholders.third", "discoverUi.placeholders.fourth"
];

/**
 * StashBox 结果本地分组；资源搜索按此大小向服务器请求当前页。
 */
export const DISCOVERY_PAGE_SIZE = 10;
export const DISCOVERY_RESULT_LIMIT = 50;
