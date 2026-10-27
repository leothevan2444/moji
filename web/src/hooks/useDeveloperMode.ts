import { useCallback, useEffect, useRef, useState } from "react";
import { settingsClient } from "../graphql/client";
import { DeveloperModeDocument } from "../graphql/generated/graphql";

// The settings route uses its own uncached client. Apply only successful server
// saves here, and fence older reads so they cannot undo a newer saved value.
export function useDeveloperMode() {
  const [enabled, setEnabled] = useState(false);
  const generation = useRef(0);
  const update = useCallback((value: boolean) => {
    generation.current += 1;
    setEnabled(value);
  }, []);
  useEffect(() => {
    let mounted = true;
    const refresh = async () => {
      const current = ++generation.current;
      const result = await settingsClient.query(DeveloperModeDocument, {}, { requestPolicy: "network-only" }).toPromise();
      if (mounted && current === generation.current && result.data) setEnabled(result.data.settings.system.developerMode);
    };
    void refresh();
    window.addEventListener("focus", refresh);
    return () => { mounted = false; window.removeEventListener("focus", refresh); };
  }, []);
  return { enabled, update };
}
