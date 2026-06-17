/**
 * Task-related utility functions and types.
 */

import type { DashboardDocumentQuery } from "../graphql/generated/graphql";

// ── Re-usable type aliases ──────────────────────────────────────────

export type DashboardTask = DashboardDocumentQuery["tasks"][number];

export type TaskGroupKey = "需处理" | "运行中" | "待入库" | "已完成";

export type TaskFailureSummary = {
  title: string;
  detail: string;
  tone: "tone-danger" | "tone-warn" | "tone-info" | "tone-neutral";
};

export type TaskLifecycleState = "done" | "current" | "error" | "upcoming";

export type TaskLifecycleStep = {
  key: string;
  label: string;
  detail: string;
  state: TaskLifecycleState;
  tone: "tone-success" | "tone-danger" | "tone-info" | "tone-warn" | "tone-neutral";
  time?: string | null;
};

export type TaskPresentation = {
  phase: "queued" | "downloading" | "scanPending" | "scanRunning" | "completed" | "failed";
  label: string;
  tone: "tone-success" | "tone-danger" | "tone-info" | "tone-warn" | "tone-neutral";
  summary: string;
  detail: string;
  progressPercent: number;
  progressLabel: string;
  metaLine: string;
  lifecycle: TaskLifecycleStep[];
};

/**
 * Visual state for a task card body. Driving the four exclusive branches
 * (error / pending / progress / completed) keeps the card render explicit
 * instead of relying on overlapping ad-hoc booleans.
 */
export type TaskCardState = "error" | "pending" | "progress" | "completed";

// ── Simple helpers ──────────────────────────────────────────────────

export function normalizeStatus(value: string) {
  return value.trim().toLowerCase();
}

export function isStatus(task: DashboardTask, ...values: string[]) {
  const status = normalizeStatus(task.status);
  return values.some((value) => status === value || status.includes(value));
}

export function isTaskActive(task: DashboardTask) {
  return !isStatus(task, "completed", "failed", "cancelled", "canceled", "paused");
}

export function isScanPending(task: DashboardTask) {
  const status = task.stashScanStatus.trim().toLowerCase();
  if (!status) return false;
  return !["completed", "done", "failed", "skipped", "idle"].includes(status);
}

export function isMagnetLink(value?: string | null) {
  return Boolean(value && /^magnet:\?/i.test(value.trim()));
}

export function taskQueryLabel(task: DashboardTask) {
  if (isMagnetLink(task.query)) {
    return "手动磁链任务";
  }
  return task.query || task.id;
}

export function isCopyableTaskValue(value?: string | null) {
  return Boolean(value && value.trim() !== "");
}

export function taskSummary(task: DashboardTask) {
  return task.torrentName || taskQueryLabel(task);
}

export function statusTone(status: string) {
  const normalized = status.toLowerCase();
  if (normalized.includes("complete")) return "tone-success";
  if (normalized.includes("fail")) return "tone-danger";
  if (normalized.includes("download") || normalized.includes("sync")) return "tone-info";
  if (normalized.includes("pending") || normalized.includes("wait")) return "tone-warn";
  return "tone-neutral";
}

// ── Task grouping ───────────────────────────────────────────────────

export function taskGroup(task: DashboardTask): TaskGroupKey {
  if (isStatus(task, "failed") || task.stashScanError) return "需处理";
  if (isTaskActive(task)) return "运行中";
  if (isScanPending(task)) return "待入库";
  return "已完成";
}

export function taskGroupTone(group: TaskGroupKey) {
  if (group === "需处理") return "tone-danger";
  if (group === "运行中") return "tone-info";
  if (group === "待入库") return "tone-warn";
  return "tone-success";
}

export function taskGroupDescription(group: TaskGroupKey) {
  if (group === "需处理") return "失败、扫描报错或需要人工回看的任务。";
  if (group === "运行中") return "仍在下载、同步或等待外部状态推进。";
  if (group === "待入库") return "下载已完成，但 Stash 扫描尚未收口。";
  return "流程已闭环的任务。";
}

// ── Task actions ────────────────────────────────────────────────────

export function canTriggerTaskStashScan(task: DashboardTask) {
  return task.status.trim().toLowerCase() === "completed" && task.stashScanStatus.trim().toLowerCase() !== "started";
}

// ── Task presentation helpers ───────────────────────────────────────

function simplifyMessage(message: string) {
  return message.replace(/^downloader:\s*/i, "").replace(/^stashsync:\s*/i, "").trim();
}

