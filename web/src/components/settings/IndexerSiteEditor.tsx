import { parseIndexerTrustedOrigins } from "./indexerTrustedOrigins";
import { useIndexerConnections } from "../../hooks/useIndexerConnections";
import { IndexerConnectionButton } from "./IndexerConnectionButton";
import { useState, type FormEvent } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { useBeforeUnload, useBlocker } from "react-router";
import type { BuiltinIndexerInput, BuiltinIndexerPatchInput, IndexerSettingChangeInput } from "../../graphql/generated/graphql";
import type { IndexerSite, SiteSetting } from "../../hooks/useIndexerManagement";
import { Drawer } from "../layout/Drawer";
import { SettingsSwitch } from "./SettingsSwitch";
import { SecretInput } from "./SettingsTabCommon";
import { describeQueryError } from "../../services/queryError";

export interface SiteEditorDefinition { id: string; name: string; links: string[]; settings: SiteSetting[] }
export function IndexerSiteEditor({ site, definition, revision, profiles, busy, error, onSave, onClose, connectionPending = false }: {
  connectionPending?: boolean; site?: IndexerSite; definition: SiteEditorDefinition; revision: string; profiles: string[]; busy: boolean; error: unknown;
  onSave: (id: string | null, revision: string, input: BuiltinIndexerInput | BuiltinIndexerPatchInput) => Promise<boolean>; onClose: () => void;
}) {
  const { t } = useTranslation();
  const [name, setName] = useState(site?.name ?? definition.name);
  const [baseUrl, setBaseUrl] = useState(site?.baseUrl ?? "");
  const [profile, setProfile] = useState(site?.networkProfile || "direct");
  const [enabled, setEnabled] = useState(site?.enabled ?? false);
  const initialOrigins = (site?.trustedOrigins ?? []).join("\n");
  const [originsText, setOriginsText] = useState(initialOrigins);
  const origins = parseIndexerTrustedOrigins(originsText);
  const [changes, setChanges] = useState<Record<string, IndexerSettingChangeInput>>({});
  const [discard, setDiscard] = useState(false);
  const dirty = originsText !== initialOrigins || Object.keys(changes).length > 0 || name !== (site?.name ?? definition.name) || baseUrl !== (site?.baseUrl ?? "") || profile !== (site?.networkProfile || "direct") || enabled !== (site?.enabled ?? false);
  const blocker = useBlocker(dirty || busy);
  useBeforeUnload(event => { if (dirty || busy) { event.preventDefault(); event.returnValue = ""; } });
  const keepEditing = () => { setDiscard(false); if (blocker.state === "blocked") blocker.reset(); };
  const discardEdits = () => { if (blocker.state === "blocked") blocker.proceed(); else onClose(); };
  const close = () => { if (busy) return; if (dirty) setDiscard(true); else onClose(); };
  const change = (setting: SiteSetting, value: string | boolean) => setChanges(current => ({ ...current, [setting.name]: { name: setting.name, value: typeof value === "boolean" ? { boolean: value } : { string: value } } }));
  const draftPatch = () => {
    const patch: BuiltinIndexerPatchInput = {};
    if (!site || name !== site.name) patch.name = name;
    if (!site || baseUrl !== site.baseUrl) patch.baseUrl = baseUrl;
    if (!site || profile !== site.networkProfile) patch.networkProfile = profile;
    if (!site || enabled !== site.enabled) patch.enabled = enabled;
    if (originsText !== initialOrigins) patch.trustedOrigins = origins.origins;
    if (Object.keys(changes).length) patch.settings = Object.values(changes);
    return patch;
  };
  // Draft settings (including secrets) are scoped in memory only.
  const checks = useIndexerConnections(revision, JSON.stringify({ revision, patch: draftPatch() }));
  const submit = async (event:FormEvent) => {
    event.preventDefault();if(busy || origins.error)return;const patch=draftPatch();
    if (await onSave(site?.id ?? null, revision, site ? patch : { ...patch, definitionId: definition.id })) onClose();
  };
  return createPortal(<>
    <Drawer visibleDrawer="settings" closing={false} title={site ? t("indexers.editSite") : t("indexers.addSite")} closeDisabled={busy} onClose={close} footer={<div className="indexer-actions"><button className="primary-button" type="submit" form="indexer-site-form" disabled={busy || Boolean(origins.error) || Boolean(site && !dirty)}>{t(busy ? "indexers.saving" : site ? "indexers.save" : "indexers.add")}</button><IndexerConnectionButton result={checks.results.draft} disabled={busy || connectionPending || checks.busy || Boolean(origins.error)} onClick={() => void checks.run(["draft"], false, { instanceId: site?.id, definitionId: site ? undefined : definition.id, draft: draftPatch() })} /></div>}>
      <form id="indexer-site-form" className="settings-form indexer-editor" onSubmit={event => void submit(event)}>
        {error ? <p role="alert" className="settings-feedback tone-danger">{describeQueryError(error)}</p> : null}
        <label className="settings-field"><span>{t("indexers.name")}</span><input value={name} disabled={busy} maxLength={256} onChange={event => setName(event.target.value)} /></label>
        <label className="settings-field"><span>{t("indexers.baseUrl")}</span><input type="url" value={baseUrl} disabled={busy} placeholder={definition.links[0] ?? "https://"} onChange={event => setBaseUrl(event.target.value)} /></label>
        <label className="settings-field"><span>{t("indexers.network")}</span><select value={profile} disabled={busy} onChange={event => setProfile(event.target.value)}>{profiles.map(value => <option key={value} value={value}>{value === "direct" ? t("indexers.direct") : value}</option>)}</select></label>
        {definition.settings.map(setting => {
          const edit = changes[setting.name]; const value = edit?.clear && setting.type === "select" ? setting.defaultValue : edit?.value ?? setting.value ?? setting.defaultValue;
          const text = edit?.clear && setting.type !== "select" ? "" : value?.string ?? (value?.number === null || value?.number === undefined ? "" : String(value.number));
          if (setting.type === "info") return <p className="settings-note" key={setting.name}>{setting.label}</p>;
          if (setting.type === "checkbox") return <label className="settings-toggle" key={setting.name}><input type="checkbox" disabled={busy} checked={value?.boolean ?? false} onChange={event => change(setting, event.target.checked)} /><span>{setting.label || setting.name}</span></label>;
          return <div className="settings-field" key={setting.name}><label htmlFor={`site-field-${setting.name}`}>{setting.label || setting.name}</label>
            {setting.type === "unknown" ? <span>{t("indexers.removedSetting")}</span> : setting.secret ? <SecretInput id={`site-field-${setting.name}`} disabled={busy} value={edit?.value?.string ?? ""} placeholder={setting.configured && !edit?.clear ? t("indexers.secretSaved") : ""} onChange={value => change(setting, value)} />
              : setting.type === "select" ? <select id={`site-field-${setting.name}`} disabled={busy} value={text} onChange={event => change(setting, event.target.value)}>{text === "" && !setting.options.some(option => option.value === "") ? <option value="" disabled>{t("indexers.noSelection")}</option> : null}{setting.options.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}</select>
              : <input id={`site-field-${setting.name}`} disabled={busy} value={text} onChange={event => change(setting, event.target.value)} />}
            {setting.configured ? <button type="button" className="ghost-button" disabled={busy || Boolean(edit?.clear)} onClick={() => setChanges(current => ({ ...current, [setting.name]: { name: setting.name, clear: true } }))}>{t(edit?.clear ? "indexers.willClear" : "indexers.clearSetting")}</button> : null}
          </div>;
        })}
        <details className="indexer-editor__advanced">
          <summary>{t("indexers.advanced")}</summary>
          <label className="settings-field"><span>{t("indexers.trustedOrigins")}</span><textarea value={originsText} disabled={busy} rows={3} maxLength={66000} spellCheck={false} autoCapitalize="none" autoComplete="off" aria-invalid={Boolean(origins.error)} aria-describedby={origins.error ? "indexer-origins-help indexer-origins-error" : "indexer-origins-help"} onChange={event => setOriginsText(event.target.value)} /></label>
          <p id="indexer-origins-help" className="settings-note">{t("indexers.trustedOriginsHint")}</p>
          {origins.error ? <p id="indexer-origins-error" role="alert" className="settings-feedback tone-danger">{t(`indexers.${origins.error}`)}</p> : null}
        </details>
        <label className="switch-row"><span>{t("indexers.enable")}</span><SettingsSwitch checked={enabled} disabled={busy} label={t("indexers.enable")} onChange={setEnabled} /></label>
      </form>
    </Drawer>
    {discard || blocker.state === "blocked" ? <Drawer visibleDrawer="confirm" closing={false} title={t("indexers.discardTitle")} onClose={keepEditing} footer={<button disabled={busy} onClick={discardEdits}>{t("indexers.discard")}</button>}><p>{t("indexers.unsaved")}</p><button disabled={busy} onClick={keepEditing}>{t("indexers.keepEditing")}</button></Drawer> : null}
  </>, document.body);
}
