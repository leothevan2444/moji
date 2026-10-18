import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faEllipsis } from "@fortawesome/free-solid-svg-icons/faEllipsis";
import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { taskSummary, type DashboardTask } from "../../utils";

export function TaskMoreActions({ task, pendingDelete, onDelete }: {
  task: DashboardTask;
  pendingDelete?: boolean;
  onDelete: (id: string) => void;
}) {
  const { t } = useTranslation();
  const ref = useRef<HTMLDetailsElement>(null);
  useEffect(() => {
    const close = (event: PointerEvent) => {
      if (event.target instanceof Node && !ref.current?.contains(event.target)) ref.current?.removeAttribute("open");
    };
    document.addEventListener("pointerdown", close);
    return () => document.removeEventListener("pointerdown", close);
  }, []);
  return <details ref={ref} className="task-more"
    onClick={event => event.stopPropagation()}
    onBlur={event => { if (!event.currentTarget.contains(event.relatedTarget)) event.currentTarget.open = false; }}
    onKeyDown={event => {
      if (event.key === "Escape" && ref.current?.open) {
        event.stopPropagation();
        ref.current.open = false;
        ref.current.querySelector("summary")?.focus();
      }
    }}
  >
    <summary aria-label={t("taskExperience.more", { task: taskSummary(task) })}><FontAwesomeIcon icon={faEllipsis} /></summary>
    <div className="task-more__panel" aria-label={t("taskExperience.options")}>
      <button type="button" className="ghost-button task-ops__button--danger" disabled={pendingDelete}
        aria-label={t("taskUi.card.deleteLabel", { task: taskSummary(task) })}
        onClick={() => { ref.current?.removeAttribute("open"); ref.current?.querySelector("summary")?.focus(); onDelete(task.id); }}
      >{t(pendingDelete ? "taskUi.card.deleting" : "taskUi.card.delete")}</button>
    </div>
  </details>;
}