export function taskFailureSummary(task: DashboardTask): TaskFailureSummary {
  const stashError = simplifyMessage(task.stashScanError || "");
  const taskError = simplifyMessage(task.error || "");
  const status = normalizeStatus(task.status);
  const scanStatus = normalizeStatus(task.stashScanStatus || "");
  const qbtState = normalizeStatus(task.qbittorrentState || "");

  if (stashError) {
    if (stashError.includes("at least one scan path is required")) {
      return {
        title: "缺少扫描路径",
        detail: "任务没有可用于 Stash 扫描的内容路径或保存路径。",
        tone: "tone-danger"
      };
    }
    if (stashError.includes("not configured")) {
      return {
        title: "Stash 未配置",
        detail: "当前任务需要触发 Stash 扫描，但后端未启用对应连接。",
        tone: "tone-danger"
      };
    }
    return {
      title: "Stash 扫描失败",
      detail: stashError,
      tone: "tone-danger"
    };
  }

  if (taskError) {
    if (taskError.includes("no downloadable torrent candidate found")) {
      return {
        title: "没有可下载候选",
        detail: "搜索返回了结果，但没有可直接提交的 magnet 或种子链接。",
        tone: "tone-warn"
      };
    }
    if (taskError.includes("tracker is not configured")) {
      return {
        title: "索引器未配置",
        detail: "当前下载链路无法访问 Jackett 或其他搜索后端。",
        tone: "tone-danger"
      };
    }
    if (taskError.includes("torrent url is required")) {
      return {
        title: "缺少种子地址",
        detail: "手动添加任务时没有提供有效的磁链或下载地址。",
        tone: "tone-warn"
      };
    }
    if (taskError.includes("qBittorrent client is required") || taskError.includes("qBittorrent client is not configured")) {
      return {
        title: "下载器未启用",
        detail: "任务无法提交到 qBittorrent，需先补齐下载器配置。",
        tone: "tone-danger"
      };
    }
    if (taskError.includes("add torrent")) {
      return {
        title: "提交下载失败",
        detail: taskError,
        tone: "tone-danger"
      };
    }
    return {
      title: "任务执行失败",
      detail: taskError,
      tone: "tone-danger"
    };
  }

  if (status.includes("failed")) {
    return {
      title: "任务状态失败",
      detail: task.status || "任务被标记为失败，但没有更多错误上下文。",
      tone: "tone-danger"
    };
  }

  if (scanStatus) {
    return {
      title: "等待扫描收口",
      detail: task.stashScanStatus || "下载已完成，等待 Stash 扫描继续推进。",
      tone: "tone-warn"
    };
  }

  if (qbtState) {
    return {
      title: "下载进行中",
      detail: task.qbittorrentState || "任务仍在等待下载状态变化。",
      tone: "tone-info"
    };
  }

  return {
    title: "状态正常",
    detail: "当前任务没有显式错误，等待下一次同步。",
    tone: "tone-neutral"
  };
}

export function taskProgressPercent(task: DashboardTask) {
  if (isStatus(task, "completed")) return 100;
  return Math.max(0, Math.min(100, Math.round(task.progress * 100)));
}

export function taskPrimaryState(task: DashboardTask): Pick<TaskPresentation, "phase" | "label" | "tone"> {
  if (task.stashScanError || task.error || isStatus(task, "failed")) {
    if (task.stashScanError) {
      return { phase: "failed", label: "扫描失败", tone: "tone-danger" as const };
    }
    if (task.error) {
      return { phase: "failed", label: "下载失败", tone: "tone-danger" as const };
    }
    return { phase: "failed", label: "任务失败", tone: "tone-danger" as const };
  }

  const scanStatus = normalizeStatus(task.stashScanStatus || "");
  if (scanStatus === "started") {
    return { phase: "scanRunning", label: "扫描中", tone: "tone-info" as const };
  }

  if (isStatus(task, "completed")) {
    if (scanStatus || task.stashJobId) {
      return { phase: "scanPending", label: "待扫描", tone: "tone-warn" as const };
    }
    return { phase: "completed", label: "已完成", tone: "tone-success" as const };
  }

  if (task.qbittorrentState || isTaskActive(task)) {
    return { phase: "downloading", label: "下载中", tone: "tone-info" as const };
  }

  return { phase: "queued", label: "待下载", tone: "tone-neutral" as const };
}

export function taskMetaLine(task: DashboardTask) {
  const parts = [taskQueryLabel(task)];
  if (task.torrentHash) {
    parts.push(`Hash ${task.torrentHash.slice(0, 8)}`);
  }
  if (task.category) {
    parts.push(task.category);
  }
  return parts.filter(Boolean).join(" · ");
}

