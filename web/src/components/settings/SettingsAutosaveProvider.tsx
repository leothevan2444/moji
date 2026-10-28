import { createContext, useEffect, useRef, type PropsWithChildren } from "react";
import type { SettingsAutosave } from "./settingsAutosave";

export const SettingsAutosaveContext = createContext<Map<string, SettingsAutosave<unknown>> | null>(null);

export function SettingsAutosaveProvider({children}:PropsWithChildren) {
  const entries=useRef(new Map<string,SettingsAutosave<unknown>>());
  useEffect(()=>{
    const beforeUnload=(event:BeforeUnloadEvent)=>{
      if([...entries.current.values()].some(entry=>["pending","saving","error"].includes(entry.snapshot.state))){event.preventDefault();event.returnValue="";}
    };
    window.addEventListener("beforeunload",beforeUnload);
    return ()=>window.removeEventListener("beforeunload",beforeUnload);
  },[]);
  // Queues own their timers and remain alive until outstanding work settles,
  // including when a tab or the settings route unmounts.
  return <SettingsAutosaveContext.Provider value={entries.current}>{children}</SettingsAutosaveContext.Provider>;
}
