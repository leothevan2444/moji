import { useTranslation } from "react-i18next";
import { SUPPORTED_LOCALES, type LocalePreference } from "../../i18n/locales";
import { useLocale } from "../../i18n/LocaleProvider";

export function LocaleSelect() {
  const { t } = useTranslation();
  const { preference, setPreference } = useLocale();
  return <label className="locale-select">
    <span>{t("common.language")}</span>
    <select value={preference} onChange={(event) => setPreference(event.target.value as LocalePreference)}>
      <option value="auto">{t("common.auto")}</option>
      {Object.entries(SUPPORTED_LOCALES).map(([value, meta]) => <option key={value} value={value}>{meta.label}</option>)}
    </select>
  </label>;
}
