import { createContext, useCallback, useContext, useEffect, useRef, useState, type Dispatch, type PropsWithChildren, type SetStateAction } from "react";
import type { SettingsTab } from "../../types";

interface DraftEntry<T> { value: T; dirty: boolean }
interface DraftStore { read<T>(tab: SettingsTab): DraftEntry<T> | undefined; write<T>(tab: SettingsTab, entry: DraftEntry<T>): void }

const SettingsDraftContext = createContext<DraftStore | null>(null);

export function SettingsDraftProvider({ children }: PropsWithChildren) {
  const drafts = useRef(new Map<SettingsTab, DraftEntry<unknown>>());
  const store = useRef<DraftStore>({
    read: <T,>(tab: SettingsTab) => drafts.current.get(tab) as DraftEntry<T> | undefined,
    write: <T,>(tab: SettingsTab, entry: DraftEntry<T>) => { drafts.current.set(tab, entry as DraftEntry<unknown>); }
  });
  return <SettingsDraftContext.Provider value={store.current}>{children}</SettingsDraftContext.Provider>;
}

export function useSettingsDraft<T>(tab: SettingsTab, initial: T): [T, Dispatch<SetStateAction<T>>, boolean, (value: T) => void] {
  const store = useContext(SettingsDraftContext);
  if (!store) throw new Error("useSettingsDraft must be used inside SettingsDraftProvider");
  const saved = store.read<T>(tab);
  const [value, setValueState] = useState<T>(() => saved?.value ?? initial);
  const [dirty, setDirty] = useState(saved?.dirty ?? false);

  useEffect(() => {
    const current = store.read<T>(tab);
    if (current?.dirty) return;
    setValueState(initial);
    setDirty(false);
    store.write(tab, { value: initial, dirty: false });
  }, [initial, store, tab]);

  const setValue = useCallback<Dispatch<SetStateAction<T>>>((next) => {
    setValueState((current) => {
      const resolved = typeof next === "function" ? (next as (value: T) => T)(current) : next;
      store.write(tab, { value: resolved, dirty: true });
      return resolved;
    });
    setDirty(true);
  }, [store, tab]);

  const markSaved = useCallback((next: T) => {
    store.write(tab, { value: next, dirty: false });
    setValueState(next);
    setDirty(false);
  }, [store, tab]);

  return [value, setValue, dirty, markSaved];
}
