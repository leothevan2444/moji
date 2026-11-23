import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faTrashCan } from "@fortawesome/free-solid-svg-icons";
import type { ResourceConditionInput } from "../../graphql/generated/graphql";
import { emptyResourceCondition } from "./resourceSelectionDraft";

function RangeValue({ value, size, label, onChange }: { value: string; size: boolean; label: string; onChange: (value: string) => void }) {
  const display = (value: string) => size && value && Number.isFinite(Number(value)) ? String(Number(value) / (1024 ** 3)) : value;
  const [draft, setDraft] = useState(() => display(value));
  const emitted = useRef<string | null>(null);
  useEffect(() => { if (value !== emitted.current) setDraft(display(value)); }, [value, size]);
  return <input aria-label={label} title={label} placeholder={label} inputMode={size ? "decimal" : "numeric"} value={draft} onChange={event => {
    const text = event.target.value;
    setDraft(text);
    const bytes = Math.round(Number(text) * (1024 ** 3));
    const encoded = size && /^(?:\d+(?:\.\d*)?|\.\d+)$/.test(text) && Number(text) >= 0 && Number.isSafeInteger(bytes) ? String(bytes) : text;
    emitted.current = encoded;
    onChange(encoded);
  }} />;
}

export function ResourceConditionRow({ value, index, canRemove, onChange, onRemove }: {
  value: ResourceConditionInput;
  index: number;
  canRemove: boolean;
  onChange: (value: ResourceConditionInput) => void;
  onRemove: () => void;
}) {
  const { t } = useTranslation();
  const patch = (patch: Partial<ResourceConditionInput>) => onChange({ ...value, ...patch });
  const numeric = value.scope === "SIZE" || value.scope === "SEEDERS";
  return <div className="resource-condition-row" role="group" aria-label={t("resourceSelection.conditionNumber", { number: index + 1 })}>
    <select aria-label={t("resourceSelection.scope")} value={value.scope} onChange={event => onChange({ ...emptyResourceCondition(), scope: event.target.value })}>
      {["TITLE", "SIZE", "SEEDERS"].map(scope => <option key={scope} value={scope}>{t(`resourceSelection.conditionScopes.${scope}`)}</option>)}
    </select>
    <div className="resource-condition-row__value">
      {numeric ? <>
        <RangeValue key={`${value.scope}-min`} value={value.minimum} size={value.scope === "SIZE"} label={t(value.scope === "SIZE" ? "resourceSelection.minimumGiB" : "resourceSelection.minimum")} onChange={minimum => patch({ minimum })} />
        <span aria-hidden="true">–</span>
        <RangeValue key={`${value.scope}-max`} value={value.maximum} size={value.scope === "SIZE"} label={t(value.scope === "SIZE" ? "resourceSelection.maximumGiB" : "resourceSelection.maximum")} onChange={maximum => patch({ maximum })} />
      </> : <input aria-label={t("resourceSelection.expression")} title={t("resourceSelection.expressionHelp")} placeholder={t("resourceSelection.expression")} value={value.value} maxLength={514} spellCheck={false} onChange={event => patch({ value: event.target.value })} />}
    </div>
    <button type="button" className="icon-button resource-selection__delete" aria-label={t("resourceSelection.removeCondition")} title={t("resourceSelection.removeCondition")} disabled={!canRemove} onClick={onRemove}><FontAwesomeIcon icon={faTrashCan} aria-hidden="true" /></button>
  </div>;
}
