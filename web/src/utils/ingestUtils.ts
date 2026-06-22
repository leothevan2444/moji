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

export function ingestModeGuide(mode: string): IngestModeGuide {
  switch (mode) {
    case "SHARED_STORAGE":
      return {
        title: "共享存储 / 路径映射",
        tone: "tone-info",
        summary: "适用于 qBittorrent 和 Stash 共用同一批文件，只是挂载路径不同。",
        caution: "要求下载路径命中 qBittorrent 路径前缀；映射失败时不会自动退回整库扫描。"
      };
    case "FILE_TRANSFER":
      return {
        title: "文件搬运",
        tone: "tone-warn",
        summary: "由 Moji 在本地文件系统执行复制或移动，成功后再扫描目标文件。",
        caution: "目标目录已有同名文件时会直接失败，不覆盖也不自动重命名。"
      };
    case "LIBRARY_SCAN":
      return {
        title: "整库扫描",
        tone: "tone-danger",
        summary: "始终扫描整个库目录，适合先跑通接入或无法稳定定位单文件时使用。",
        caution: "无法精确锁定本次下载文件，扫描范围也最大。"
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

export const INGEST_BLOCKERS = [
  "任务完成后无法入库",
  "订阅扫描无目标库",
  "Stash-Box 元数据无法获取"
];