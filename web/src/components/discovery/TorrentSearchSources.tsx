import { useCallback } from "react";
import { useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { IndexerEventKind, TorrentSearchSourcesDocument } from "../../graphql/generated/graphql";
import { useIndexerEvents } from "../../hooks/useIndexerEvents";
import { describeQueryError } from "../../services/queryError";
export function TorrentSearchSources({ selected, onChange, disabled }: { selected: string[]; onChange: (ids: string[]) => void; disabled: boolean }) {
 const {t}=useTranslation();const [query,refresh]=useQuery({query:TorrentSearchSourcesDocument,requestPolicy:"network-only"});
 const read=useCallback(()=>refresh({requestPolicy:"network-only"}),[refresh]);const events=useIndexerEvents(read,[IndexerEventKind.Configuration,IndexerEventKind.Catalog]);
 const items=query.data?.indexerInstances.items??[];
 return <details className="filter-panel"><summary>{t("discoverUi.filters.title")}{selected.length?` (${selected.length})`:""}</summary><div className="filter-panel__body">
 {(query.error || events.error)?<p className="inline-error">{describeQueryError(query.error??events.error)}</p>:null}
 {items.length ? <div className="filter-panel__chips">{items.map(item=><button key={item.id} type="button" className={`filter-panel__chip${selected.includes(item.id)?" is-active":""}`} disabled={disabled || !item.enabled || !!item.errorCode} aria-pressed={selected.includes(item.id)} onClick={()=>onChange(selected.includes(item.id)?selected.filter(id=>id!==item.id):[...selected,item.id])}>{item.name}{item.kind==="JACKETT"?" · Jackett":""}</button>)}</div> : <p>{t(query.fetching?"discoverUi.filters.loading":"torrentSearch.noSources")}</p>}
 {selected.length?<button type="button" className="filter-panel__clear" disabled={disabled} onClick={()=>onChange([])}>{t("discoverUi.filters.clear")}</button>:null}
 </div></details>;
}
