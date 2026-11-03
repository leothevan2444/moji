import { useEffect, useRef, type PropsWithChildren } from "react";
import { useIndexerEvents } from "../../hooks/useIndexerEvents";
import { IndexerEventKind } from "../../graphql/generated/graphql";
import { IndexerLibraryContext, IndexerManagementContext, useIndexerLibraryState, useIndexerManagementState } from "../../hooks/useIndexerManagement";

export function IndexerManagementProvider({ children }: PropsWithChildren) {
  const management = useIndexerManagementState(); const library = useIndexerLibraryState();
  const events = useIndexerEvents(management.refresh, [IndexerEventKind.Configuration, IndexerEventKind.Catalog, IndexerEventKind.Site]);
  const seen = useRef("");
  useEffect(() => {
    if (!library.status || !management.snapshot) return;
    const key = `${library.status.streamId}:${library.status.activeRevision}:${library.status.available}`;
    if (seen.current === key) return;
    seen.current = key;
    if (management.snapshot.ruleRevision !== library.status.activeRevision || (library.status.available && management.snapshot.ruleErrorCode)) management.refresh();
  }, [library.status, management.snapshot, management.refresh]);
  return <IndexerManagementContext.Provider value={{...management,error:management.error ?? events.error,refresh:()=>{management.refresh();if(events.error)events.retry();}}}><IndexerLibraryContext.Provider value={library}>{children}</IndexerLibraryContext.Provider></IndexerManagementContext.Provider>;
}
