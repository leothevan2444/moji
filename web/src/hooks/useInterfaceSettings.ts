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
  const updateDeveloperMode = useCallback((value: boolean) => {
    generation.current += 1;
    setDeveloperMode(value);
  }, []);
  const updateLanguage = useCallback((value: LocalePreference) => { generation.current += 1; setLanguage(value); }, []);
  useEffect(() => {
    let mounted = true;
    const refresh = async () => {
      const current = ++generation.current;
      const result = await settingsClient.query(InterfaceSettingsDocument, {}, { requestPolicy: "network-only" }).toPromise();
      if (mounted && current === generation.current && result.data) {
        setDeveloperMode(result.data.settings.system.developerMode);
        const value = result.data.settings.system.language;
        if (isLocalePreference(value)) setLanguage(value);
      }
    };
    void refresh();
    window.addEventListener("focus", refresh);
    return () => { mounted = false; window.removeEventListener("focus", refresh); };
  }, []);
  return { developerMode, updateDeveloperMode, language, updateLanguage };
}
