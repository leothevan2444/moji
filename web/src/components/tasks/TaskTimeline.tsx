import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleCheck } from "@fortawesome/free-solid-svg-icons/faCircleCheck";
import { faClock } from "@fortawesome/free-solid-svg-icons/faClock";
import { faRotate } from "@fortawesome/free-solid-svg-icons/faRotate";
import { faTriangleExclamation } from "@fortawesome/free-solid-svg-icons/faTriangleExclamation";
import { useTranslation } from "react-i18next";
import { formatDateTime, type TaskLifecycleStep } from "../../utils";

function Steps({ steps }: { steps: TaskLifecycleStep[] }) {
  return <ol className="task-timeline">
    {steps.map(step => <li key={step.key} className={`task-timeline__item is-${step.state}`}
      aria-current={step.state === "current" || step.state === "error" ? "step" : undefined}>
      <div className={`task-timeline__icon ${step.tone}`}>
        <FontAwesomeIcon icon={step.state === "error" ? faTriangleExclamation : step.state === "current" ? faClock : step.state === "done" ? faCircleCheck : faRotate} />
      </div>
      <div className="task-timeline__copy">
        <div className="task-timeline__head"><strong>{step.label}</strong>
          {step.time ? <time dateTime={step.time}>{formatDateTime(step.time)}</time> : null}
        </div>
        <p>{step.detail}</p>
      </div>
    </li>)}
  </ol>;
}

export function TaskTimeline({ steps }: { steps: TaskLifecycleStep[] }) {
  const { t } = useTranslation();
  const ingestSteps = steps.filter(step => !["sourcing", "downloading", "completed"].includes(step.key));
  const active = ingestSteps.find(step => step.state === "error" || step.state === "current");
  const done = ingestSteps.length > 0 && ingestSteps.every(step => step.state === "done");
  const ingest: TaskLifecycleStep = {
    key: "ingest", label: t("taskExperience.ingest"),
    detail: active ? t("taskExperience.currentStep", { stage: active.label }) : t(done ? "taskModel.ingest.stepDone" : "taskModel.ingest.stepUpcoming"),
    state: active?.state ?? (done ? "done" : "upcoming"), tone: active?.tone ?? (done ? "tone-success" : "tone-neutral")
  };
  const summary = [steps.find(step => step.key === "sourcing"), steps.find(step => step.key === "downloading"), ingest, steps.find(step => step.key === "completed")]
    .filter((step): step is TaskLifecycleStep => Boolean(step));
  return <>
    <Steps steps={summary} />
    <details className="task-ingest-steps"><summary>{t("taskExperience.steps")}</summary><Steps steps={ingestSteps} /></details>
  </>;
}
