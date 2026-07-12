import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import { DEFAULT_LOCALE, SUPPORTED_LOCALES, TEST_LOCALES } from "./locales";
import lazyBackend from "./lazyBackend";

void i18n.use(lazyBackend).use(initReactI18next).init({
  lng: DEFAULT_LOCALE,
  fallbackLng: ["en", DEFAULT_LOCALE],
  // `en-XA` is the internal BCP 47 test locale used for the public `qps-ploc` mode.
  supportedLngs: [...Object.keys(SUPPORTED_LOCALES), ...Object.keys(TEST_LOCALES).filter((locale) => locale !== "qps-ploc"), "en-XA"],
  ns: ["translation"],
  defaultNS: "translation",
  interpolation: { escapeValue: false },
  returnNull: false,
  saveMissing: import.meta.env.DEV,
  missingKeyHandler(languages: readonly string[], namespace: string, key: string) {
    const detail = { languages, namespace, key };
    console.error("Missing translation", detail);
    if (typeof window !== "undefined") window.dispatchEvent(new CustomEvent("moji:i18n-missing-key", { detail }));
    if (import.meta.env.DEV) throw new Error(`Missing translation: ${namespace}:${key}`);
  }
});

i18n.on("failedLoading", (language, namespace, message) => {
  console.error("Translation resource failed to load", { language, namespace, message });
  if (typeof window !== "undefined") window.dispatchEvent(new CustomEvent("moji:i18n-load-error", { detail: { language, namespace, message } }));
});

export default i18n;
