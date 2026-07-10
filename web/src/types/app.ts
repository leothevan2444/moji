/**
 * Application-level type definitions shared across components.
 */

export type TabKey = "主页" | "任务" | "演员" | "发现";
export type DrawerKey = "stats" | "settings" | "help" | "task" | "task-resolution" | "discovery" | "confirm" | null;

export type ToastTone = "tone-success" | "tone-danger" | "tone-info";
export type ToastPhase = "entering" | "leaving";
export type ToastItem = {
  id: number;
  tone: ToastTone;
  message: string;
  phase: ToastPhase;
  /** Override the default lifetime (ms) for the progress bar animation. */
  lifetimeMs?: number;
};

export type SettingsTab =
  | "连接"
  | "入库"
  | "自动化"
  | "系统"
  | "日志"
  | "关于";

export type TaskStatusFilter = "全部" | "运行中" | "完成" | "失败" | "待扫描";
export type TaskSortKey = "最新" | "更新时间" | "进度";
