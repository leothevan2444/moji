import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faCircleCheck,
  faClock,
  faRotate,
  faTriangleExclamation
} from "@fortawesome/free-solid-svg-icons";
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
