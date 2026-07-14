import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleCheck } from "@fortawesome/free-solid-svg-icons/faCircleCheck";
import { faCircleExclamation } from "@fortawesome/free-solid-svg-icons/faCircleExclamation";
import { faForward } from "@fortawesome/free-solid-svg-icons/faForward";
import { useTranslation } from "react-i18next";
import type { PerformerBatchStatus } from "../../graphql/generated/graphql";

export interface PerformerBatchResultView {
  batchId: string;
  summary: { requestedCount: number; succeededCount: number; skippedCount: number; failedCount: number };
  results: Array<{ performerId: string; status: PerformerBatchStatus; reasonCode: string; performer?: { id: string; name: string } | null }>;
}

export function PerformerBatchResultDrawer({ payload }: { payload: PerformerBatchResultView }) {
  const { t, i18n } = useTranslation();
  const reason = (code: string) => i18n.exists(`performerBatch.reasons.${code}`) ? t(`performerBatch.reasons.${code}`) : t("performerBatch.reasons.UNKNOWN");
  return <div className="drawer-stack">
    <article className="drawer-card">
      <div className="drawer-card__head"><div><h3>{t("performerBatch.resultTitle")}</h3><p>{t("performerBatch.batchId", { id: payload.batchId })}</p></div></div>
      <dl className="settings-grid task-batch-summary">
        <div><dt>{t("performerBatch.requested")}</dt><dd>{payload.summary.requestedCount}</dd></div>
        <div><dt>{t("performerBatch.succeeded")}</dt><dd>{payload.summary.succeededCount}</dd></div>
        <div><dt>{t("performerBatch.skipped")}</dt><dd>{payload.summary.skippedCount}</dd></div>
        <div><dt>{t("performerBatch.failed")}</dt><dd>{payload.summary.failedCount}</dd></div>
      </dl>
    </article>
    <article className="drawer-card task-batch-results">
      {payload.results.map((item) => <div key={item.performerId} className="task-batch-result">
        <FontAwesomeIcon icon={item.status === "SUCCEEDED" ? faCircleCheck : item.status === "SKIPPED" ? faForward : faCircleExclamation} />
        <span><strong>{item.performer?.name || item.performerId}</strong><small>{reason(item.reasonCode)}</small></span>
      </div>)}
    </article>
  </div>;
}
