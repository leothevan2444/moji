import { createContext, useContext, useEffect, useMemo, useState, type PropsWithChildren } from "react";
import i18n from "./i18n";
import { isLocalePreference, LOCALE_STORAGE_KEY, resolveLocale, SUPPORTED_LOCALES, type LocalePreference, type SupportedLocale } from "./locales";

export interface LocaleContextValue {
  preference: LocalePreference;
  locale: SupportedLocale;
  setPreference(value: LocalePreference): void;
}

const LocaleContext = createContext<LocaleContextValue | null>(null);

function storedPreference(): LocalePreference {
  try {
    const value = localStorage.getItem(LOCALE_STORAGE_KEY);
    return isLocalePreference(value) ? value : "auto";
  } catch { return "auto"; }
}

export function LocaleProvider({ children }: PropsWithChildren) {
  const [preference, setPreferenceState] = useState<LocalePreference>(storedPreference);
  const browserLanguages = typeof navigator === "undefined" ? [] : navigator.languages;
  const locale = resolveLocale(preference, browserLanguages);

  useEffect(() => {
    void i18n.changeLanguage(locale);
    document.documentElement.lang = locale;
    document.documentElement.dir = SUPPORTED_LOCALES[locale].dir;
  }, [locale]);

  const value = useMemo<LocaleContextValue>(() => ({
    preference, locale,
    setPreference(next) {
      setPreferenceState(next);
      try { localStorage.setItem(LOCALE_STORAGE_KEY, next); } catch { /* storage is optional */ }
    }
  }), [locale, preference]);

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>;
}

export function useLocale() {
  const context = useContext(LocaleContext);
  if (!context) throw new Error("useLocale must be used within LocaleProvider");
  return context;
}
