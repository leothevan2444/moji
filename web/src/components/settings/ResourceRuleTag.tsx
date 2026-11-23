import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faXmark } from "@fortawesome/free-solid-svg-icons";

export function ResourceRuleTag({ name, removeLabel, onRemove }: { name: string; removeLabel: string; onRemove: () => void }) {
  return <span role="listitem" className="resource-rule-tags__tag">
    <span className="resource-rule-tags__name">{name}</span>
    <button type="button" className="resource-rule-tags__remove" aria-label={removeLabel} title={removeLabel} onClick={onRemove}><FontAwesomeIcon icon={faXmark} aria-hidden="true" /></button>
  </span>;
}
