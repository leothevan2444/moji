import { useRef } from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faPlus } from "@fortawesome/free-solid-svg-icons";
import { useTranslation } from "react-i18next";
import type { ResourceRuleInput } from "../../graphql/generated/graphql";
import { FieldLabel } from "./SettingsTabCommon";
import { ResourceConditionRow } from "./ResourceConditionRow";
import { emptyResourceCondition } from "./resourceSelectionDraft";

export function ResourceRuleEditor({ value, onChange }: {
  value: ResourceRuleInput;
  onChange: (value: ResourceRuleInput) => void;
}) {
  const { t } = useTranslation();
  const addButton = useRef<HTMLButtonElement>(null);
  const remove = (index: number) => {
    onChange({ ...value, conditions: value.conditions.filter((_, i) => i !== index) });
    queueMicrotask(() => addButton.current?.focus());
  };
  return <div className="resource-rule-editor settings-form">
    <div className="resource-rule-editor__conditions">
      <div className="resource-rule-editor__condition-heading">
        <div className="settings-field"><FieldLabel text={t("resourceSelection.anyConditionHelp")} info={[t("resourceSelection.expressionHelp"), t("resourceSelection.rangeHelp"), t("resourceSelection.emptyConditionsHelp")].join(" ")} /></div>
        <button ref={addButton} type="button" className="icon-button" title={t("resourceSelection.addCondition")} aria-label={t("resourceSelection.addCondition")} onClick={() => onChange({ ...value, conditions: [...value.conditions, emptyResourceCondition()] })}><FontAwesomeIcon icon={faPlus} aria-hidden="true" /></button>
      </div>
      {value.conditions.map((condition, index) => <ResourceConditionRow key={index} value={condition} index={index} canRemove={value.conditions.length > 1} onRemove={() => remove(index)} onChange={condition => onChange({ ...value, conditions: value.conditions.map((item, i) => i === index ? condition : item) })} />)}
    </div>
  </div>;
}
