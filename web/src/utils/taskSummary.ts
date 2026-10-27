import { TaskListSummaryFragmentDoc, type TaskListSummaryFragment } from "../graphql/generated/graphql";
import { useFragment, type FragmentType } from "../graphql/generated/fragment-masking";
import type { DashboardTask } from "./taskUtils";

export function taskSummaryToDashboard(value: FragmentType<typeof TaskListSummaryFragmentDoc> | TaskListSummaryFragment): DashboardTask {
  const summary = useFragment(TaskListSummaryFragmentDoc, value as FragmentType<typeof TaskListSummaryFragmentDoc>);
  return { files: [], torrentUrl: "", contentPath: null, torrentHash: null, savePath: null, category: null, tags: null,
    mojiSourcePath: null, transferAction: null, mojiTransferPath: null, transferError: null,
    stashGenerateJobId: null, stashIdentifyJobId: null, stashScanJobId: null,
    stashScanPath: null, stashScanError: null, stashScanHint: null, stashScanStartedAt: null,
    ...summary, stashSceneIds: summary.stashSceneId ? [summary.stashSceneId] : [], __typename: "Task" } as DashboardTask;
}
