/**
 * Application-level type definitions shared across components.
 */

export type TabKey = "主页" | "任务" | "演员" | "发现";
export type DrawerKey = "stats" | "settings" | "help" | "task" | null;

export type ToastTone = "tone-success" | "tone-danger" | "tone-info";
export type ToastPhase = "entering" | "leaving";
export type ToastItem = { id: number; tone: ToastTone; message: string; phase: ToastPhase };

export type SettingsTab =
  | "连接"
  | "入库"
  | "自动化"
  | "日志"
  | "关于";
