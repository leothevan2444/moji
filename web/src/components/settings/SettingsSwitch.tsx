import "../../styles/settings-system.scss";

export function SettingsSwitch({ checked, disabled = false, label, onChange }: {
  checked: boolean;
  disabled?: boolean;
  label: string;
  onChange: (checked: boolean) => void;
}) {
  return <span className="switch settings-switch">
    <input type="checkbox" role="switch" aria-label={label} checked={checked} disabled={disabled} onChange={event => onChange(event.target.checked)} />
    <span className="switch__track" aria-hidden="true" />
    <span className="switch__thumb" aria-hidden="true" />
  </span>;
}
