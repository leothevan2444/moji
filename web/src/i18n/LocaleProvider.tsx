import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type PropsWithChildren } from "react";
import i18n from "./i18n";
import { applyDocumentLocale, isLocalePreference, LOCALE_STORAGE_KEY, resolveLocale, type LocalePreference, type SupportedLocale } from "./locales";

export interface LocaleContextValue {
  preference: LocalePreference;
  locale: SupportedLocale;
  loadError: boolean;
  setPreference(value: LocalePreference): void;
  retry(): void;
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
  const initialLocale = resolveLocale("auto", typeof navigator === "undefined" ? [] : navigator.languages);
  const [locale, setLocale] = useState<SupportedLocale>(initialLocale);
  const activeLocale = useRef(initialLocale);
  const [loadError, setLoadError] = useState(false);
  const [retryVersion, setRetryVersion] = useState(0);
  const browserLanguages = typeof navigator === "undefined" ? [] : navigator.languages;
  const requestedLocale = resolveLocale(preference, browserLanguages);

  useEffect(() => {
    let cancelled = false;
    void i18n.changeLanguage(requestedLocale).then(() => {
      if (cancelled) return;
      if (!i18n.hasResourceBundle(requestedLocale, "translation")) {
        setLoadError(true);
        void i18n.changeLanguage(activeLocale.current);
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
    i18n.removeResourceBundle(requestedLocale, "translation");
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
