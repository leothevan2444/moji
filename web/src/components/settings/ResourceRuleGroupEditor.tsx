import { useListReorder } from "../../hooks/useListReorder";
import { useTranslation } from "react-i18next";
import type { ResourceRuleInput, ResourceRuleGroupInput } from "../../graphql/generated/graphql";
import { ResourceRuleTags } from "./ResourceRuleTags";
import { ResourceTierCard } from "./ResourceTierCard";
import { resourceRuleID } from "./resourceSelectionDraft";

export function ResourceRuleGroupEditor({ value, onChange, rules }: {
  value: ResourceRuleGroupInput;
  onChange: (value: ResourceRuleGroupInput) => void;
  rules: ResourceRuleInput[];
}) {
  const { t } = useTranslation();
  const patch = (patch: Partial<ResourceRuleGroupInput>) => onChange({ ...value, ...patch });
  const reorder = useListReorder<HTMLDivElement>(value.tiers.map(tier => tier.id), (from, to) => {
    const tiers = [...value.tiers];
    const [tier] = tiers.splice(from, 1);
    tiers.splice(to, 0, tier);
    patch({ tiers });
  }, false, { horizontal: true, attribute: "data-tier-id" });
  return <div className="resource-rule-editor settings-form">
    <fieldset className="resource-selection__prefilter"><legend>{t("resourceSelection.prefilter")}</legend>
      <ResourceRuleTags label={t("resourceSelection.require")} description={t("resourceSelection.requireAnyHelp")} rules={rules} value={value.require} unavailable={value.exclude} onChange={require => patch({ require })} />
      <ResourceRuleTags label={t("resourceSelection.exclude")} description={t("resourceSelection.excludeHelp")} rules={rules} value={value.exclude} unavailable={[...value.require, ...value.tiers.flatMap(tier => tier.rules)]} onChange={exclude => patch({ exclude })} />
    </fieldset>
    <div className="resource-selection__tier-heading"><p className="settings-note">{t("resourceSelection.tiersHelp")}</p><button type="button" className="ghost-button" disabled={value.tiers.length >= 16} onClick={() => patch({ tiers: [...value.tiers, { id: resourceRuleID(), name: t("resourceSelection.tierNumber", { number: value.tiers.length + 1 }), rules: [] }] })}>{t("resourceSelection.addTier")}</button></div>
    <div className="resource-selection__tier-grid" ref={reorder.list}>{value.tiers.map((tier, index) => <ResourceTierCard key={tier.id} value={tier} index={index} rules={rules} excluded={value.exclude}
      onChange={tier => patch({ tiers: value.tiers.map((item, i) => i === index ? tier : item) })}
      rowProps={reorder.row(tier.id)} handleProps={reorder.handle(tier.id)} dragged={reorder.dragged === tier.id} over={reorder.over === tier.id} onRemove={() => patch({ tiers: value.tiers.filter((_, i) => i !== index) })} />)}</div>
    {!value.tiers.length ? <p className="settings-note">{t(value.fallback ? "resourceSelection.noTiersFallback" : "resourceSelection.noTiersReject")}</p> : null}
  </div>;
}
