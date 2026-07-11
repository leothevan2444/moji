import type { SettingsTab } from "../types";

export const NAV_ITEMS = [
  { label: "主页", to: "/", end: true },
  { label: "任务", to: "/tasks", end: false },
  { label: "演员", to: "/performers", end: false },
  { label: "发现", to: "/discover", end: false }
] as const;

export const SETTINGS_TABS: SettingsTab[] = ["连接", "入库", "自动化", "系统", "日志", "关于"];
