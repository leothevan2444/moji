/**
 * Task-related utility functions and types.
 */

import type { TasksOverviewDocumentQuery } from "../graphql/generated/graphql";
import { deliveryModeLabel, transferActionLabel } from "./ingestUtils";

// ── Re-usable type aliases ──────────────────────────────────────────

export type DashboardTask = TasksOverviewDocumentQuery["tasks"][number];

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
  phase: "queued" | "downloading" | "transferRunning" | "scanPending" | "scanRunning" | "completed" | "failed";
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

const TASK_STAGE_SEQUENCE = [
  "sourcing",
  "downloading",
  "pending_ingest",
  "transferring",
  "scanning",
  "completed"
] as const;

type TaskStageKey = typeof TASK_STAGE_SEQUENCE[number];

// ── Simple helpers ──────────────────────────────────────────────────

export function normalizeStatus(value?: string | null) {
  return (value ?? "").trim().toLowerCase();
}

export function stageValue(task: DashboardTask) {
  return normalizeStatus(task.stage);
}

export function stageStatusValue(task: DashboardTask) {
  return normalizeStatus(task.stageStatus);
}

export function isStatus(task: DashboardTask, ...values: string[]) {
  const status = stageValue(task);
  return values.some((value) => status === value || status.includes(value));
}

export function isTaskActive(task: DashboardTask) {
  if (stageStatusValue(task) === "blocked" || stageValue(task) === "completed") return false;
  return ["sourcing", "downloading", "transferring", "scanning"].includes(stageValue(task))
    || Boolean(task.stashScanJobId);
}

export function isScanPending(task: DashboardTask) {
  return stageValue(task) === "pending_ingest"
    || (stageValue(task) === "scanning" && stageStatusValue(task) === "pending")
    || stageValue(task) === "transferring";
}

export function isMagnetLink(value?: string | null) {
  return Boolean(value && /^magnet:\?/i.test(value.trim()));
}

export function taskQueryLabel(task: DashboardTask) {
  if (isMagnetLink(task.torrentUrl)) {
    return "手动磁链任务";
  }
  return task.code || task.id;
}

export function isCopyableTaskValue(value?: string | null) {
  return Boolean(value && value.trim() !== "");
}

export function taskSummary(task: DashboardTask) {
  return task.torrentName || taskQueryLabel(task);
}

export function taskSourceLabel(source: string) {
  const normalized = source.trim().toUpperCase();
  if (normalized === "SEARCH") return "搜索";
  if (normalized === "SUBSCRIPTION") return "订阅";
  return "手动";
}

// ── Task grouping ───────────────────────────────────────────────────

export function taskGroup(task: DashboardTask): TaskGroupKey {
  if (stageStatusValue(task) === "blocked") return "需处理";
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
  return stageValue(task) === "pending_ingest" && !task.stashScanJobId;
}

// ── Task presentation helpers ───────────────────────────────────────

function simplifyMessage(message: string) {
  return message.replace(/^downloader:\s*/i, "").replace(/^stashsync:\s*/i, "").trim();
}

