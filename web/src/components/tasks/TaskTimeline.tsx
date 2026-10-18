import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleCheck } from "@fortawesome/free-solid-svg-icons/faCircleCheck";
import { faClock } from "@fortawesome/free-solid-svg-icons/faClock";
import { faTriangleExclamation } from "@fortawesome/free-solid-svg-icons/faTriangleExclamation";
import { useTranslation } from "react-i18next";
import { formatDateTime, type TaskLifecycleStep } from "../../utils";

const isActive = (step: TaskLifecycleStep) => step.state === "current" || step.state === "error";

function StepIcon({ step, position }: { step: TaskLifecycleStep; position: number }) {
  return <span aria-hidden="true">{step.state === "upcoming" ? String(position + 1).padStart(2, "0") :
    <FontAwesomeIcon icon={step.state === "error" ? faTriangleExclamation : step.state === "done" ? faCircleCheck : faClock} />}</span>;
}

function IngestProgress({ steps }: { steps: TaskLifecycleStep[] }) {
  const { t } = useTranslation();
  const completed = steps.filter(step => step.state === "done").length;
  return <section className="task-ingest-progress" aria-label={t("taskFlow.ingestProgress")}>
    <div className="task-ingest-progress__head">
      <strong>{t("taskFlow.ingestProgress")}</strong>
      <span>{t("taskFlow.stepsCompleted", { completed, total: steps.length })}</span>
    </div>
    <div className="task-ingest-progress__segments" aria-hidden="true">
      {steps.map(step => <span key={step.key} className={`is-${step.state}`} />)}
    </div>
    <ol className="task-ingest-progress__steps" aria-label={t("taskFlow.ingestSteps")}>
      {steps.map((step, index) => <li key={step.key} className={`task-ingest-progress__step is-${step.state}`}
        aria-current={isActive(step) ? "step" : undefined}>
        <div className={`task-ingest-progress__icon ${step.tone}`}><StepIcon step={step} position={index} /></div>
        <div className="task-ingest-progress__copy">
          <div className="task-ingest-progress__label"><strong>{step.label}</strong><span>{t(`taskFlow.states.${step.state}`)}</span></div>
          {isActive(step) ? <p>{step.detail}</p> : null}
        </div>
      </li>)}
    </ol>
  </section>;
}

export function TaskTimeline({ steps }: { steps: TaskLifecycleStep[] }) {
  const { t } = useTranslation();
  const ingestSteps = steps.filter(step => !["sourcing", "downloading", "completed"].includes(step.key));
  const active = ingestSteps.find(isActive);
  const done = ingestSteps.length > 0 && ingestSteps.every(step => step.state === "done");
  const ingest: TaskLifecycleStep = {
    key: "ingest", label: t("taskExperience.ingest"), detail: active?.detail ?? "",
    state: active?.state ?? (done ? "done" : "upcoming"), tone: active?.tone ?? (done ? "tone-success" : "tone-neutral")
  };
  const stages = [steps.find(step => step.key === "sourcing"), steps.find(step => step.key === "downloading"), ingest]
    .filter((step): step is TaskLifecycleStep => Boolean(step));

  return <ol className="task-timeline" aria-label={t("taskDetail.lifecycle")}>
    {stages.map((step, index) => <li key={step.key} className={`task-timeline__item is-${step.state}`}
      aria-current={isActive(step) ? "step" : undefined}>
      <div className={`task-timeline__icon ${step.tone}`}><StepIcon step={step} position={index} /></div>
      <div className="task-timeline__copy">
        <div className="task-timeline__head">
          <strong>{step.label}</strong>
          <span className="task-timeline__state">{t(`taskFlow.states.${step.state}`)}</span>
          {step.time ? <time dateTime={step.time}>{formatDateTime(step.time)}</time> : null}
        </div>
        {isActive(step) ? step.key === "ingest" ? <IngestProgress steps={ingestSteps} /> : <p>{step.detail}</p> : null}
      </div>
    </li>)}
  </ol>;
}
