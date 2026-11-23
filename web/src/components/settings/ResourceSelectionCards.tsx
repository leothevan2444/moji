import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleCheck, faGripVertical, faCircle, faTrashCan } from "@fortawesome/free-solid-svg-icons";
import type { ResourceRuleGroupInput, ResourceRuleInput } from "../../graphql/generated/graphql";
import { useListReorder } from "../../hooks/useListReorder";

export function ResourceRuleLibrary({ rules, onChange, onEdit, onDelete }: {
  rules: ResourceRuleInput[];
  onChange: (rules: ResourceRuleInput[]) => void;
  onEdit: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  const { t } = useTranslation();
  const reorder = useListReorder<HTMLDivElement>(rules.map(rule => rule.id), (from, to) => {
    const next = [...rules];
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    onChange(next);
  }, false, { horizontal: true, attribute: "data-library-rule-id", scrollSelector: ".settings-main__content" });
  if (!rules.length) return <p className="settings-note">{t("resourceSelection.noRules")}</p>;
  return <div className="resource-selection__rule-grid" ref={reorder.list} role="list" aria-label={t("resourceSelection.ruleLibrary")}>
    {rules.map((rule, index) => {
      const name = rule.name.trim() || t("resourceSelection.ruleNumber", { number: index + 1 });
      const removeLabel = t("resourceSelection.deleteRuleNamed", { name });
      return <div key={rule.id} role="listitem" className={`resource-selection__rule-tile${reorder.dragged === rule.id ? " is-dragging" : ""}${reorder.over === rule.id ? " is-drop-target" : ""}`} {...reorder.row(rule.id)} onClick={event => { if (!(event.target as HTMLElement).closest("button")) { event.currentTarget.querySelector<HTMLButtonElement>(".resource-selection__rule-edit")?.focus(); onEdit(rule.id); } }}>
        <button type="button" className="icon-button resource-selection__drag" aria-label={t("resourceSelection.dragRule", { name })} title={t("resourceSelection.libraryOrderHelp")} aria-keyshortcuts="ArrowLeft ArrowRight ArrowUp ArrowDown" {...reorder.handle(rule.id)}><FontAwesomeIcon icon={faGripVertical} aria-hidden="true" /></button>
        <button type="button" className="resource-selection__rule-edit" title={name} aria-haspopup="dialog" onClick={() => onEdit(rule.id)}>{name}</button>
        <button type="button" className="icon-button resource-selection__delete" aria-label={removeLabel} title={removeLabel} onClick={() => onDelete(rule.id)}><FontAwesomeIcon icon={faTrashCan} aria-hidden="true" /></button>
      </div>;
    })}
  </div>;
}

export function ResourceGroupCards({ groups, activeGroup, onEdit, onActivate, onDelete, onChange }: {
  groups: ResourceRuleGroupInput[];
  activeGroup: string;
  onEdit: (id: string) => void;
  onActivate: (id: string) => void;
  onChange: (groups: ResourceRuleGroupInput[]) => void;
  onDelete: (id: string) => void;
}) {
  const { t } = useTranslation();
  const reorder = useListReorder<HTMLDivElement>(groups.map(group => group.id), (from, to) => {
    const next = [...groups];
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    onChange(next);
  }, false, { horizontal: true, attribute: "data-library-group-id", scrollSelector: ".settings-main__content" });
  if (!groups.length) return <p className="settings-note">{t("resourceSelection.noGroupItems")}</p>;
  return <div className="resource-selection__group-grid" ref={reorder.list} role="list" aria-label={t("resourceSelection.groups")}>
    {groups.map((group, index) => {
      const name = group.name.trim() || t("resourceSelection.groupNumber", { number: index + 1 });
      const active = group.id === activeGroup;
      const activateLabel = t(active ? "resourceSelection.activeGroupNamed" : "resourceSelection.useGroup", { name });
      const removeLabel = t("resourceSelection.deleteGroupNamed", { name });
      return <article role="listitem" className={`resource-selection__group-tile${active ? " is-active" : ""}${reorder.dragged === group.id ? " is-dragging" : ""}${reorder.over === group.id ? " is-drop-target" : ""}`} key={group.id} {...reorder.row(group.id)} onClick={event => { if (!(event.target as HTMLElement).closest("button")) { event.currentTarget.querySelector<HTMLButtonElement>(".resource-selection__group-edit")?.focus(); onEdit(group.id); } }}>
        <div className="resource-selection__group-heading">
          <button type="button" className="icon-button resource-selection__drag" aria-label={t("resourceSelection.dragGroup", { name })} title={t("resourceSelection.groupOrderHelp")} aria-keyshortcuts="ArrowLeft ArrowRight ArrowUp ArrowDown" {...reorder.handle(group.id)}><FontAwesomeIcon icon={faGripVertical} aria-hidden="true" /></button>
          <button type="button" className="resource-selection__group-edit" title={name} aria-label={name} aria-haspopup="dialog" onClick={() => onEdit(group.id)}><strong>{name}</strong></button>
        </div>
        <div className="resource-selection__group-body">
          <dl className="resource-selection__group-stats">
            <div><dt title={t("resourceSelection.cardAdmission")}>{t("resourceSelection.cardAdmission")}</dt><dd>{group.require.length}</dd></div>
            <div><dt title={t("resourceSelection.cardExclusion")}>{t("resourceSelection.cardExclusion")}</dt><dd>{group.exclude.length}</dd></div>
            <div><dt title={t("resourceSelection.cardTiers")}>{t("resourceSelection.cardTiers")}</dt><dd>{group.tiers.length}</dd></div>
          </dl>
          {group.tiers.length ? <ol className="resource-selection__tier-preview" aria-label={t("resourceSelection.priorityTiers")}>
            {group.tiers.slice(0, 3).map((tier, i) => <li key={tier.id}><span aria-hidden="true">{i + 1}.</span><span title={tier.name}>{tier.name}</span></li>)}
            {group.tiers.length > 3 ? <li><span aria-hidden="true">…</span><span>{t("resourceSelection.moreTiers", { count: group.tiers.length - 3 })}</span></li> : null}
          </ol> : null}
        </div>
        <div className="resource-selection__card-actions">
          <button type="button" className={`resource-selection__activate${active ? " is-active" : ""}`} aria-label={activateLabel} title={activateLabel} aria-pressed={active} disabled={active} onClick={() => onActivate(group.id)}><FontAwesomeIcon icon={active ? faCircleCheck : faCircle} aria-hidden="true" />{t(active ? "resourceSelection.activeLabel" : "resourceSelection.inactiveLabel")}</button>
          <span className="resource-selection__group-fallback" title={t(group.fallback ? "resourceSelection.fallbackOn" : "resourceSelection.fallbackOff")}>{t(group.fallback ? "resourceSelection.fallbackOn" : "resourceSelection.fallbackOff")}</span>
          <button type="button" className="icon-button resource-selection__delete" aria-label={removeLabel} title={removeLabel} onClick={() => onDelete(group.id)}><FontAwesomeIcon icon={faTrashCan} aria-hidden="true" /></button>
        </div>
      </article>;
    })}
  </div>;
}
