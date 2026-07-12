/**
 * Mode metadata for the ingest pipeline. Shared by the home IngestCard and
 * the Settings drawer ingest form so the explanation never drifts between
 * surfaces.
 */
export interface IngestModeGuide {
  title: string;
  tone: "tone-info" | "tone-warn" | "tone-danger" | "tone-neutral";
  summary: string;
  caution: string;
}

export function deliveryModeGuide(mode: string): IngestModeGuide {
  switch (mode) {
    case "PATH_MAP":
      return {
        title: "路径映射",
        tone: "tone-info",
        summary: "Moji 只负责把 qB 下载路径翻译成 Stash 扫描路径，不直接搬运文件。",
        caution: "要求先配置 qB 下载根路径和 Stash 媒体库根路径；两者使用各自命名空间。"
      };
    case "TRANSFER":
      return {
        title: "文件交付",
        tone: "tone-warn",
        summary: "Moji 先把 qB 下载路径翻译成自己的可操作路径，再交付到媒体库并换算为 Stash 扫描路径。",
        caution: "要求同时配置 qB、Moji、Stash 三套根路径；目标已有同名文件或目录时会直接失败。"
      };
    default:
      return {
        title: "未选择工作方式",
        tone: "tone-neutral",
        summary: "请选择入库策略后再继续。",
        caution: ""
      };
  }
}

export function deliveryModeLabel(mode: string) {
  switch (mode) {
    case "PATH_MAP":
      return i18n.t("home.ingest.pathMap");
    case "TRANSFER":
      return i18n.t("home.ingest.transfer");
    default:
      return mode || i18n.t("home.ingest.none");
  }
}

export function transferActionLabel(action: string) {
  switch (action) {
    case "COPY":
      return i18n.t("home.ingest.copy");
    case "MOVE":
      return i18n.t("home.ingest.move");
    case "SYMLINK":
      return i18n.t("home.ingest.symlink");
    default:
      return action || "—";
  }
}

export const INGEST_BLOCKERS = [
  "qB 下载根路径未映射",
  "Stash 媒体库根路径未映射",
  "任务完成后无法闭环换算扫描路径"
];
import i18n from "../i18n/i18n";
