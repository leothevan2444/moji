import { describe, expect, it } from "vitest";
import { isLocalePreference, normalizeLocale, resolveLocale } from "./locales";

describe("locale resolution", () => {
  it("normalizes supported language variants", () => {
    expect(normalizeLocale("zh-Hans-CN")).toBe("zh-CN");
    expect(normalizeLocale("en-US")).toBe("en");
    expect(normalizeLocale("fr-FR")).toBeNull();
  });

  it("uses an explicit preference before browser languages", () => {
    expect(resolveLocale("en", ["zh-CN"])).toBe("en");
    expect(resolveLocale("auto", ["fr-FR", "en-GB"])).toBe("en");
    expect(resolveLocale("auto", ["fr-FR"])).toBe("zh-CN");
  });

  it("rejects damaged persisted values", () => {
    expect(isLocalePreference("auto")).toBe(true);
    expect(isLocalePreference("zh")).toBe(false);
    expect(isLocalePreference(null)).toBe(false);
  });
});
