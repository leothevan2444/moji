import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleCheck } from "@fortawesome/free-solid-svg-icons/faCircleCheck";
import { faCircleExclamation } from "@fortawesome/free-solid-svg-icons/faCircleExclamation";
import { faForward } from "@fortawesome/free-solid-svg-icons/faForward";
import { useTranslation } from "react-i18next";
import type { TaskBatchStatus } from "../../graphql/generated/graphql";

export interface TaskBatchResultView {
  batchId: string;
  summary: { requestedCount: number; succeededCount: number; skippedCount: number; failedCount: number };
  results: Array<{ taskId: string; status: TaskBatchStatus; reasonCode: string; task?: { id: string; code?: string | null } | null }>;
}

export function TaskBatchResultDrawer({ payload, onOpenTask }: { payload: TaskBatchResultView; onOpenTask: (id: string) => void }) {
  const { t, i18n } = useTranslation();
  const reason = (code: string) => i18n.exists(`taskBatch.reasons.${code}`) ? t(`taskBatch.reasons.${code}`) : t("taskBatch.reasons.UNKNOWN");
  return <div className="drawer-stack">
    <article className="drawer-card">
      <div className="drawer-card__head"><div><h3>{t("taskBatch.resultTitle")}</h3><p>{t("taskBatch.batchId", { id: payload.batchId })}</p></div></div>
      <dl className="settings-grid task-batch-summary">
        <div><dt>{t("taskBatch.requested")}</dt><dd>{payload.summary.requestedCount}</dd></div>
        <div><dt>{t("taskBatch.succeeded")}</dt><dd>{payload.summary.succeededCount}</dd></div>
        <div><dt>{t("taskBatch.skipped")}</dt><dd>{payload.summary.skippedCount}</dd></div>
        <div><dt>{t("taskBatch.failed")}</dt><dd>{payload.summary.failedCount}</dd></div>
      </dl>
    </article>
    <article className="drawer-card task-batch-results">
      {payload.results.map((item) => <button type="button" key={item.taskId} className="task-batch-result" disabled={!item.task} onClick={() => item.task && onOpenTask(item.task.id)}>
        <FontAwesomeIcon icon={item.status === "SUCCEEDED" ? faCircleCheck : item.status === "SKIPPED" ? faForward : faCircleExclamation} />
        <span><strong>{item.task?.code || item.taskId}</strong><small>{reason(item.reasonCode)}</small></span>
      </button>)}
    </article>
  </div>;
}
