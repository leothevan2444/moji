import { setAppTimezone } from "../utils/appTimezone";
import { useCallback, useEffect, useRef, useState } from "react";
import { isLocalePreference, type LocalePreference } from "../i18n/locales";
import { settingsClient } from "../graphql/client";
import { InterfaceSettingsDocument } from "../graphql/generated/graphql";

// The settings route uses its own uncached client. Apply only successful server
// saves here, and fence older reads so they cannot undo a newer saved value.
export function useInterfaceSettings() {
  const [language, setLanguage] = useState<LocalePreference | null>(null);
  const [developerMode, setDeveloperMode] = useState(false);
  const generation = useRef(0);
  const systemTimezone = useRef("UTC");
  const updateDeveloperMode = useCallback((value: boolean) => {
    generation.current += 1;
    setDeveloperMode(value);
  }, []);
  const updateLanguage = useCallback((value: LocalePreference) => { generation.current += 1; setLanguage(value); }, []);
  const updateTimezone = useCallback((value: string) => {
    const current = ++generation.current;
    setAppTimezone(value === "auto" ? systemTimezone.current : value);
    void settingsClient.query(InterfaceSettingsDocument, {}, { requestPolicy: "network-only" }).toPromise().then(result => {
      if (current === generation.current && result.data) {
        systemTimezone.current = result.data.settings.system.systemTimezone ?? systemTimezone.current;
        setAppTimezone(result.data.settings.system.effectiveTimezone);
      }
    });
  }, []);
  useEffect(() => {
    let mounted = true;
    const refresh = async () => {
      const current = ++generation.current;
      const result = await settingsClient.query(InterfaceSettingsDocument, {}, { requestPolicy: "network-only" }).toPromise();
      if (mounted && current === generation.current && result.data) {
        systemTimezone.current = result.data.settings.system.systemTimezone ?? systemTimezone.current;
        setAppTimezone(result.data.settings.system.effectiveTimezone);
        setDeveloperMode(result.data.settings.system.developerMode);
        const value = result.data.settings.system.language;
        if (isLocalePreference(value)) setLanguage(value);
      }
    };
    void refresh();
    window.addEventListener("focus", refresh);
    return () => { mounted = false; generation.current += 1; window.removeEventListener("focus", refresh); };
  }, []);
  return { updateTimezone, developerMode, updateDeveloperMode, language, updateLanguage };
}
