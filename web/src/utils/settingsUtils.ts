import type { DashboardDocumentQuery } from "../graphql/generated/graphql";

type RuntimeSettings = NonNullable<DashboardDocumentQuery["settings"]>;

export function boolState(value: boolean, positive = "已配置", negative = "未配置") {
  return value ? positive : negative;
}

export function serviceStatus(configured: boolean, enabled: boolean) {
  if (enabled) return { label: "已启用", tone: "tone-success" };
  if (configured) return { label: "待启用", tone: "tone-warn" };
  return { label: "未配置", tone: "tone-neutral" };
}

export function taskSyncStatus(settings: RuntimeSettings["tasks"]) {
  return settings.progressSyncEnabled ? "已启用" : "未启用";
}
