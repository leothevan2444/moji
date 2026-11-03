import { useCallback, useContext, useEffect, useRef, useSyncExternalStore } from "react";
import { useMutation } from "urql";
import { useTranslation } from "react-i18next";
import { PatchSettingsDocument, type SettingsPatchInput } from "../../graphql/generated/graphql";
import { SettingsAutosaveContext } from "./SettingsAutosaveProvider";
import { SettingsAutosave, type SettingsObject } from "./settingsAutosave";

interface Options<T> {
  persist?: (patch: SettingsObject, sourceRevision?: string) => Promise<string | void>;
  sourceRevision?: string;
  onSaved?: (value: T, patch: SettingsObject) => void;
}
export function useSettingsAutosave<T>(key: string, initial: T, encode: (value: T) => SettingsObject, options: Options<T> = {}) {
  const store = useContext(SettingsAutosaveContext);
  if (!store) throw new Error("Settings autosave requires SettingsAutosaveProvider");
  const [, mutate] = useMutation(PatchSettingsDocument);
  const { t } = useTranslation();
  const entry = useRef<SettingsAutosave<T> | null>(null);
  if (!entry.current) {
    const existing = store.get(key) as SettingsAutosave<T> | undefined;
    entry.current = existing ?? new SettingsAutosave(initial, encode, async () => {}, options.sourceRevision);
    store.set(key, entry.current as SettingsAutosave<unknown>);
  }
  const current = entry.current;
  current.encode = encode;
  current.persist = async patch => {
    const valid = (value: unknown): boolean => typeof value === "number" ? Number.isFinite(value) && Number.isInteger(value) : value !== null && typeof value === "object" ? Object.values(value).every(valid) : true;
    // Floating point cleanup thresholds are intentionally supported.
    const finite = (value: unknown): boolean => typeof value === "number" ? Number.isFinite(value) : value !== null && typeof value === "object" ? Object.values(value).every(finite) : true;
    if (!(key === "downloader:policy" ? finite(patch) : valid(patch))) throw new Error(t("settings.autosave.invalid"));
    if (options.persist) {
      const revision = await options.persist(patch, current.sourceRevision);
      if (revision !== undefined) current.sourceRevision = revision;
      return;
    }
    const result = await mutate({ input: patch as SettingsPatchInput });
    if (result.error) throw result.error;
    if (!result.data) throw new Error(t("settings.autosave.noAcknowledgement"));
  };
  current.onSaved = options.onSaved;
  const snapshot = useSyncExternalStore(current.subscribe, current.getSnapshot);
  // Initial query snapshots may rerender as object identities change. Compare
  // serialized data and never reinitialize on autosave state transitions.
  const serialized = JSON.stringify(initial);
  useEffect(() => { current.initialize(JSON.parse(serialized) as T, options.sourceRevision); }, [current, serialized, options.sourceRevision]);
  const setValue = useCallback((next: T | ((value: T) => T)) => current.edit(next), [current]);
  const discard = useCallback((value: T, revision?: string) => current.discard(value, revision), [current]);
  return { ...snapshot, setValue, retry: current.retry, discard };
}
