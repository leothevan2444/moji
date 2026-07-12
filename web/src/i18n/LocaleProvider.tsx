import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type PropsWithChildren } from "react";
import i18n from "./i18n";
import { applyDocumentLocale, isLocalePreference, LOCALE_STORAGE_KEY, resolveLocale, TEST_LOCALES, type LocaleCode, type LocalePreference, type TestLocale } from "./locales";

export interface LocaleContextValue {
  preference: LocalePreference;
  locale: LocaleCode;
  loadError: boolean;
  setPreference(value: LocalePreference): void;
  retry(): void;
}

const LocaleContext = createContext<LocaleContextValue | null>(null);

function resourceLocale(locale: LocaleCode) {
  return locale === "qps-ploc" ? "en-XA" : locale;
}

function storedPreference(): LocalePreference {
  try {
    const value = localStorage.getItem(LOCALE_STORAGE_KEY);
    return isLocalePreference(value) ? value : "auto";
  } catch { return "auto"; }
}

export function LocaleProvider({ children }: PropsWithChildren) {
  const [preference, setPreferenceState] = useState<LocalePreference>(storedPreference);
  const developmentLocale = import.meta.env.DEV && typeof window !== "undefined"
    ? new URLSearchParams(window.location.search).get("__locale") as TestLocale | null
    : null;
  const testLocale = developmentLocale && developmentLocale in TEST_LOCALES ? developmentLocale : null;
  const initialLocale: LocaleCode = testLocale ?? resolveLocale("auto", typeof navigator === "undefined" ? [] : navigator.languages);
  const [locale, setLocale] = useState<LocaleCode>(initialLocale);
  const activeLocale = useRef(initialLocale);
  const [loadError, setLoadError] = useState(false);
  const [retryVersion, setRetryVersion] = useState(0);
  const browserLanguages = typeof navigator === "undefined" ? [] : navigator.languages;
  const requestedLocale: LocaleCode = testLocale ?? resolveLocale(preference, browserLanguages);

  useEffect(() => {
    let cancelled = false;
    const language = resourceLocale(requestedLocale);
    void i18n.changeLanguage(language).then(() => {
      if (cancelled) return;
      if (!i18n.hasResourceBundle(language, "translation")) {
        setLoadError(true);
        void i18n.changeLanguage(resourceLocale(activeLocale.current));
        return;
      }
      setLoadError(false);
      setLocale(requestedLocale);
      activeLocale.current = requestedLocale;
      applyDocumentLocale(requestedLocale);
    });
    return () => { cancelled = true; };
  }, [requestedLocale, retryVersion]);

  const retry = useCallback(() => {
    i18n.removeResourceBundle(resourceLocale(requestedLocale), "translation");
    setRetryVersion((value) => value + 1);
  }, [requestedLocale]);

  const value = useMemo<LocaleContextValue>(() => ({
    preference, locale, loadError, retry,
    setPreference(next) {
      setPreferenceState(next);
      try { localStorage.setItem(LOCALE_STORAGE_KEY, next); } catch { /* storage is optional */ }
    }
  }), [loadError, locale, preference, retry]);

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>;
}

export function useLocale() {
  const context = useContext(LocaleContext);
  if (!context) throw new Error("useLocale must be used within LocaleProvider");
  return context;
}
