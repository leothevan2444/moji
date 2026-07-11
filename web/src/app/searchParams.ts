import type { TaskSortKey, TaskStatusFilter } from "../types";

export type TaskStatusParam = "all" | "active" | "completed" | "failed" | "pending-scan";
export type TaskSortParam = "newest" | "updated" | "progress";

const taskStatusFromParam: Record<TaskStatusParam, TaskStatusFilter> = {
  all: "全部",
  active: "运行中",
  completed: "完成",
  failed: "失败",
  "pending-scan": "待扫描"
};
const taskStatusToParam = Object.fromEntries(
  Object.entries(taskStatusFromParam).map(([key, value]) => [value, key])
) as Record<TaskStatusFilter, TaskStatusParam>;

const taskSortFromParam: Record<TaskSortParam, TaskSortKey> = {
  newest: "最新",
  updated: "更新时间",
  progress: "进度"
};
const taskSortToParam = Object.fromEntries(
  Object.entries(taskSortFromParam).map(([key, value]) => [value, key])
) as Record<TaskSortKey, TaskSortParam>;

export interface TaskSearchState {
  q: string;
  status: TaskStatusFilter;
  sort: TaskSortKey;
}

export function parseTaskSearchParams(params: URLSearchParams): TaskSearchState {
  const status = params.get("status") as TaskStatusParam | null;
  const sort = params.get("sort") as TaskSortParam | null;
  return {
    q: params.get("q")?.trim() ?? "",
    status: status && taskStatusFromParam[status] ? taskStatusFromParam[status] : "全部",
    sort: sort && taskSortFromParam[sort] ? taskSortFromParam[sort] : "最新"
  };
}

export function serializeTaskSearchParams(state: TaskSearchState): URLSearchParams {
  const params = new URLSearchParams();
  if (state.q.trim()) params.set("q", state.q.trim());
  if (state.status !== "全部") params.set("status", taskStatusToParam[state.status]);
  if (state.sort !== "最新") params.set("sort", taskSortToParam[state.sort]);
  return params;
}

export interface PerformerSearchState {
  q: string;
  page: number;
  pageSize: number;
  sceneQ: string;
  source: "all" | "stash" | "stashbox";
  library: "all" | "in-library" | "not-in-library";
  scenePage: number;
  scenePageSize: number;
}

const PAGE_SIZES = [12, 24, 48, 96] as const;
function positiveInt(value: string | null, fallback: number) {
  const parsed = Number.parseInt(value ?? "", 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

export function parsePerformerSearchParams(params: URLSearchParams): PerformerSearchState {
  const pageSize = positiveInt(params.get("pageSize"), 24);
  const scenePageSize = positiveInt(params.get("scenePageSize"), 24);
  const source = params.get("source");
  const library = params.get("library");
  return {
    q: params.get("q")?.trim() ?? "",
    page: positiveInt(params.get("page"), 1),
    pageSize: PAGE_SIZES.includes(pageSize as (typeof PAGE_SIZES)[number]) ? pageSize : 24,
    sceneQ: params.get("sceneQ")?.trim() ?? "",
    source: source === "stash" || source === "stashbox" ? source : "all",
    library: library === "in-library" || library === "not-in-library" ? library : "all",
    scenePage: positiveInt(params.get("scenePage"), 1),
    scenePageSize: PAGE_SIZES.includes(scenePageSize as (typeof PAGE_SIZES)[number]) ? scenePageSize : 24
  };
}

export function serializePerformerSearchParams(state: PerformerSearchState): URLSearchParams {
  const params = new URLSearchParams();
  if (state.q) params.set("q", state.q);
  if (state.page !== 1) params.set("page", String(state.page));
  if (state.pageSize !== 24) params.set("pageSize", String(state.pageSize));
  if (state.sceneQ) params.set("sceneQ", state.sceneQ);
  if (state.source !== "all") params.set("source", state.source);
  if (state.library !== "all") params.set("library", state.library);
  if (state.scenePage !== 1) params.set("scenePage", String(state.scenePage));
  if (state.scenePageSize !== 24) params.set("scenePageSize", String(state.scenePageSize));
  return params;
}

export interface DiscoverSearchState {
  q: string;
  source: "stashbox" | "jackett";
  sort: string;
  page: number;
  trackers: string[];
  fastRules?: boolean;
  fileRules?: boolean;
}

const STASHBOX_SORTS = new Set(["relevance", "date-desc", "date-asc", "duration-desc", "title-asc"]);
const JACKETT_SORTS = new Set(["relevance", "seeders-desc", "size-desc", "date-desc"]);

export function parseDiscoverSearchParams(params: URLSearchParams): DiscoverSearchState {
  const source = params.get("source") === "jackett" ? "jackett" : "stashbox";
  const rawSort = params.get("sort") ?? "relevance";
  const allowed = source === "stashbox" ? STASHBOX_SORTS : JACKETT_SORTS;
  const booleanParam = (key: string) => {
    const value = params.get(key);
    return value === "1" ? true : value === "0" ? false : undefined;
  };
  return {
    q: params.get("q")?.trim() ?? "",
    source,
    sort: allowed.has(rawSort) ? rawSort : "relevance",
    page: positiveInt(params.get("page"), 1),
    trackers: [...new Set((params.get("trackers") ?? "").split(",").map((item) => item.trim()).filter(Boolean))].sort(),
    fastRules: booleanParam("fastRules"),
    fileRules: booleanParam("fileRules")
  };
}

export function serializeDiscoverSearchParams(state: DiscoverSearchState): URLSearchParams {
  const params = new URLSearchParams();
  if (state.q) params.set("q", state.q);
  if (state.source !== "stashbox") params.set("source", state.source);
  if (state.sort !== "relevance") params.set("sort", state.sort);
  if (state.page !== 1) params.set("page", String(state.page));
  if (state.trackers.length) params.set("trackers", [...new Set(state.trackers)].sort().join(","));
  if (state.fastRules !== undefined) params.set("fastRules", state.fastRules ? "1" : "0");
  if (state.fileRules !== undefined) params.set("fileRules", state.fileRules ? "1" : "0");
  return params;
}
