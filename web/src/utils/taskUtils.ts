/**
 * Task-related utility functions and types.
 */

import type { TasksOverviewDocumentQuery } from "../graphql/generated/graphql";
import { deliveryModeLabel, transferActionLabel } from "./ingestUtils";
import i18n from "../i18n/i18n";

const tr = (key: string, options?: Record<string, unknown>) => i18n.t(key, options);

// ── Re-usable type aliases ──────────────────────────────────────────

export type DashboardTask = TasksOverviewDocumentQuery["tasks"][number];

export type TaskGroupKey = "attention" | "active" | "ingestPending" | "completed";

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
    return tr("taskModel.manualMagnet");
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
  if (normalized === "SEARCH") return tr("taskModel.sources.search");
  if (normalized === "SUBSCRIPTION") return tr("taskModel.sources.subscription");
  return tr("taskModel.sources.manual");
}

// ── Task grouping ───────────────────────────────────────────────────

export function taskGroup(task: DashboardTask): TaskGroupKey {
  if (stageStatusValue(task) === "blocked") return "attention";
  if (isTaskActive(task)) return "active";
  if (isScanPending(task)) return "ingestPending";
  return "completed";
}

export function taskGroupTone(group: TaskGroupKey) {
  if (group === "attention") return "tone-danger";
  if (group === "active") return "tone-info";
  if (group === "ingestPending") return "tone-warn";
  return "tone-success";
}


// ── Task actions ────────────────────────────────────────────────────

export function canTriggerTaskStashScan(task: DashboardTask) {
  return stageValue(task) === "pending_ingest" && !task.stashScanJobId;
}

export function taskBatchEligibility(tasks: DashboardTask[]) {
  return {
    retryIds: tasks.filter((task) => task.stageStatus === "BLOCKED" && task.stage !== "COMPLETED").map((task) => task.id),
    ingestIds: tasks.filter((task) => ["PENDING_INGEST", "TRANSFERRING", "SCANNING"].includes(task.stage) && !(task.stage === "SCANNING" && task.stageStatus === "RUNNING")).map((task) => task.id)
  };
}

export function mergeTaskSelection(current: string[], added: string[], limit = 100) {
  return [...new Set([...current, ...added])].slice(0, limit);
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
      title: tr("taskModel.failure.blocked"),
      detail: task.stageErrorMessage || tr("taskModel.failure.blockedDetail"),
      tone: "tone-danger"
    };
  }

  if (task.stashScanJobId || task.stashScanStartedAt) {
    return {
      title: tr("taskModel.failure.scanPending"),
      detail: task.stashScanHint || tr("taskModel.failure.scanPendingDetail"),
      tone: "tone-warn"
    };
  }

  if (qbtState) {
    return {
      title: tr("taskModel.failure.downloading"),
      detail: task.qbittorrentState || tr("taskModel.failure.downloadingDetail"),
      tone: "tone-info"
    };
  }

  return {
    title: tr("taskModel.failure.healthy"),
    detail: tr("taskModel.failure.healthyDetail"),
    tone: "tone-neutral"
  };
}

function describeTransferFailure(transferError: string): TaskFailureSummary | null {
  if (!transferError) return null;
  if (transferError.includes("resolve qB relative path failed")) {
    return {
      title: tr("taskModel.failure.qbPath"),
      detail: transferError,
      tone: "tone-danger"
    };
  }
  if (transferError.includes("build Moji transfer source path failed")) {
    return {
      title: tr("taskModel.failure.sourcePath"),
      detail: transferError,
      tone: "tone-danger"
    };
  }
  if (transferError.includes("build Moji transfer target path failed")) {
    return {
      title: tr("taskModel.failure.targetPath"),
      detail: transferError,
      tone: "tone-danger"
    };
  }
  return {
    title: tr("taskModel.failure.transfer"),
    detail: transferError,
    tone: "tone-danger"
  };
}

