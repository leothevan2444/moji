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
  if (!configured) return { labelKey: "home.status.unconfigured", tone: "tone-neutral" as const };
  if (ready) return { labelKey: "home.status.enabled", tone: "tone-success" as const };
  if (lastError) return { labelKey: "home.status.error", tone: "tone-danger" as const };
  if (!okAt) return { labelKey: "home.status.pending", tone: "tone-warn" as const };
  return { labelKey: "home.status.stale", tone: "tone-warn" as const };
}
