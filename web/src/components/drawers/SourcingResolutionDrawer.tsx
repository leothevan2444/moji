import { BlockedSourcingResolution } from "../tasks/BlockedSourcingResolution";
import { isWaitingForResource, type DashboardTask } from "../../utils";
import { useTranslation } from "react-i18next";

interface SourcingResolutionDrawerProps {
  task: DashboardTask | null;
  onResolved: () => void | Promise<void>;
}

export function SourcingResolutionDrawer({ task, onResolved }: SourcingResolutionDrawerProps) {
  const { t } = useTranslation();
  if (!task || task.stage !== "SOURCING" || (task.stageStatus !== "BLOCKED" && !isWaitingForResource(task))) {
    return (
      <article className="drawer-card">
        <h3>{t("taskUi.cannotResolve")}</h3>
        <p>{t("taskUi.cannotResolveDetail")}</p>
      </article>
    );
  }

  return (
    <div className="drawer-stack">
      <BlockedSourcingResolution key={task.id} task={task} onResolved={onResolved} />
    </div>
  );
}