function describeScanFailure(stashError: string): TaskFailureSummary | null {
  if (!stashError) return null;
  if (stashError.includes("build Stash scan path failed")) {
    return {
      title: tr("taskModel.failure.scanPath"),
      detail: stashError,
      tone: "tone-danger"
    };
  }
  if (stashError.includes("at least one scan path is required")) {
    return {
      title: tr("taskModel.failure.missingScanPath"),
      detail: tr("taskModel.failure.missingScanPathDetail"),
      tone: "tone-danger"
    };
  }
  if (stashError.includes("not configured")) {
    return {
      title: tr("taskModel.failure.stashMissing"),
      detail: tr("taskModel.failure.stashMissingDetail"),
      tone: "tone-danger"
    };
  }
  return {
    title: tr("taskModel.failure.scan"),
    detail: stashError,
    tone: "tone-danger"
  };
}

function describeTaskFailure(taskError: string): TaskFailureSummary | null {
  if (!taskError) return null;
  if (taskError.includes("no downloadable torrent candidate found")) {
    return {
      title: tr("taskModel.failure.noCandidate"),
      detail: tr("taskModel.failure.noCandidateDetail"),
      tone: "tone-warn"
    };
  }
  if (taskError.includes("tracker is not configured")) {
    return {
      title: tr("taskModel.failure.trackerMissing"),
      detail: tr("taskModel.failure.trackerMissingDetail"),
      tone: "tone-danger"
    };
  }
  if (taskError.includes("torrent url is required")) {
    return {
      title: tr("taskModel.failure.torrentMissing"),
      detail: tr("taskModel.failure.torrentMissingDetail"),
      tone: "tone-warn"
    };
  }
  if (taskError.includes("duplicate torrent task")) {
    return {
      title: tr("taskModel.failure.duplicateTorrent"),
      detail: tr("taskModel.failure.duplicateTorrentDetail"),
      tone: "tone-warn"
    };
  }
  if (taskError.includes("duplicate code task")) {
    return {
      title: tr("taskModel.failure.duplicateCode"),
      detail: tr("taskModel.failure.duplicateCodeDetail"),
      tone: "tone-warn"
    };
  }
  if (taskError.includes("task code is required")) {
    return {
      title: tr("taskModel.failure.codeMissing"),
      detail: tr("taskModel.failure.codeMissingDetail"),
      tone: "tone-warn"
    };
  }
  if (taskError.includes("qBittorrent client is required") || taskError.includes("qBittorrent client is not configured")) {
    return {
      title: tr("taskModel.failure.downloaderMissing"),
      detail: tr("taskModel.failure.downloaderMissingDetail"),
      tone: "tone-danger"
    };
  }
  if (taskError.includes("add torrent")) {
    return {
      title: tr("taskModel.failure.submit"),
      detail: taskError,
      tone: "tone-danger"
    };
  }
  return {
    title: tr("taskModel.failure.generic"),
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
    return { phase: "failed", label: tr("taskModel.states.blocked"), tone: "tone-danger" as const };
  }

  if (stageValue(task) === "transferring") {
    return { phase: "transferRunning", label: tr("taskModel.states.transferring"), tone: "tone-info" as const };
  }

  if (stageValue(task) === "scanning" && stageStatusValue(task) === "running") {
    return { phase: "scanRunning", label: tr("taskModel.states.scanning"), tone: "tone-info" as const };
  }

  if (stageValue(task) === "pending_ingest" || (stageValue(task) === "scanning" && stageStatusValue(task) === "pending")) {
    return { phase: "scanPending", label: tr("taskModel.states.pendingIngest"), tone: "tone-warn" as const };
  }

  if (stageValue(task) === "completed") {
    return { phase: "completed", label: tr("taskModel.states.completed"), tone: "tone-success" as const };
  }

  if (stageValue(task) === "downloading" || isTaskActive(task)) {
    return { phase: "downloading", label: tr("taskModel.states.downloading"), tone: "tone-info" as const };
  }

  return { phase: "queued", label: tr("taskModel.states.queued"), tone: "tone-neutral" as const };
}

