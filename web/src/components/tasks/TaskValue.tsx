import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCopy } from "@fortawesome/free-solid-svg-icons/faCopy";
import { useTranslation } from "react-i18next";

interface TaskValueProps {
  label: string;
  value?: string | null;
  className?: string;
  onCopy?: (value: string, successMessage: string) => void | Promise<void>;
}

export function TaskValue({ label, value, className, onCopy }: TaskValueProps) {
  const { t } = useTranslation();
  return <div className={className}>
    <dt>{label}</dt>
    <dd>{value ? <div className="task-value">
      <span className="task-value__text task-inline-value task-inline-value--truncate" title={value}>{value}</span>
      {onCopy ? <button type="button" className="ghost-button task-value__copy"
        aria-label={t("taskExperience.copy", { label })}
        onClick={() => void onCopy(value, t("taskExperience.copied", { label }))}><FontAwesomeIcon icon={faCopy} /></button> : null}
    </div> : "—"}</dd>
  </div>;
}
