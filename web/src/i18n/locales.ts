export type SupportedLocale = "zh-CN" | "en";
export type TestLocale = "qps-ploc" | "ar-XB";
export type LocaleCode = SupportedLocale | TestLocale;
export type LocalePreference = "auto" | SupportedLocale;

export const DEFAULT_LOCALE: SupportedLocale = "zh-CN";
export const LOCALE_STORAGE_KEY = "moji:locale";

export const SUPPORTED_LOCALES = {
  "zh-CN": { label: "简体中文", intlLocale: "zh-CN", dir: "ltr" },
  en: { label: "English", intlLocale: "en", dir: "ltr" }
} as const satisfies Record<SupportedLocale, { label: string; intlLocale: string; dir: "ltr" | "rtl" }>;

export const TEST_LOCALES = {
  "qps-ploc": { label: "Pseudo", intlLocale: "en", dir: "ltr" },
  "ar-XB": { label: "RTL test", intlLocale: "en", dir: "rtl" }
} as const satisfies Record<TestLocale, { label: string; intlLocale: string; dir: "ltr" | "rtl" }>;

export function applyDocumentLocale(locale: LocaleCode) {
  const metadata = locale in SUPPORTED_LOCALES
    ? SUPPORTED_LOCALES[locale as SupportedLocale]
    : TEST_LOCALES[locale as TestLocale];
  document.documentElement.lang = locale;
  document.documentElement.dir = metadata.dir;
}

export function normalizeLocale(value?: string | null): SupportedLocale | null {
  const normalized = value?.trim().toLowerCase().replaceAll("_", "-");
  if (!normalized) return null;
  const supported = Object.keys(SUPPORTED_LOCALES) as SupportedLocale[];
  const exact = supported.find((locale) => locale.toLowerCase() === normalized);
  if (exact) return exact;
  const primary = normalized.split("-")[0];
  const compatible = supported.find((locale) => locale.toLowerCase().split("-")[0] === primary);
  if (compatible) return compatible;
  return null;
}

export function resolveLocale(preference: LocalePreference, browserLanguages: readonly string[] = []): SupportedLocale {
  if (preference !== "auto") return preference;
  for (const language of browserLanguages) {
    const locale = normalizeLocale(language);
    if (locale) return locale;
  }
  return DEFAULT_LOCALE;
}

export function isLocalePreference(value: unknown): value is LocalePreference {
  return value === "auto" || (typeof value === "string" && value in SUPPORTED_LOCALES);
}
