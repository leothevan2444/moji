import type { RefObject } from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faPlus } from "@fortawesome/free-solid-svg-icons";
import type { ResourceRuleInput } from "../../graphql/generated/graphql";

// Keep the platform's keyboard and touch picker behind the compact icon.
export function ResourceRulePicker({ label, rules, onSelect, selectRef, disabled = false }: {
  label: string;
  rules: ResourceRuleInput[];
  onSelect: (id: string) => void;
  selectRef: RefObject<HTMLSelectElement | null>;
  disabled?: boolean;
}) {
  const unavailable = disabled || !rules.length;
  return <span className={`resource-rule-picker icon-button${unavailable ? " is-disabled" : ""}`} title={label}>
    <FontAwesomeIcon icon={faPlus} aria-hidden="true" />
    <select ref={selectRef} aria-label={label} disabled={unavailable} value="" onChange={event => { if (rules.some(rule => rule.id === event.target.value)) onSelect(event.target.value); }}>
      <option value="" disabled>{label}</option>
      {rules.map(rule => <option value={rule.id} key={rule.id}>{rule.name}</option>)}
    </select>
  </span>;
}
