// @vitest-environment jsdom
import { describe, expect, it } from "vitest";
import { resources } from "./resources";
import { pseudoLocalize } from "./pseudo";
import { applyDocumentLocale } from "./locales";

function flatten(value: unknown, prefix = "", result = new Map<string, string>()) {
  if (typeof value === "string") {
    result.set(prefix, value);
  } else if (value && typeof value === "object") {
    for (const [key, item] of Object.entries(value)) flatten(item, prefix ? `${prefix}.${key}` : key, result);
  }
  return result;
}

function canonicalKey(key: string) {
  return key.replace(/_(zero|one|two|few|many|other)$/u, "");
}

function variables(message: string) {
  return [...message.matchAll(/\{\{\s*([^},\s]+)[^}]*\}\}/g)].map((match) => match[1]).sort();
}

describe("translation resource integrity", () => {
  const zh = flatten(resources["zh-CN"].translation);
  const en = flatten(resources.en.translation);

  it("keeps official locale key sets and interpolation variables aligned", () => {
    const zhKeys = new Set([...zh.keys()].map(canonicalKey));
    const enKeys = new Set([...en.keys()].map(canonicalKey));
    expect(zhKeys).toEqual(enKeys);
    for (const key of zhKeys) {
      const zhValue = zh.get(key);
      const enValue = en.get(key);
      expect(zhValue, `${key} must be non-empty in zh-CN`).toBeTruthy();
      expect(enValue, `${key} must be non-empty in en`).toBeTruthy();
      expect(variables(zhValue!), `${key} variables`).toEqual(variables(enValue!));
    }
  });

  it("pseudo-localizes copy while preserving interpolation and markup", () => {
    const result = pseudoLocalize("Hello {{name}} <strong>world</strong>");
    expect(result).toContain("{{name}}");
    expect(result).toContain("<strong>");
    expect(result).not.toBe("Hello {{name}} <strong>world</strong>");
  });

  it("applies RTL metadata for the development test locale", () => {
    applyDocumentLocale("ar-XB");
    expect(document.documentElement.dir).toBe("rtl");
    applyDocumentLocale("en");
    expect(document.documentElement.dir).toBe("ltr");
  });
});
