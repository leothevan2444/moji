/**
 * Application-level type definitions shared across components.
 */

export type TabKey = "主页" | "任务" | "订阅" | "发现";
export type DrawerKey = "stats" | "settings" | "help" | "task" | null;

export type ToastTone = "tone-success" | "tone-danger" | "tone-info";
export type ToastPhase = "entering" | "leaving";
export type ToastItem = { id: number; tone: ToastTone; message: string; phase: ToastPhase };

export type SettingsTab =
  | "Stash"
  | "索引器"
  | "下载器"
  | "任务"
  | "订阅"
  | "安全性"
  | "系统"
  | "日志"
  | "工具"
  | "更新历史"
  | "关于";
