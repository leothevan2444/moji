import type { PerformerActivityQuery } from "../../graphql/generated/graphql";

type Activity = PerformerActivityQuery["performerActivity"];
export type ActivityDisplayStatus = `${Activity["evaluationStatus"]}` | "LOADING" | "READ_FAILED" | "EVALUATING" | "WAITING_EVALUATION";

export function activityDisplayStatus(activity: Activity | null | undefined, readFailed: boolean, checkStatus?: string): ActivityDisplayStatus {
  if (!activity) return readFailed ? "READ_FAILED" : "LOADING";
  // A temporary read failure does not erase an already confirmed rating.
  if (activity.value != null && activity.rank != null) return "READY";
  if (!activity.observedAt) {
    if (checkStatus === "RUNNING") return "EVALUATING";
    if (checkStatus === "PENDING") return "WAITING_EVALUATION";
  }
  return activity.evaluationStatus === "READY" ? "INCOMPLETE_CATALOG" : activity.evaluationStatus;
}

// A zero activity value is a valid level 1, not an unavailable rating.
export function activityRankForDisplay(activity: Activity | null | undefined): number {
  if (activity?.value == null || activity.rank == null || !Number.isInteger(activity.rank) || activity.rank < 1 || activity.rank > 5) return 0;
  return activity.rank;
}
