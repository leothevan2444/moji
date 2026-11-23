export function ResourceEditableTitle({ value, label, onChange }: {
  value: string;
  label: string;
  onChange: (value: string) => void;
}) {
  return <input className="resource-editable-title" aria-label={label} title={label} placeholder={label} value={value} maxLength={160} onChange={event => onChange(event.target.value)} onKeyDown={event => { if (event.key === "Enter") { event.preventDefault(); event.currentTarget.blur(); } }} />;
}
