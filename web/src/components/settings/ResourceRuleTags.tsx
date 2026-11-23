import { useId, useRef } from "react";
import { useTranslation } from "react-i18next";
import { ResourceRulePicker } from "./ResourceRulePicker";
import { ResourceRuleTag } from "./ResourceRuleTag";
import type { ResourceRuleInput } from "../../graphql/generated/graphql";

export function ResourceRuleTags({ label, description, rules, value, onChange, unavailable }: {
  label: string;
  description: string;
  rules: ResourceRuleInput[];
  value: string[];
  unavailable: string[];
  onChange: (value: string[]) => void;
}) {
  const { t } = useTranslation();
  const addRef = useRef<HTMLSelectElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const heading = useId();
  const hint = useId();
  const selected = rules.filter(rule => value.includes(rule.id));
  const available = rules.filter(rule => !value.includes(rule.id) && !unavailable.includes(rule.id));
  const remove = (id: string, index: number) => {
    onChange(selected.filter(rule => rule.id !== id).map(rule => rule.id));
    queueMicrotask(() => {
      const buttons = listRef.current?.querySelectorAll<HTMLButtonElement>("button");
      (buttons?.[Math.min(index, buttons.length - 1)] ?? addRef.current)?.focus();
    });
  };
  return <section className="resource-rule-tags" role="group" aria-labelledby={heading} aria-describedby={hint}>
    <div className="resource-rule-tags__heading">
      <h4 id={heading}>{label}</h4><p className="settings-note" id={hint}>{description}</p>
      <ResourceRulePicker label={t("resourceSelection.addFilterRule", { section: label })} rules={available} selectRef={addRef} onSelect={id => onChange(rules.filter(rule => value.includes(rule.id) || rule.id === id).map(rule => rule.id))} />
    </div>
    <div className="resource-rule-tags__items" ref={listRef} role="list" aria-label={label}>
      {selected.map((rule, index) => <ResourceRuleTag key={rule.id} name={rule.name} removeLabel={t("resourceSelection.removeReference", { name: rule.name })} onRemove={() => remove(rule.id, index)} />)}
    </div>
  </section>;
}
