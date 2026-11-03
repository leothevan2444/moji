import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleInfo } from "@fortawesome/free-solid-svg-icons/faCircleInfo";
import { faEye } from "@fortawesome/free-solid-svg-icons/faEye";
import { faEyeSlash } from "@fortawesome/free-solid-svg-icons/faEyeSlash";
import { useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { describeQueryError } from "../../services/queryError";
import "../../styles/settings-shell.scss";

export function SettingsLoading({ title }: { title: string }) {
  const { t } = useTranslation();
  return <article className="drawer-card"><div className="drawer-card__head"><h3>{title}</h3></div><div className="settings-loading" role="status" aria-label={t("common.loading")}><span className="settings-loading__spinner" aria-hidden="true" /></div></article>;
}

export function SettingsError({ title, error, onRetry }: { title: string; error: unknown; onRetry: () => void }) {
  const { t } = useTranslation();
  return <article className="drawer-card"><div className="drawer-card__head"><h3>{title}</h3></div><p className="settings-feedback tone-danger">{describeQueryError(error as Error)}</p><button type="button" onClick={onRetry}>{t("common.retry")}</button></article>;
}

export function FieldLabel({ text, info }: { text: string; info?: string }) {
  return <span className="settings-field__label"><span>{text}</span>{info ? <span className="settings-info" tabIndex={0} aria-label={info}><FontAwesomeIcon icon={faCircleInfo} aria-hidden="true" /><span className="settings-info__tooltip" role="tooltip">{info}</span></span> : null}</span>;
}

export function SecretInput({ value, onChange, placeholder, id, disabled = false }: { value: string; onChange: (value: string) => void; placeholder?: string; id?: string; disabled?: boolean }) {
  const [visible, setVisible] = useState(false);
  const { t } = useTranslation();
  return <div className="secret-input"><input id={id} disabled={disabled} className="secret-input__field" type={visible ? "text" : "password"} value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} autoComplete="off" spellCheck={false} /><button type="button" disabled={disabled} className="secret-input__toggle" aria-label={t(visible ? "settings.hideSecret" : "settings.showSecret")} onClick={() => setVisible((current) => !current)}><FontAwesomeIcon icon={visible ? faEyeSlash : faEye} aria-hidden="true" /></button></div>;
}

export function SettingsCard({ title, children }: { title: string; children: ReactNode }) {
  return <article className="drawer-card"><div className="drawer-card__head"><h3>{title}</h3></div>{children}</article>;
}

export function SettingsSaveStatus({ state, error, retry }: { state: import("./settingsAutosave").SaveState; error: unknown; retry: () => void }) {
  const { t } = useTranslation();
  if (state === "idle" || state === "saved") return null;
  return <div className={`settings-save-status ${state === "error" ? "tone-danger" : "settings-note"}`} role={state === "error" ? "alert" : "status"} aria-live="polite">
    {t(`settings.autosave.${state}`)}{state === "error" ? <> {describeQueryError(error as Error)} <button type="button" onClick={retry}>{t("common.retry")}</button></> : null}
  </div>;
}
