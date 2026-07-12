import { useTranslation } from "react-i18next";
import { SUPPORTED_LOCALES, type LocalePreference } from "../../i18n/locales";
import { useLocale } from "../../i18n/LocaleProvider";

export function LocaleSelect() {
  const { t } = useTranslation();
  const { preference, loadError, retry, setPreference } = useLocale();
  return <div className="locale-select"><label>
    <span>{t("common.language")}</span>
    <select value={preference} onChange={(event) => setPreference(event.target.value as LocalePreference)}>
      <option value="auto">{t("common.auto")}</option>
      {Object.entries(SUPPORTED_LOCALES).map(([value, meta]) => <option key={value} value={value}>{meta.label}</option>)}
    </select>
  </label>{loadError ? <span role="alert">{t("localeUi.loadFailed")} <button type="button" className="ghost-button" onClick={retry}>{t("localeUi.retry")}</button></span> : null}</div>;
}
