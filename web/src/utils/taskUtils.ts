/**
 * Task-related utility functions and types.
 */

import type { TasksOverviewDocumentQuery } from "../graphql/generated/graphql";
import { transferActionLabel } from "./ingestUtils";
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
  showDownloadProgress: boolean;
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
  "generating",
  "identifying",
  "verifying",
  "completed"
] as const;

type TaskStageKey = typeof TASK_STAGE_SEQUENCE[number];

export function stashIngestLabel(task: DashboardTask) {
  const phase = task.stashIngestPhase;
  return tr(`taskModel.ingest.${phase === "GENERATE" ? "generate" : phase === "IDENTIFY" ? "identify" : phase === "VERIFY" ? "verify" : "scan"}`);
}

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
  if (task.code?.trim()) return task.code.trim();
  if (isMagnetLink(task.torrentUrl)) {
    return tr("taskModel.manualMagnet");
  }
  return task.id;
}

export function isCopyableTaskValue(value?: string | null) {
  return Boolean(value && value.trim() !== "");
}

export function taskSummary(task: DashboardTask) {
  return task.code?.trim() || task.torrentName?.trim() || taskQueryLabel(task);
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

export function needsTaskFileSelection(task: DashboardTask) {
  return task.stage === "DOWNLOADING" && task.stageStatus === "BLOCKED"
    && (task.stageErrorCode === "FILE_SELECTION_REQUIRED"
      || (!task.stageErrorCode && Boolean(task.files?.some(file => file.kind === "UNKNOWN_VIDEO" && file.reason === "IDENTITY_UNCERTAIN"))));
}

export function isWaitingForResource(task: DashboardTask) {
  return task.stage === "SOURCING" && task.stageStatus === "PENDING" && Boolean(task.sourcingNextCheckAt);
}

export function taskPrimaryAction(task: DashboardTask): "resolve" | "files" | "retry" | "ingest" | null {
  if (isWaitingForResource(task)) return "retry";
  if (task.stageStatus === "BLOCKED") {
    if (task.stage === "SOURCING") return "resolve";
    if (needsTaskFileSelection(task)) return "files";
    return task.stage === "COMPLETED" ? null : "retry";
  }
  return canTriggerTaskStashScan(task) ? "ingest" : null;
}

export function taskBatchEligibility(tasks: DashboardTask[]) {
  return {
    retryIds: tasks.filter((task) => (task.stageStatus === "BLOCKED" || isWaitingForResource(task)) && task.stage !== "COMPLETED").map((task) => task.id),
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
  const sourcingFailures: Record<string, string> = {
    CANDIDATE_CODE_MISMATCH: "candidateMismatch",
    CANDIDATE_IDENTITY_UNVERIFIED: "candidateUnverified",
    TORRENT_METADATA_FAILED: "torrentMetadata",
  };
  const sourcingFailure = sourcingFailures[task.stageErrorCode || ""];
  if (sourcingFailure) {
    return { title: tr(`taskModel.failure.${sourcingFailure}`), detail: tr(`taskModel.failure.${sourcingFailure}Detail`), tone: "tone-warn" };
  }
  if (task.stageErrorCode === "FILE_SELECTION_REQUIRED") return { title: tr("mediaFiles.blocked"), detail: tr("mediaFiles.blockedDetail"), tone: "tone-warn" };
  if (["MEDIA_PREPARATION_FAILED", "FILE_SELECTION_FAILED"].includes(task.stageErrorCode || "")) return { title: tr("mediaFiles.prepareFailed"), detail: tr("mediaFiles.prepareFailedDetail"), tone: "tone-danger" };
  const transferError = simplifyMessage(task.transferError || "");
  const stashError = simplifyMessage(task.stashScanError || "");
  const taskError = simplifyMessage(task.stageErrorMessage || "");
  const qbtState = normalizeStatus(task.qbittorrentState || "");

  const ingestErrors: Record<string, string> = {
    STASH_SCENE_MISSING: "sceneMissing",
    STASH_SCENE_CHANGED: "sceneChanged",
    STASH_IDENTIFY_SOURCES_MISSING: "sourcesMissing",
  };
  const ingestError = ingestErrors[task.stageErrorCode || ""];
  if (ingestError) {
    return { title: stashIngestLabel(task), detail: tr(`taskModel.ingest.${ingestError}`), tone: "tone-danger" };
  }
  if (["STASH_GENERATE_FAILED", "STASH_IDENTIFY_FAILED", "STASH_INGEST_INCOMPLETE"].includes(task.stageErrorCode || "")) {
    return { title: stashIngestLabel(task), detail: tr(task.stageErrorCode === "STASH_INGEST_INCOMPLETE" ? "taskModel.ingest.incomplete" : "taskModel.ingest.failed"), tone: "tone-danger" };
  }

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
  if (isWaitingForResource(task)) return { phase: "queued", label: tr("resourceTracking.waiting"), tone: "tone-neutral" };
  if (stageStatusValue(task) === "blocked") {
    return { phase: "failed", label: tr("taskModel.states.blocked"), tone: "tone-danger" as const };
  }

  if (stageValue(task) === "sourcing") {
    const pending = stageStatusValue(task) === "pending";
    return { phase: "queued", label: tr(pending ? "taskQueue.pending" : "taskQueue.running"), tone: pending ? "tone-neutral" : "tone-info" };
  }

  if (stageValue(task) === "transferring") {
    return { phase: "transferRunning", label: tr("taskModel.states.transferring"), tone: "tone-info" as const };
  }

  if (stageValue(task) === "scanning" && stageStatusValue(task) === "running") {
    return { phase: "scanRunning", label: stashIngestLabel(task), tone: "tone-info" as const };
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
  return task.category?.trim() || "";
}

function lifecycleStageValue(task: DashboardTask): TaskStageKey {
  if (stageValue(task) === "scanning") {
    if (task.stashIngestPhase === "GENERATE") return "generating";
    if (task.stashIngestPhase === "IDENTIFY") return "identifying";
    if (task.stashIngestPhase === "VERIFY") return "verifying";
  }
  return stageValue(task) as TaskStageKey;
}

function taskStageIndex(task: DashboardTask) {
  const index = TASK_STAGE_SEQUENCE.indexOf(lifecycleStageValue(task));
  return index >= 0 ? index : 0;
}

function lifecycleLabel(stage: TaskStageKey) {
  return tr(`taskModel.lifecycle.${stage}`);
}

function lifecycleTime(task: DashboardTask, stage: TaskStageKey) {
  // Only show timestamps explicitly recorded for this milestone.
  if (stage === "sourcing") return task.sourcingCompletedAt;
  if (stage === "downloading") return task.downloadCompletedAt;
  if (stage === "completed") return task.ingestCompletedAt;
  if (stage === "scanning") return task.stashScanStartedAt;
  return null;
}

function lifecycleDetail(task: DashboardTask, stage: TaskStageKey, failure: TaskFailureSummary, progress: number) {
  const currentStage = lifecycleStageValue(task);
  const currentStatus = stageStatusValue(task);
  const isCurrent = currentStage === stage;
  const isBlocked = isCurrent && currentStatus === "blocked";

  if (stage === "sourcing") {
    if (isCurrent && isWaitingForResource(task)) return tr("resourceTracking.waitingDetail");
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
      return tr("taskFlow.running.scanning");
    }
    if (task.stashScanPath) return tr("taskModel.lifecycle.scanPath", { path: task.stashScanPath });
    return tr("taskModel.lifecycle.scanDone");
  }

  if (stage === "generating" || stage === "identifying" || stage === "verifying") {
    if (isBlocked) return failure.detail;
    if (isCurrent) {
      return tr(`taskFlow.running.${stage}`);
    }
    return tr(taskStageIndex(task) > TASK_STAGE_SEQUENCE.indexOf(stage) ? "taskModel.ingest.stepDone" : "taskModel.ingest.stepUpcoming");
  }

  if (isCurrent && currentStatus === "done") return task.stashScanPath || task.contentPath || tr("taskModel.lifecycle.closed");
  if (currentStage === "completed") return task.stashScanPath || task.contentPath || tr("taskModel.lifecycle.closed");
  return tr("taskModel.lifecycle.completeUpcoming");
}

export function taskLifecycle(task: DashboardTask, failure: TaskFailureSummary): TaskLifecycleStep[] {
  const progress = taskProgressPercent(task);
  const currentStage = lifecycleStageValue(task);
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
      if (stage === "sourcing" && currentStatus === "pending") {
        state = "upcoming";
      } else if (currentStatus === "blocked") {
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
    summary = stashIngestLabel(task);
    detail = lifecycleDetail(task, lifecycleStageValue(task), failure, progress);
  } else if (primary.phase === "scanPending") {
    summary = tr("taskModel.presentation.pending");
    detail = task.stashScanHint || tr("taskModel.presentation.noScan");
  } else if (primary.phase === "completed") {
    summary = tr("taskModel.presentation.completed");
    detail = task.stashScanPath || task.contentPath || tr("taskModel.presentation.closed");
  } else if (primary.phase === "downloading") {
    summary = tr("taskModel.states.downloading");
    detail = tr("taskExperience.afterDownload");
  } else {
    summary = stageValue(task) === "sourcing" ? primary.label : tr("taskModel.presentation.queued");
    detail = isWaitingForResource(task) ? tr("resourceTracking.waitingDetail") : stageValue(task) === "sourcing" && stageStatusValue(task) === "pending" ? tr("taskQueue.waiting") : tr("taskModel.presentation.stageWaiting");
  }

  const showDownloadProgress = stageValue(task) === "downloading";
  return {
    phase: primary.phase,
    label: primary.label,
    tone: primary.tone,
    summary,
    detail,
    progressPercent: progress,
    showDownloadProgress,
    progressLabel: showDownloadProgress
      ? tr("taskExperience.downloadProgress", { progress })
      : primary.label,
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