export function taskFailureSummary(task: DashboardTask): TaskFailureSummary {
  const transferError = simplifyMessage(task.transferError || "");
  const stashError = simplifyMessage(task.stashScanError || "");
  const taskError = simplifyMessage(task.stageErrorMessage || "");
  const qbtState = normalizeStatus(task.qbittorrentState || "");

  const transferFailure = describeTransferFailure(transferError);
  if (transferFailure) return transferFailure;

  const scanFailure = describeScanFailure(stashError);
  if (scanFailure) return scanFailure;

  const taskFailure = describeTaskFailure(taskError);
  if (taskFailure) return taskFailure;

  if (stageStatusValue(task) === "blocked") {
    return {
      title: "当前阶段受阻",
      detail: task.stageErrorMessage || "任务被阻塞，但没有更多错误上下文。",
      tone: "tone-danger"
    };
  }

  if (task.stashScanJobId || task.stashScanStartedAt) {
    return {
      title: "等待扫描收口",
      detail: task.stashScanHint || "下载已完成，等待 Stash 扫描继续推进。",
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

function describeTransferFailure(transferError: string): TaskFailureSummary | null {
  if (!transferError) return null;
  if (transferError.includes("resolve qB relative path failed")) {
    return {
      title: "qB 路径映射失败",
      detail: transferError,
      tone: "tone-danger"
    };
  }
  if (transferError.includes("build Moji transfer source path failed")) {
    return {
      title: "Moji 源路径构建失败",
      detail: transferError,
      tone: "tone-danger"
    };
  }
  if (transferError.includes("build Moji transfer target path failed")) {
    return {
      title: "Moji 目标路径构建失败",
      detail: transferError,
      tone: "tone-danger"
    };
  }
  return {
    title: "文件搬运失败",
    detail: transferError,
    tone: "tone-danger"
  };
}

function describeScanFailure(stashError: string): TaskFailureSummary | null {
  if (!stashError) return null;
  if (stashError.includes("build Stash scan path failed")) {
    return {
      title: "Stash 扫描路径构建失败",
      detail: stashError,
      tone: "tone-danger"
    };
  }
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

function describeTaskFailure(taskError: string): TaskFailureSummary | null {
  if (!taskError) return null;
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
  if (taskError.includes("duplicate torrent task")) {
    return {
      title: "重复种子任务",
      detail: "同一个 torrent 或 magnet 已存在对应的 Moji 任务。",
      tone: "tone-warn"
    };
  }
  if (taskError.includes("duplicate code task")) {
    return {
      title: "重复番号任务",
      detail: "同一个番号已经存在 Moji 任务，当前请求被严格去重拦截。",
      tone: "tone-warn"
    };
  }
  if (taskError.includes("task code is required")) {
    return {
      title: "无法提取番号",
      detail: "任务创建前必须解析出影片番号，但当前输入无法稳定提取 code。",
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

export function taskProgressPercent(task: DashboardTask) {
  if (stageValue(task) === "completed") return 100;
  return Math.max(0, Math.min(100, Math.round(task.progress * 100)));
}

export function taskPrimaryState(task: DashboardTask): Pick<TaskPresentation, "phase" | "label" | "tone"> {
  if (stageStatusValue(task) === "blocked") {
    return { phase: "failed", label: `${task.stageLabel}受阻`, tone: "tone-danger" as const };
  }

  if (stageValue(task) === "transferring") {
    return { phase: "transferRunning", label: "搬运中", tone: "tone-info" as const };
  }

  if (stageValue(task) === "scanning" && stageStatusValue(task) === "running") {
    return { phase: "scanRunning", label: "扫描中", tone: "tone-info" as const };
  }

  if (stageValue(task) === "pending_ingest" || (stageValue(task) === "scanning" && stageStatusValue(task) === "pending")) {
    return { phase: "scanPending", label: "待入库", tone: "tone-warn" as const };
  }

  if (stageValue(task) === "completed") {
    return { phase: "completed", label: "已完成", tone: "tone-success" as const };
  }

  if (stageValue(task) === "downloading" || isTaskActive(task)) {
    return { phase: "downloading", label: "下载中", tone: "tone-info" as const };
  }

  return { phase: "queued", label: "待下载", tone: "tone-neutral" as const };
}

export function taskMetaLine(task: DashboardTask) {
  const parts = [taskQueryLabel(task), `来源 ${taskSourceLabel(task.source)}`];
  if (task.torrentHash) {
    parts.push(`Hash ${task.torrentHash.slice(0, 8)}`);
  }
  if (task.category) {
    parts.push(task.category);
  }
  if (task.deliveryMode) {
    parts.push(deliveryModeLabel(task.deliveryMode));
  }
  return parts.filter(Boolean).join(" · ");
}

function taskStageIndex(task: DashboardTask) {
  const index = TASK_STAGE_SEQUENCE.indexOf(stageValue(task) as TaskStageKey);
  return index >= 0 ? index : 0;
}

function lifecycleLabel(stage: TaskStageKey) {
  if (stage === "sourcing") return "待选种";
  if (stage === "downloading") return "下载中";
  if (stage === "pending_ingest") return "待入库";
  if (stage === "transferring") return "搬运中";
  if (stage === "scanning") return "扫描中";
  return "已完成";
}

function lifecycleTime(task: DashboardTask, stage: TaskStageKey) {
  const currentIndex = taskStageIndex(task);
  const stageIndex = TASK_STAGE_SEQUENCE.indexOf(stage);

  if (stage === "sourcing") return task.createdAt;
  if (stage === "downloading") {
    return currentIndex >= stageIndex ? (task.downloadCompletedAt || task.updatedAt) : null;
  }
  if (stage === "pending_ingest") {
    return currentIndex >= stageIndex ? (task.downloadCompletedAt || task.updatedAt) : null;
  }
  if (stage === "transferring") {
    return currentIndex >= stageIndex ? task.updatedAt : null;
  }
  if (stage === "scanning") {
    return task.stashScanStartedAt || (currentIndex >= stageIndex ? task.updatedAt : null);
  }
  return stageValue(task) === "completed" ? task.updatedAt : null;
}

function lifecycleDetail(task: DashboardTask, stage: TaskStageKey, failure: TaskFailureSummary, progress: number) {
  const currentStage = stageValue(task);
  const currentStatus = stageStatusValue(task);
  const isCurrent = currentStage === stage;
  const isBlocked = isCurrent && currentStatus === "blocked";

  if (stage === "sourcing") {
    if (isBlocked) return failure.detail;
    if (isCurrent) return "Moji 已创建正式任务，正在搜索并筛选可用资源。";
    return "等待创建正式任务后开始搜索资源。";
  }

  if (stage === "downloading") {
    if (isBlocked) return failure.detail;
    if (currentStage === "sourcing") return "选种完成并提交到 qBittorrent 后进入此阶段。";
    if (isCurrent) {
      return task.qbittorrentState
        ? `${task.qbittorrentState} · ${progress}%`
        : `已提交到 qBittorrent，当前进度 ${progress}%。`;
    }
    if (task.contentPath) return `内容已落地：${task.contentPath}`;
    return "下载已完成。";
  }

  if (stage === "pending_ingest") {
    if (isBlocked) return failure.detail;
    if (currentStage === "sourcing" || currentStage === "downloading") return "qB 下载完成后，任务会进入待入库阶段。";
    if (isCurrent) return task.stashScanHint || "下载已完成，等待开始入库处理。";
    return task.stashScanHint || "下载已完成，已进入入库链路。";
  }

  if (stage === "transferring") {
    if (task.deliveryMode === "PATH_MAP") {
      return isCurrent ? "当前入库方式无需搬运文件，Moji 会直接进入扫描阶段。" : "当前入库方式无需搬运文件。";
    }
    if (isBlocked) return failure.detail;
    if (currentStage === "sourcing" || currentStage === "downloading" || currentStage === "pending_ingest") {
      return "需要搬运时，Moji 会在这里执行复制、移动或符号链接。";
    }
    if (isCurrent) {
      return task.mojiTransferPath
        ? `${transferActionLabel(task.transferAction ?? "") || "交付"} -> ${task.mojiTransferPath}`
        : "Moji 正在准备搬运目标路径。";
    }
    if (task.mojiTransferPath) return `搬运已完成：${task.mojiTransferPath}`;
    return "搬运已完成。";
  }

  if (stage === "scanning") {
    if (isBlocked) return failure.detail;
    if (currentStage === "sourcing" || currentStage === "downloading" || currentStage === "pending_ingest") {
      return "入库准备完成后，Moji 会触发 Stash 扫描。";
    }
    if (isCurrent) {
      return task.stashScanJobId
        ? `Stash job ${task.stashScanJobId} 正在执行。`
        : task.stashScanHint || "等待 Stash 接手当前扫描。";
    }
    if (task.stashScanPath) return `扫描路径：${task.stashScanPath}`;
    return "扫描已完成。";
  }

  if (isCurrent && currentStatus === "done") return task.stashScanPath || task.contentPath || "任务已完成闭环。";
  if (currentStage === "completed") return task.stashScanPath || task.contentPath || "任务已完成闭环。";
  return "扫描完成并收口后，任务会进入最终完成态。";
}

export function taskLifecycle(task: DashboardTask, failure: TaskFailureSummary): TaskLifecycleStep[] {
  const progress = taskProgressPercent(task);
  const currentStage = stageValue(task);
  const currentStatus = stageStatusValue(task);
  const currentIndex = taskStageIndex(task);

  return TASK_STAGE_SEQUENCE.map((stage) => {
    const stageIndex = TASK_STAGE_SEQUENCE.indexOf(stage);
    const isCurrent = currentStage === stage;

    let state: TaskLifecycleState = "upcoming";
    let tone: TaskLifecycleStep["tone"] = "tone-neutral";

    if (stageIndex < currentIndex) {
      state = "done";
      tone = "tone-success";
    } else if (isCurrent) {
      if (currentStatus === "blocked") {
        state = "error";
        tone = "tone-danger";
      } else if (stage === "completed" && currentStatus === "done") {
        state = "done";
        tone = "tone-success";
      } else {
        state = "current";
        tone = stage === "pending_ingest" ? "tone-warn" : "tone-info";
      }
    }

    return {
      key: stage,
      label: lifecycleLabel(stage),
      detail: lifecycleDetail(task, stage, failure, progress),
      state,
      tone,
      time: lifecycleTime(task, stage)
    };
  });
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
  } else if (primary.phase === "transferRunning") {
    summary = "下载已完成，Moji 正在准备文件搬运。";
    detail = task.mojiTransferPath ? `${transferActionLabel(task.transferAction ?? "") || "交付"} -> ${task.mojiTransferPath}` : "正在准备交付目标路径。";
  } else if (primary.phase === "scanRunning") {
    summary = "已完成下载，正在等待 Stash 收口。";
    detail = task.stashScanJobId ? `Stash job ${task.stashScanJobId} 正在执行。` : "Stash 已接手当前任务。";
  } else if (primary.phase === "scanPending") {
    summary = "下载已结束，等待触发或完成 Stash 扫描。";
    detail = task.stashScanHint || task.stageLabel || "当前尚未有扫描结果。";
  } else if (primary.phase === "completed") {
    summary = "下载与入库链路已完成。";
    detail = task.stashScanPath || task.contentPath || "任务已闭环。";
  } else if (primary.phase === "downloading") {
    summary = task.qbittorrentState ? `qBittorrent: ${task.qbittorrentState}` : "任务正在等待下载器推进。";
    detail = `当前进度 ${progress}%`;
  } else {
    summary = "任务已创建，正在搜寻资源。";
    detail = task.stageLabel || "当前阶段等待继续推进。";
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
