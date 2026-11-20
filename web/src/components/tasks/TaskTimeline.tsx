import type { ReactNode } from "react";
import { useAppTimezone } from "../../hooks/useAppTimezone";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleCheck } from "@fortawesome/free-solid-svg-icons/faCircleCheck";
import { faClock } from "@fortawesome/free-solid-svg-icons/faClock";
import { faTriangleExclamation } from "@fortawesome/free-solid-svg-icons/faTriangleExclamation";
import { useTranslation } from "react-i18next";
import { formatDateTime, transferActionLabel, type TaskLifecycleStep, type TaskPresentation } from "../../utils";

import { TaskValue } from "./TaskValue";

interface StageInformation {
  status?: Pick<TaskPresentation, "label" | "tone" | "summary" | "detail">;
  animateIngest?: boolean;
  transfer?: { action?: string | null; target?: string | null };
  onCopy?: (value: string, message: string) => void | Promise<void>;
}

const isActive = (step: TaskLifecycleStep) => step.state === "current" || step.state === "error";

function StepIcon({ step, position }: { step: TaskLifecycleStep; position: number }) {
  return <span aria-hidden="true">{step.state === "upcoming" ? String(position + 1).padStart(2, "0") :
    <FontAwesomeIcon icon={step.state === "error" ? faTriangleExclamation : step.state === "done" ? faCircleCheck : faClock} />}</span>;
}

function IngestProgress({ steps, transfer, onCopy, status, animateIngest = true }: { steps: TaskLifecycleStep[] } & StageInformation) {
  const { t } = useTranslation();
  const completed = steps.filter(step => step.state === "done").length;
  return <section className="task-ingest-progress" aria-label={t("taskFlow.ingestProgress")}>
    <div className="task-ingest-progress__head">
      <span>{t("taskFlow.stepsCompleted", { completed, total: steps.length })}</span>
    </div>
    <div className="task-ingest-progress__segments" aria-hidden="true">
      {steps.map(step => <span key={step.key} className={`is-${step.state}${animateIngest && step.state === "current" && step.key !== "pending_ingest" ? " is-animating" : ""}`} />)}
    </div>
    <ol className="task-ingest-progress__steps" aria-label={t("taskFlow.ingestSteps")}>
      {steps.map((step, index) => <li key={step.key} className={`task-ingest-progress__step is-${step.state}`}
        aria-current={isActive(step) ? "step" : undefined}>
        <div className={`task-ingest-progress__icon ${step.tone}`}><StepIcon step={step} position={index} /></div>
        <div className="task-ingest-progress__copy">
          <div className="task-ingest-progress__label"><strong>{step.label}</strong>{step.state !== "done" ? <span>{t(`taskFlow.states.${step.state}`)}</span> : null}</div>
          {isActive(step) ? <>
            {step.state === "error" && status ? <p className="task-timeline__failure"><strong>{status.summary}</strong></p> : null}
            {step.key !== "transferring" || !transfer?.target || step.state === "error" ? <p>{step.detail}</p> : null}
            {step.key === "transferring" && (transfer?.action || transfer?.target) ? <dl className="task-ingest-progress__information">
              {transfer.action ? <TaskValue label={t("taskInfo.transferMethod")} value={transferActionLabel(transfer.action)} /> : null}
              {transfer.target ? <TaskValue label={t("taskInfo.transferTarget")} value={transfer.target} onCopy={onCopy} /> : null}
            </dl> : null}
          </> : null}
        </div>
      </li>)}
    </ol>
  </section>;
}

interface TaskTimelineProps extends StageInformation {
  steps: TaskLifecycleStep[];
  action?: ReactNode;
  schedule?: {last?:string|null;next?:string|null};
  downloadProgress?: { percent: number; label: string; active?: boolean };
}

export function TaskTimeline({ steps, downloadProgress, transfer, onCopy, status, schedule, action, animateIngest }: TaskTimelineProps) {
  useAppTimezone();
  const { t } = useTranslation();
  const ingestSteps = steps.filter(step => !["sourcing", "downloading", "completed"].includes(step.key));
  const active = ingestSteps.find(isActive);
  const done = ingestSteps.length > 0 && ingestSteps.every(step => step.state === "done");
  const ingest: TaskLifecycleStep = {
    time: done ? steps.find(step => step.key === "completed")?.time : null,
    key: "ingest", label: t(done ? "taskFlow.ingested" : "taskExperience.ingest"), detail: active?.detail ?? "",
    state: active?.state ?? (done ? "done" : "upcoming"), tone: active?.tone ?? (done ? "tone-success" : "tone-neutral")
  };
  const stages = [steps.find(step => step.key === "sourcing"), steps.find(step => step.key === "downloading"), ingest]
    .filter((step): step is TaskLifecycleStep => Boolean(step));

  // Pending sourcing is the current waiting node even before its worker starts.
  const currentKey = stages.find(isActive)?.key ?? (status ? stages.find(step=>step.state === "upcoming")?.key : undefined);
  const displayedStages = stages.map(step=>step.key === currentKey && step.state === "upcoming" ? {...step,state:"current" as const,tone:status?.tone??step.tone} : step);
  return <ol className="task-timeline" aria-label={t("taskDetail.lifecycle")}>
    {displayedStages.map((step, index) => <li key={step.key} className={`task-timeline__item is-${step.state}`}
      aria-current={isActive(step) ? "step" : undefined}>
      <div className={`task-timeline__icon ${step.tone}`}><StepIcon step={step} position={index} /></div>
      <div className="task-timeline__copy">
        <div className="task-timeline__head">
          <strong>{step.label}</strong>
          {step.state !== "done" ? <span className="task-timeline__state">{isActive(step) && status ? status.label : t(`taskFlow.states.${step.state}`)}</span> : null}
          {step.time ? <time dateTime={step.time}>{step.state === "done" ? t("taskFlow.completedAt", { time: formatDateTime(step.time) }) : formatDateTime(step.time)}</time> : null}
        </div>
        {isActive(step) ? step.key === "ingest" ? <IngestProgress steps={ingestSteps} status={status} animateIngest={animateIngest} transfer={transfer} onCopy={onCopy} /> : <>
          {step.state === "error" && status ? <p className="task-timeline__failure"><strong>{status.summary}</strong></p> : null}
          {step.key !== "downloading" || !downloadProgress || step.state === "error" ? <p>{step.key === "sourcing" && status && step.state !== "error" ? status.detail : step.detail}</p> : null}
          {step.key === "downloading" && downloadProgress ? <div className="task-detail-progress">
            <span>{downloadProgress.label}</span>
            <div className="progress-shell progress-shell--detail" role="progressbar" aria-label={t("taskModel.states.downloading")}
              aria-valuemin={0} aria-valuemax={100} aria-valuenow={downloadProgress.percent}>
              <div className={`progress-fill${downloadProgress.active ? " is-animating" : ""}`} style={{ width: `${downloadProgress.percent}%` }} />
            </div>
          </div> : null}
        </> : null}
        {step.key === "sourcing" && schedule?.next ? <dl className="settings-grid task-detail-dates task-detail-schedule">
          <div><dt>{t("resourceTracking.lastChecked")}</dt><dd>{formatDateTime(schedule.last)}</dd></div>
          <div><dt>{t("resourceTracking.nextChecked")}</dt><dd>{formatDateTime(schedule.next)}</dd></div>
        </dl> : null}
        {isActive(step) && action ? <div className="task-timeline__actions">{action}</div> : null}
      </div>
    </li>)}
  </ol>;
}
