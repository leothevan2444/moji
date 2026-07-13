import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleCheck } from "@fortawesome/free-solid-svg-icons/faCircleCheck";
import { faClock } from "@fortawesome/free-solid-svg-icons/faClock";
import { faRotate } from "@fortawesome/free-solid-svg-icons/faRotate";
import { faTriangleExclamation } from "@fortawesome/free-solid-svg-icons/faTriangleExclamation";
import { formatDateTime, type TaskLifecycleStep } from "../../utils";

interface TaskTimelineProps {
  steps: TaskLifecycleStep[];
}

export function TaskTimeline({ steps }: TaskTimelineProps) {
  return (
    <div className="task-timeline">
      {steps.map((step) => (
        <div key={step.key} className={`task-timeline__item is-${step.state}`}>
          <div className={`task-timeline__icon ${step.tone}`}>
            <FontAwesomeIcon
              icon={
                step.state === "error"
                  ? faTriangleExclamation
                  : step.state === "current"
                    ? faClock
                    : step.state === "done"
                      ? faCircleCheck
                      : faRotate
              }
            />
          </div>
          <div className="task-timeline__copy">
            <div className="task-timeline__head">
              <strong>{step.label}</strong>
              <span>{formatDateTime(step.time)}</span>
            </div>
            <p>{step.detail}</p>
          </div>
        </div>
      ))}
    </div>
  );
}
