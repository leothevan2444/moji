import type { DashboardDocumentQuery } from "../graphql/generated/graphql";

export function boolState(value: boolean, positive = "已配置", negative = "未配置") {
  return value ? positive : negative;
}

/**
 * Map a service's runtime readiness to a chip-friendly label and tone.
 *
 *   - configured=false         → 未配置
 *   - ready=true              → 已启用
 *   - ready=false + lastError → 运行异常
 *   - configured=true + no ok → 待检测
 *   - configured=true + stale → 数据陈旧
 */
export function serviceStatus(
  configured: boolean,
  ready: boolean,
  lastError: string | null,
  okAt?: string | null
) {
  if (!configured) return { label: "未配置", tone: "tone-neutral" as const };
  if (ready) return { label: "已启用", tone: "tone-success" as const };
  if (lastError) return { label: "运行异常", tone: "tone-danger" as const };
  if (!okAt) return { label: "待检测", tone: "tone-warn" as const };
  return { label: "数据陈旧", tone: "tone-warn" as const };
}
