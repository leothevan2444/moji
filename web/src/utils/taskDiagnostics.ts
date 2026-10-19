import type { DashboardTask } from "./taskUtils";

// Keep operational data available without copying signed resource URLs or credentials.
export function taskDiagnostics(task: DashboardTask) {
  return JSON.stringify({
    id: task.id, code: task.code, source: task.source,
    stage: task.stage, stageStatus: task.stageStatus, stageErrorCode: task.stageErrorCode,
    stageErrorMessage: task.stageErrorMessage,
    category: task.category, tags: task.tags, qbittorrentState: task.qbittorrentState,
    progress: task.progress, torrentHash: task.torrentHash,
    savePath: task.savePath, contentPath: task.contentPath,
    mojiSourcePath: task.mojiSourcePath, transferAction: task.transferAction,
    mojiTransferPath: task.mojiTransferPath, transferError: task.transferError,
    stashIngestPhase: task.stashIngestPhase, stashScanPath: task.stashScanPath,
    stashScanJobId: task.stashScanJobId, stashGenerateJobId: task.stashGenerateJobId,
    stashIdentifyJobId: task.stashIdentifyJobId, stashSceneIds: task.stashSceneIds,
    stashScanHint: task.stashScanHint, stashScanError: task.stashScanError,
    stashScanStartedAt: task.stashScanStartedAt, files: task.files,
    createdAt: task.createdAt, updatedAt: task.updatedAt, lastProgressAt: task.lastProgressAt,
    sourcingCompletedAt: task.sourcingCompletedAt, downloadCompletedAt: task.downloadCompletedAt,
    ingestCompletedAt: task.ingestCompletedAt
  }, null, 2);
}
