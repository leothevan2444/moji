import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import { DEFAULT_LOCALE } from "./locales";
import { resources } from "./resources";

void i18n.use(initReactI18next).init({
  resources,
  lng: DEFAULT_LOCALE,
  fallbackLng: ["en", DEFAULT_LOCALE],
  interpolation: { escapeValue: false },
  returnNull: false
});

export default i18n;
