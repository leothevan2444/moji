/**
 * Mode metadata for the ingest pipeline. Shared by the home IngestCard and
 * the Settings drawer ingest form so the explanation never drifts between
 * surfaces.
 *
 *  - `summary` describes what the mode does.
 *  - `caution` is a short operational warning callers should display verbatim.
 *  - `tone` picks the colour variant used for the side accent in the home card.
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
        summary: "适用于 qBittorrent 和 Stash 共享底层存储；Moji 会基于任务实际保存路径自动换算到所选 Stash 媒体库。",
        caution: "要求任务能拿到真实保存目录与内容路径；换算失败时不会自动退回其他模式。"
      };
    case "TRANSFER":
      return {
        title: "文件交付",
        tone: "tone-warn",
        summary: "由 Moji 读取下载区文件并交付到媒体库挂载目录，再换算为 Stash 媒体库路径触发扫描。",
        caution: "Moji 需要同时访问下载区和媒体库；目标已有同名文件或目录时会直接失败。"
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
      return "路径映射";
    case "TRANSFER":
      return "文件交付";
    default:
      return mode || "未选择";
  }
}

export function transferActionLabel(action: string) {
  switch (action) {
    case "COPY":
      return "复制";
    case "MOVE":
      return "移动";
    case "SYMLINK":
      return "符号链接";
    default:
      return action || "—";
  }
}

export const INGEST_BLOCKERS = [
  "任务完成后无法入库",
  "订阅扫描无目标库",
  "Stash-Box 元数据无法获取"
];
