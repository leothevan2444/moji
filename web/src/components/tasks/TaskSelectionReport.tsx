import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleCheck, faCircleMinus, faCircleQuestion } from "@fortawesome/free-solid-svg-icons";
import { useState } from "react";
import { createPortal } from "react-dom";
import { Drawer } from "../layout/Drawer";
import { useTranslation } from "react-i18next";
import { formatBytes, type DashboardTask } from "../../utils";

type Report = NonNullable<DashboardTask["selectionReport"]>;
type Evaluation = Report["evaluations"][number];

function CandidateExplanation({ evaluation }: { evaluation: Evaluation }) {
  const { t } = useTranslation();
  const statusLabel = t(`resourceSelection.statuses.${evaluation.status}`);
  const state = evaluation.status === "ACCEPTABLE" ? "accepted" : evaluation.status === "REJECTED" ? "rejected" : "unknown";
  const statusIcon = state === "accepted" ? faCircleCheck : state === "rejected" ? faCircleMinus : faCircleQuestion;
  const reasonText = (reason: string) => {
    if (reason === "TIER_MATCHED" && evaluation.tierName) return t("resourceSelection.tierReason", { name: evaluation.tierName });
    if (reason === "FALLBACK_ALLOWED") return t("resourceSelection.fallbackReason");
    if (reason === "EXCLUDED_RULE_MATCHED") return t("resourceSelection.excludedReason", { rules: evaluation.excludedRules.join(", ") || t("resourceSelection.exclusionUnrecorded") });
    return t(`resourceSelection.reasons.${reason}`, { defaultValue: reason });
  };
  return <article className={`task-selection-report__candidate${evaluation.selected ? " is-selected" : ""}`} aria-label={evaluation.selected ? `${evaluation.title} · ${t("resourceSelection.selectedWatermark")}` : evaluation.title}>
    {evaluation.selected ? <span className="task-selection-report__watermark" aria-hidden="true">{t("resourceSelection.selectedWatermark")}</span> : null}
    <div className="task-selection-report__candidate-head">
      <span className={`task-selection-report__status-icon is-${state}`} role="img" aria-label={statusLabel} title={statusLabel}>
        <FontAwesomeIcon icon={statusIcon} aria-hidden="true" />
      </span>
      <h5 className="task-selection-report__candidate-title">{evaluation.title || t("resourceSelection.unnamedResource")}</h5>
      {state === "unknown" ? <span className="task-selection-report__status-text">{statusLabel}</span> : null}
    </div>
    <div className="task-selection-report__candidate-meta">
      <dl className="task-selection-report__context">
        <div><dt>{t("resourceSelection.candidateSize")}</dt><dd>{evaluation.size == null ? t("resourceSelection.truth.UNKNOWN") : formatBytes(evaluation.size)}</dd></div>
        <div><dt>{t("resourceSelection.scopes.SEEDERS")}</dt><dd>{evaluation.seeders == null ? t("resourceSelection.truth.UNKNOWN") : evaluation.seeders}</dd></div>
      </dl>
      {evaluation.reasons.length ? <ul className="task-selection-report__reasons">{evaluation.reasons.map((reason, index) => <li key={index}>{reasonText(reason)}</li>)}</ul> : null}
    </div>
  </article>;
}

export function TaskSelectionReport({ report, taskCode }: { report: Report; taskCode: string }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const title = `${t("resourceSelection.report")} · ${taskCode}`;
  const end = report.end === "PREFERRED" && report.groupName === "" ? "BEST_AVAILABLE" : report.end;
  return <>
    <button type="button" className="ghost-button task-primary-action task-primary-action--secondary task-selection-report-entry" aria-haspopup="dialog" aria-expanded={open} onClick={() => setOpen(true)}>
      {t("resourceSelection.viewReport")}
    </button>
    {open ? createPortal(<Drawer visibleDrawer="task-selection-report" closing={false} title={title} initialFocus="panel" onClose={() => setOpen(false)} className="task-selection-report-dialog">
    <div className="task-selection-report task-selection-report__body">
      <dl className="task-selection-report__context"><div><dt>{t("resourceSelection.groupLabel")}</dt><dd>{report.groupName == null ? t("resourceSelection.unrecordedGroup") : report.groupName || t("resourceSelection.noGroupUsed")}</dd></div></dl>
      <p className="task-selection-report__result">{t(`resourceSelection.ends.${end}`)}</p>

      {report.partial ? <p className="task-selection-report__notice">{t("resourceSelection.partial")}</p> : null}
      {report.evaluations.length ? <div>
        <h4>{t("resourceSelection.evaluatedResources", { count: report.evaluations.length })}</h4>
        <ol className="task-selection-report__candidates">{report.evaluations.map((evaluation, index) => <li key={index}><CandidateExplanation evaluation={evaluation} /></li>)}</ol>
      </div> : null}
    </div>
    </Drawer>, document.body) : null}
  </>;
}
