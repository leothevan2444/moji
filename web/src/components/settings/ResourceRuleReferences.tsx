import type { RefObject } from "react";
import { useTranslation } from "react-i18next";
import type { ResourceRuleInput } from "../../graphql/generated/graphql";
import { ResourceRuleTag } from "./ResourceRuleTag";

export function ResourceRuleReferences({ label, rules, value, onChange, addRef }: {
  addRef: RefObject<HTMLSelectElement | null>; label: string; rules: ResourceRuleInput[]; value: string[]; onChange: (value: string[]) => void;
}) {
  const { t } = useTranslation();
  if (!value.length) return null;
  return <div className="resource-selection__references" role="group" aria-label={label}>
    <div className="resource-rule-tags__items" role="list" aria-label={label}>
      {value.map(id => <ResourceRuleTag key={id} name={rules.find(rule => rule.id === id)?.name || id} removeLabel={t("resourceSelection.removeReference", { name: rules.find(rule => rule.id === id)?.name || id })}
        onRemove={() => { onChange(value.filter(item => item !== id)); queueMicrotask(() => addRef.current?.focus()); }} />)}
    </div>
  </div>;
}
