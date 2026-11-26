import { faCheck } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import "../../styles/card-selection.scss";

export function CardSelectionControl({ checked, disabled = false, label, description, inputClassName, onChange }: {
  checked: boolean;
  disabled?: boolean;
  label: string;
  description?: string;
  inputClassName: string;
  onChange: () => void;
}) {
  return <label className={`card-selection-control${disabled ? " is-disabled" : ""}`} title={description || label} onClick={event => event.stopPropagation()}>
    <input type="checkbox" className={`card-selection-control__input ${inputClassName}`} checked={checked} disabled={disabled}
      aria-label={label} aria-description={description} onChange={onChange} />
    <span className="card-selection-control__mark" aria-hidden="true"><FontAwesomeIcon icon={faCheck} /></span>
  </label>;
}