export function taskLifecycle(task: DashboardTask, failure: TaskFailureSummary): TaskLifecycleStep[] {
  const primary = taskPrimaryState(task);
  const progress = taskProgressPercent(task);
  const hasCompletedDownload = isStatus(task, "completed");
  const scanStarted = normalizeStatus(task.stashScanStatus || "") === "started";
  const hasScanFailure = Boolean(task.stashScanError);
  const hasFailure = Boolean(task.error) || isStatus(task, "failed");

  return [
    {
      key: "created",
      label: "已创建",
      detail: "Moji 已记录任务并等待后续处理。",
      state: "done",
      tone: "tone-success",
      time: task.createdAt
    },
    {
      key: "download",
      label: hasCompletedDownload ? "下载完成" : primary.phase === "failed" && hasFailure ? "下载失败" : "下载阶段",
      detail: hasCompletedDownload
        ? `内容已落地${task.contentPath ? `：${task.contentPath}` : "。"}`
        : task.qbittorrentState
          ? `${task.qbittorrentState} · ${progress}%`
          : primary.phase === "queued"
            ? "任务尚未进入下载器。"
            : failure.detail,
      state: primary.phase === "failed" && hasFailure ? "error" : hasCompletedDownload || primary.phase === "downloading" ? (hasCompletedDownload ? "done" : "current") : "upcoming",
      tone: primary.phase === "failed" && hasFailure ? "tone-danger" : primary.phase === "downloading" ? "tone-info" : hasCompletedDownload ? "tone-success" : "tone-neutral",
      time: hasCompletedDownload ? task.completedAt || task.updatedAt : task.updatedAt
    },
    {
      key: "scan",
      label: hasScanFailure ? "扫描失败" : scanStarted ? "扫描中" : hasCompletedDownload ? "入库阶段" : "等待扫描",
      detail: hasScanFailure
        ? simplifyMessage(task.stashScanError)
        : scanStarted
          ? `Stash job ${task.stashJobId || "已创建"} 正在运行。`
          : hasCompletedDownload
            ? task.stashJobId || task.stashScanStatus
              ? "下载已完成，等待 Stash 收口。"
              : "当前任务无需或尚未触发 Stash 扫描。"
            : "下载完成后才会进入此阶段。",
      state: hasScanFailure ? "error" : scanStarted ? "current" : hasCompletedDownload ? "current" : "upcoming",
      tone: hasScanFailure ? "tone-danger" : scanStarted ? "tone-info" : hasCompletedDownload ? "tone-warn" : "tone-neutral",
      time: task.updatedAt
    }
  ];
}

export function taskPresentation(task: DashboardTask): TaskPresentation {
  const primary = taskPrimaryState(task);
  const failure = taskFailureSummary(task);
  const progress = taskProgressPercent(task);

  let summary = "";
  let detail = "";

  if (primary.phase === "failed") {
    summary = failure.title;
    detail = failure.detail;
  } else if (primary.phase === "scanRunning") {
    summary = "已完成下载，正在等待 Stash 收口。";
    detail = task.stashJobId ? `Stash job ${task.stashJobId} 正在执行。` : "Stash 已接手当前任务。";
  } else if (primary.phase === "scanPending") {
    summary = "下载已结束，等待触发或完成 Stash 扫描。";
    detail = task.stashScanStatus || "当前尚未有扫描结果。";
  } else if (primary.phase === "completed") {
    summary = "下载与入库链路已完成。";
    detail = task.contentPath || "任务已闭环。";
  } else if (primary.phase === "downloading") {
    summary = task.qbittorrentState ? `qBittorrent: ${task.qbittorrentState}` : "任务正在等待下载器推进。";
    detail = `当前进度 ${progress}%`;
  } else {
    summary = "任务已创建，等待进入下载器。";
    detail = "下一次同步后会补齐下载状态。";
  }

  return {
    phase: primary.phase,
    label: primary.label,
    tone: primary.tone,
    summary,
    detail,
    progressPercent: progress,
    progressLabel: primary.phase === "completed" ? "已闭环" : `${progress}%`,
    metaLine: taskMetaLine(task),
    lifecycle: taskLifecycle(task, failure)
  };
}

/**
 * Collapse a task into one of four mutually-exclusive card states so the
 * card body has exactly one branch to render. Order matters: real failures
 * win over pending/progress, completed wins over pending.
 */
export function taskCardState(
  presentation: Pick<TaskPresentation, "phase">,
  failure: Pick<TaskFailureSummary, "tone">
): TaskCardState {
  if (presentation.phase === "failed" || failure.tone === "tone-danger") {
    return "error";
  }
  if (presentation.phase === "completed") {
    return "completed";
  }
  if (presentation.phase === "downloading" || presentation.phase === "scanRunning") {
    return "progress";
  }
  return "pending";
}
