import type { BackendModule, ReadCallback } from "i18next";
import { pseudoResource } from "./pseudo";

const lazyBackend: BackendModule = {
  type: "backend",
  init() {},
  read(language: string, namespace: string, callback: ReadCallback) {
    const isPseudoLocale = language.toLowerCase() === "en-xa";
    const sourceLanguage = isPseudoLocale || language === "ar-XB" ? "en" : language;
    if (import.meta.env.MODE === "test") {
      void import("./resources").then(({ resources }) => {
        const resource = resources[sourceLanguage as keyof typeof resources]?.translation;
        if (!resource) { callback(new Error(`Unknown test locale: ${language}`), false); return; }
        callback(null, isPseudoLocale ? pseudoResource(resource) : resource);
      }).catch((error: unknown) => callback(error instanceof Error ? error : new Error(String(error)), false));
      return;
    }
    void fetch(`/locales/${encodeURIComponent(sourceLanguage)}/${encodeURIComponent(namespace)}.json`, { credentials: "same-origin" })
      .then(async (response) => {
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const resource = await response.json() as Record<string, unknown>;
        callback(null, isPseudoLocale ? pseudoResource(resource) : resource);
      })
      .catch((error: unknown) => callback(error instanceof Error ? error : new Error(String(error)), false));
  }
};

export default lazyBackend;
