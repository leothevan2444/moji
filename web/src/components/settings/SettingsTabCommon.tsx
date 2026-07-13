import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleInfo } from "@fortawesome/free-solid-svg-icons/faCircleInfo";
import { faEye } from "@fortawesome/free-solid-svg-icons/faEye";
import { faEyeSlash } from "@fortawesome/free-solid-svg-icons/faEyeSlash";
import { useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { describeQueryError } from "../../services/queryError";

export function SettingsLoading({ title }: { title: string }) {
  const { t } = useTranslation();
  return <article className="drawer-card"><div className="drawer-card__head"><h3>{title}</h3></div><p>{t("settings.waiting")}</p></article>;
}

export function SettingsError({ title, error, onRetry }: { title: string; error: unknown; onRetry: () => void }) {
  const { t } = useTranslation();
  return <article className="drawer-card"><div className="drawer-card__head"><h3>{title}</h3></div><p className="settings-feedback tone-danger">{describeQueryError(error as Error)}</p><button type="button" onClick={onRetry}>{t("common.retry")}</button></article>;
}

export function FieldLabel({ text, info }: { text: string; info?: string }) {
  return <span className="settings-field__label"><span>{text}</span>{info ? <span className="settings-info" tabIndex={0} aria-label={info}><FontAwesomeIcon icon={faCircleInfo} aria-hidden="true" /><span className="settings-info__tooltip" role="tooltip">{info}</span></span> : null}</span>;
}

export function SecretInput({ value, onChange, placeholder }: { value: string; onChange: (value: string) => void; placeholder?: string }) {
  const [visible, setVisible] = useState(false);
  return <div className="secret-input"><input className="secret-input__field" type={visible ? "text" : "password"} value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} autoComplete="off" spellCheck={false} /><button type="button" className="secret-input__toggle" onClick={() => setVisible((current) => !current)}><FontAwesomeIcon icon={visible ? faEyeSlash : faEye} aria-hidden="true" /></button></div>;
}

export function SettingsCard({ title, children }: { title: string; children: ReactNode }) {
  return <article className="drawer-card"><div className="drawer-card__head"><h3>{title}</h3></div>{children}</article>;
}
