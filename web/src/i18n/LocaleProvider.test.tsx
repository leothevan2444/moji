// @vitest-environment jsdom
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Suspense, useState } from "react";
import { beforeEach, describe, expect, it } from "vitest";
import { useTranslation } from "react-i18next";
import { LocaleProvider, useLocale } from "./LocaleProvider";
import { LOCALE_STORAGE_KEY, type LocalePreference } from "./locales";
import "./i18n";

describe("LocaleProvider", () => {
  beforeEach(() => localStorage.clear());

  it("switches immediately, persists preference, and updates document metadata", async () => {
    const user = userEvent.setup();
    function StatefulProbe() {
      const { preference, setPreference } = useLocale();
      const { t } = useTranslation();
      const [count, setCount] = useState(0);
      return <><label>{t("common.language")}<select value={preference} onChange={event=>setPreference(event.target.value as LocalePreference)}><option value="auto">{t("common.auto")}</option><option value="en">English</option></select></label><button type="button" onClick={() => setCount((value) => value + 1)}>state:{count}</button></>;
    }
    render(<Suspense fallback={<span>loading</span>}><LocaleProvider><StatefulProbe /></LocaleProvider></Suspense>);
    await user.click(await screen.findByRole("button", { name: "state:0" }));
    await user.selectOptions(await screen.findByRole("combobox"), "en");
    await waitFor(() => expect(document.documentElement.lang).toBe("en"));
    expect(document.documentElement.dir).toBe("ltr");
    expect(localStorage.getItem(LOCALE_STORAGE_KEY)).toBe("en");
    expect(screen.getByText("Language")).toBeTruthy();
    expect(screen.getByRole("button", { name: "state:1" })).toBeTruthy();
  });
});
