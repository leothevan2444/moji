import { useEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faTrashCan } from "@fortawesome/free-solid-svg-icons";
import type { DEFAULT_RESOURCE_SELECTION } from "../../constants/forms";
import { ConfirmationDialog } from "../common/ConfirmationDialog";
import { Drawer } from "../layout/Drawer";
import { SettingsSwitch } from "./SettingsSwitch";
import { SettingsCard, FieldLabel } from "./SettingsTabCommon";
import { ResourceEditableTitle } from "./ResourceEditableTitle";
import { ResourceRuleEditor } from "./ResourceRuleEditor";
import { ResourceRuleGroupEditor } from "./ResourceRuleGroupEditor";
import { ResourceGroupCards, ResourceRuleLibrary } from "./ResourceSelectionCards";
import { emptyResourceCondition, resourceRuleID } from "./resourceSelectionDraft";
import "../../styles/resource-selection.scss";

type Draft = typeof DEFAULT_RESOURCE_SELECTION;
type Target = { kind: "rule" | "group"; id: string };

export function ResourceSelectionEditor({ value, onChange, saveStatus, active = true }: {
  value: Draft;
  onChange: (value: Draft) => void;
  saveStatus?: ReactNode;
  active?: boolean;
}) {
  const { t } = useTranslation();
  const [editor, setEditor] = useState<Target | null>(null);
  const [deleting, setDeleting] = useState<Target | null>(null);
  const addRuleRef = useRef<HTMLButtonElement>(null);
  const addGroupRef = useRef<HTMLButtonElement>(null);
  useEffect(() => { if (!active) { setEditor(null); setDeleting(null); } }, [active]);
  const patch = (patch: Partial<Draft>) => onChange({ ...value, ...patch });
  const references = (id: string) => value.groups.filter(group => [...group.require, ...group.exclude, ...group.tiers.flatMap(tier => tier.rules)].includes(id));
  const rule = editor?.kind === "rule" ? value.rules.find(rule => rule.id === editor.id) : undefined;
  const group = editor?.kind === "group" ? value.groups.find(group => group.id === editor.id) : undefined;
  const usedBy = rule ? references(rule.id) : [];
  const editing = active && Boolean(rule || group);
  const close = () => setEditor(null);
  const addRule = () => {
    const id = resourceRuleID();
    patch({ rules: [...value.rules, { id, name: t("resourceSelection.ruleNumber", { number: value.rules.length + 1 }), conditions: [emptyResourceCondition()] }] });
    setEditor({ kind: "rule", id });
  };
  const addGroup = () => {
    const id = resourceRuleID();
    patch({ activeGroup: value.activeGroup || id, groups: [...value.groups, {
      id, name: t("resourceSelection.groupNumber", { number: value.groups.length + 1 }), fallback: false, require: [], exclude: [],
      tiers: value.rules.length ? [{ id: resourceRuleID(), name: t("resourceSelection.tierNumber", { number: 1 }), rules: [] }] : []
    }] });
    setEditor({ kind: "group", id });
  };
  const deleteRule = deleting?.kind === "rule" ? value.rules.find(item => item.id === deleting.id) : undefined;
  const deleteGroup = deleting?.kind === "group" ? value.groups.find(item => item.id === deleting.id) : undefined;
  const nextGroup = deleteGroup ? value.groups.find(item => item.id !== deleteGroup.id) : undefined;
  const confirmDelete = () => {
    if (deleteRule) {
      patch({ rules: value.rules.filter(item => item.id !== deleteRule.id), groups: value.groups.map(group => ({
        ...group, require: group.require.filter(id => id !== deleteRule.id), exclude: group.exclude.filter(id => id !== deleteRule.id),
        tiers: group.tiers.map(tier => ({ ...tier, rules: tier.rules.filter(id => id !== deleteRule.id) }))
      })) });
    } else if (deleteGroup) {
      const groups = value.groups.filter(item => item.id !== deleteGroup.id);
      patch({ groups, activeGroup: value.activeGroup === deleteGroup.id ? groups[0]?.id ?? "" : value.activeGroup });
    } else return;
    if (editor?.kind === deleting?.kind && editor?.id === deleting?.id) close();
    setDeleting(null);
    queueMicrotask(() => (deleteRule ? addRuleRef : addGroupRef).current?.focus());
  };

  return <div className="resource-selection">
    <SettingsCard title={t("resourceSelection.ruleLibrary")} actions={<button type="button" className="ghost-button" ref={addRuleRef} disabled={value.rules.length >= 64} onClick={addRule}>{t("resourceSelection.addRule")}</button>}>
      <ResourceRuleLibrary rules={value.rules} onChange={rules => patch({ rules })} onEdit={id => setEditor({ kind: "rule", id })} onDelete={id => setDeleting({ kind: "rule", id })} />
    </SettingsCard>
    <SettingsCard title={t("resourceSelection.groups")} actions={<button type="button" className="ghost-button" ref={addGroupRef} disabled={value.groups.length >= 16} onClick={addGroup}>{t("resourceSelection.addGroup")}</button>}>
      <ResourceGroupCards onChange={groups => patch({ groups })} groups={value.groups} activeGroup={value.activeGroup} onActivate={activeGroup => patch({ activeGroup })} onEdit={id => setEditor({ kind: "group", id })} onDelete={id => setDeleting({ kind: "group", id })} />
    </SettingsCard>
    {!editing ? saveStatus : null}
    {editing ? createPortal(<Drawer visibleDrawer="settings" initialFocus="panel" closing={false} title={(rule ?? group)?.name || t(rule ? "resourceSelection.ruleNumber" : "resourceSelection.groupNumber", { number: 1 })} titleContent={<ResourceEditableTitle value={(rule ?? group)!.name} label={t(rule ? "resourceSelection.name" : "resourceSelection.groupName")} onChange={name => rule ? patch({ rules: value.rules.map(item => item.id === rule.id ? { ...item, name } : item) }) : patch({ groups: value.groups.map(item => item.id === group!.id ? { ...item, name } : item) })} />} onClose={close} className="resource-selection-dialog"
      footer={<div className="resource-selection-dialog__footer">{saveStatus}<div className="settings-actions">{group ? <label className="switch-row resource-selection__fallback"><FieldLabel text={t("resourceSelection.allowFallback")} info={t("resourceSelection.fallbackHelp")} /><SettingsSwitch label={t("resourceSelection.allowFallback")} checked={group.fallback} onChange={fallback => patch({ groups: value.groups.map(item => item.id === group.id ? { ...item, fallback } : item) })} /></label> : null}{rule && usedBy.length ? <p tabIndex={0} className="settings-note resource-selection-dialog__references">{t("resourceSelection.usedBy", { groups: usedBy.map(group => group.name).join("、") })}</p> : null}<button type="button" className="ghost-button" onClick={() => setDeleting(editor)}><FontAwesomeIcon icon={faTrashCan} aria-hidden="true" /> {t(rule ? "common.delete" : "resourceSelection.deleteGroup")}</button></div></div>}>
      {rule ? <ResourceRuleEditor value={rule} onChange={rule => patch({ rules: value.rules.map(item => item.id === rule.id ? rule : item) })} /> : null}
      {group ? <ResourceRuleGroupEditor value={group} rules={value.rules} onChange={group => patch({ groups: value.groups.map(item => item.id === group.id ? group : item) })} /> : null}
    </Drawer>, document.body) : null}
    {active && (deleteRule || deleteGroup) ? <ConfirmationDialog title={t(deleteRule ? "resourceSelection.confirmDeleteRule" : "resourceSelection.confirmDeleteGroup")} confirmLabel={t("common.delete")} cancelLabel={t("common.cancel")} pending={false} onConfirm={confirmDelete} onCancel={() => setDeleting(null)}>
      <p>{t(deleteRule ? "resourceSelection.deleteRulePrompt" : "resourceSelection.deleteGroupPrompt", { name: (deleteRule ?? deleteGroup)?.name })}</p>
      {deleteRule && references(deleteRule.id).length ? <p>{t("resourceSelection.deleteReferencedRule", { groups: references(deleteRule.id).map(group => group.name).join("、") })}</p> : null}
      {deleteGroup?.id === value.activeGroup ? <p>{t(nextGroup ? "resourceSelection.deleteActiveNext" : "resourceSelection.deleteActiveLast", { name: nextGroup?.name })}</p> : null}
    </ConfirmationDialog> : null}
  </div>;
}
