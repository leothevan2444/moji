import { useRef, type HTMLAttributes } from "react";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faGripVertical, faTrashCan } from "@fortawesome/free-solid-svg-icons";
import type { ResourceRuleInput, ResourceRuleTierInput } from "../../graphql/generated/graphql";
import { ResourceEditableTitle } from "./ResourceEditableTitle";
import { ResourceRulePicker } from "./ResourceRulePicker";
import { SettingsCard } from "./SettingsTabCommon";
import { ResourceRuleReferences } from "./ResourceRuleReferences";

export function ResourceTierCard({ value, index, rules, excluded, onChange, onRemove, rowProps, handleProps, dragged, over }: {
  value: ResourceRuleTierInput;
  index: number;
  rules: ResourceRuleInput[];
  excluded: string[];
  onChange: (value: ResourceRuleTierInput) => void;
  rowProps: HTMLAttributes<HTMLElement>;
  handleProps: HTMLAttributes<HTMLElement>;
  dragged: boolean;
  over: boolean;
  onRemove: () => void;
}) {
  const { t } = useTranslation();
  const addRef = useRef<HTMLSelectElement>(null);
  const available = rules.filter(rule => !excluded.includes(rule.id) && !value.rules.includes(rule.id));
  return <section {...rowProps} className={`resource-selection__tier${dragged ? " is-dragging" : ""}${over ? " is-drop-target" : ""}`} aria-label={t("resourceSelection.tierNumber", { number: index + 1 })}>
    <SettingsCard title={<><button type="button" className="icon-button resource-selection__drag" aria-label={t("resourceSelection.dragTier", { name: value.name })} title={t("resourceSelection.tierOrderHelp")} aria-keyshortcuts="ArrowLeft ArrowRight ArrowUp ArrowDown" {...handleProps}><FontAwesomeIcon icon={faGripVertical} aria-hidden="true" /></button><ResourceEditableTitle value={value.name} label={t("resourceSelection.tierName")} onChange={name => onChange({ ...value, name })} /></>} actions={<>
      <ResourceRulePicker label={t("resourceSelection.addTierRule")} rules={available} selectRef={addRef} disabled={value.rules.length >= 32} onSelect={ruleID => onChange({ ...value, rules: [...value.rules, ruleID] })} />
      <button type="button" className="icon-button resource-selection__delete" aria-label={t("resourceSelection.removeTier")} title={t("resourceSelection.removeTier")} onClick={onRemove}><FontAwesomeIcon icon={faTrashCan} aria-hidden="true" /></button>
    </>}>
      <ResourceRuleReferences addRef={addRef} label={t("resourceSelection.tierRules")} rules={rules} value={value.rules} onChange={refs => onChange({ ...value, rules: refs })} />
    </SettingsCard>
  </section>;
}
