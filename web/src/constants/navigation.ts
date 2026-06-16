import type { TabKey, SettingsTab } from "../types";

export const NAV_TABS: TabKey[] = ["主页", "任务", "订阅", "发现"];

export const SETTINGS_TABS: SettingsTab[] = [
  "Stash",
  "索引器",
  "下载器",
  "任务",
  "订阅",
  "安全性",
  "系统",
  "日志",
  "工具",
  "更新历史",
  "关于"
];

export const ENABLED_SETTINGS_TABS: ReadonlySet<SettingsTab> = new Set([
  "Stash",
  "索引器",
  "下载器",
  "任务",
  "订阅",
  "日志",
  "系统"
]);