export function taskMetaLine(task: DashboardTask) {
  const parts = [taskQueryLabel(task), tr("taskModel.source", { source: taskSourceLabel(task.source) })];
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
  return tr(`taskModel.lifecycle.${stage}`);
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
    if (isCurrent) return tr("taskModel.lifecycle.sourcingCurrent");
    return tr("taskModel.lifecycle.sourcingUpcoming");
  }

  if (stage === "downloading") {
    if (isBlocked) return failure.detail;
    if (currentStage === "sourcing") return tr("taskModel.lifecycle.downloadUpcoming");
    if (isCurrent) {
      return task.qbittorrentState
        ? `${task.qbittorrentState} · ${progress}%`
        : tr("taskModel.lifecycle.downloadProgress", { progress });
    }
    if (task.contentPath) return tr("taskModel.lifecycle.landed", { path: task.contentPath });
    return tr("taskModel.lifecycle.downloadDone");
  }

  if (stage === "pending_ingest") {
    if (isBlocked) return failure.detail;
    if (currentStage === "sourcing" || currentStage === "downloading") return tr("taskModel.lifecycle.ingestUpcoming");
    if (isCurrent) return task.stashScanHint || tr("taskModel.lifecycle.ingestCurrent");
    return task.stashScanHint || tr("taskModel.lifecycle.ingestDone");
  }

  if (stage === "transferring") {
    if (task.deliveryMode === "PATH_MAP") {
      return tr(isCurrent ? "taskModel.lifecycle.noTransferCurrent" : "taskModel.lifecycle.noTransfer");
    }
    if (isBlocked) return failure.detail;
    if (currentStage === "sourcing" || currentStage === "downloading" || currentStage === "pending_ingest") {
      return tr("taskModel.lifecycle.transferUpcoming");
    }
    if (isCurrent) {
      return task.mojiTransferPath
        ? `${transferActionLabel(task.transferAction ?? "") || tr("taskModel.lifecycle.delivery")} -> ${task.mojiTransferPath}`
        : tr("taskModel.lifecycle.preparingTarget");
    }
    if (task.mojiTransferPath) return tr("taskModel.lifecycle.transferDonePath", { path: task.mojiTransferPath });
    return tr("taskModel.lifecycle.transferDone");
  }

  if (stage === "scanning") {
    if (isBlocked) return failure.detail;
    if (currentStage === "sourcing" || currentStage === "downloading" || currentStage === "pending_ingest") {
      return tr("taskModel.lifecycle.scanUpcoming");
    }
    if (isCurrent) {
      return task.stashScanJobId
        ? tr("taskModel.lifecycle.jobRunning", { job: task.stashScanJobId })
        : task.stashScanHint || tr("taskModel.lifecycle.scanWaiting");
    }
    if (task.stashScanPath) return tr("taskModel.lifecycle.scanPath", { path: task.stashScanPath });
    return tr("taskModel.lifecycle.scanDone");
  }

  if (isCurrent && currentStatus === "done") return task.stashScanPath || task.contentPath || tr("taskModel.lifecycle.closed");
  if (currentStage === "completed") return task.stashScanPath || task.contentPath || tr("taskModel.lifecycle.closed");
  return tr("taskModel.lifecycle.completeUpcoming");
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
    summary = tr("taskModel.presentation.transfer");
    detail = task.mojiTransferPath ? `${transferActionLabel(task.transferAction ?? "") || tr("taskModel.lifecycle.delivery")} -> ${task.mojiTransferPath}` : tr("taskModel.presentation.target");
  } else if (primary.phase === "scanRunning") {
    summary = tr("taskModel.presentation.scan");
    detail = task.stashScanJobId ? tr("taskModel.lifecycle.jobRunning", { job: task.stashScanJobId }) : tr("taskModel.presentation.stashAccepted");
  } else if (primary.phase === "scanPending") {
    summary = tr("taskModel.presentation.pending");
    detail = task.stashScanHint || tr("taskModel.presentation.noScan");
  } else if (primary.phase === "completed") {
    summary = tr("taskModel.presentation.completed");
    detail = task.stashScanPath || task.contentPath || tr("taskModel.presentation.closed");
  } else if (primary.phase === "downloading") {
    summary = task.qbittorrentState ? `qBittorrent: ${task.qbittorrentState}` : tr("taskModel.presentation.downloaderWaiting");
    detail = tr("taskModel.presentation.progress", { progress });
  } else {
    summary = tr("taskModel.presentation.queued");
    detail = tr("taskModel.presentation.stageWaiting");
  }

  return {
    phase: primary.phase,
    label: primary.label,
    tone: primary.tone,
    summary,
    detail,
    progressPercent: progress,
    progressLabel: primary.phase === "completed" ? tr("taskModel.states.closed") : `${progress}%`,
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
