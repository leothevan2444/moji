export type SupportedLocale = "zh-CN" | "en";
export type LocalePreference = "auto" | SupportedLocale;

export const DEFAULT_LOCALE: SupportedLocale = "zh-CN";
export const LOCALE_STORAGE_KEY = "moji:locale";

export const SUPPORTED_LOCALES = {
  "zh-CN": { label: "简体中文", intlLocale: "zh-CN", dir: "ltr" },
  en: { label: "English", intlLocale: "en", dir: "ltr" }
} as const satisfies Record<SupportedLocale, { label: string; intlLocale: string; dir: "ltr" | "rtl" }>;

export function normalizeLocale(value?: string | null): SupportedLocale | null {
  const normalized = value?.trim().toLowerCase().replaceAll("_", "-");
  if (!normalized) return null;
  if (normalized === "en" || normalized.startsWith("en-")) return "en";
  if (normalized === "zh" || normalized.startsWith("zh-")) return "zh-CN";
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
  return value === "auto" || value === "zh-CN" || value === "en";
}
