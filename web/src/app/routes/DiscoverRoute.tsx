import { useDeferredValue, useEffect, useState, type FormEvent } from "react";
import { useLocation, useNavigate, useOutletContext, useSearchParams } from "react-router";
import { DiscoveryPage } from "../../pages/DiscoveryPage";
import { Drawer } from "../../components/layout/Drawer";
import { DiscoveryDrawer } from "../../components/drawers/DiscoveryDrawer";
import { SortAndPagination } from "../../components/drawers/SortAndPagination";
import { TorrentSearchSources } from "../../components/discovery/TorrentSearchSources";
import { TorrentSearchResults } from "../../components/discovery/TorrentSearchResults";
import { useDiscoverScenes } from "../../hooks/useDiscoverScenes";
import { useTorrentSearch } from "../../hooks/useTorrentSearch";
import { useTorrentSelectionPreview } from "../../hooks/useTorrentSelectionPreview";
import { useSearchHistory } from "../../hooks/useSearchHistory";
import { parseDiscoverSearchParams, serializeDiscoverSearchParams } from "../searchParams";
import { DISCOVERY_PAGE_SIZE, DISCOVER_SORT_OPTIONS, INDEXER_SORT_OPTIONS } from "../../constants";
import { DiscoverSortBy, TorrentSearchSort, TorrentSearchState, type DiscoverScenesDocumentQuery } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import type { AppOutletContext } from "../AppLayout";
import "../../styles/filter-panel.scss";
import "../../styles/discovery-toolbar.scss";
import "../../styles/torrent-search.scss";
import { useTranslation } from "react-i18next";

const FAST_KEY = "moji.discovery.previewFastRules";
const FILE_KEY = "moji.discovery.previewFileRules";
const stored = (key: string) => { try { return localStorage.getItem(key) === "true"; } catch { return false; } };

export function Component() {
  const { t } = useTranslation(); const navigate = useNavigate(); const location = useLocation();
  const [params, setParams] = useSearchParams(); const state = parseDiscoverSearchParams(params);
  const { pushToast, openHelp } = useOutletContext<AppOutletContext>();
  const [draft, setDraft] = useState(state.q); const [focused, setFocused] = useState(false); const [historyVisible, setHistoryVisible] = useState(false);
  const [pendingAdd, setPendingAdd] = useState<string | null>(null); const history = useSearchHistory();
  useEffect(() => setDraft(state.q), [state.q]);
  const query = useDeferredValue(state.q); const mode = state.source;
  const stashSort = state.sort.toUpperCase().replaceAll("-", "_") as DiscoverSortBy;
  const torrentSort = state.sort.toUpperCase().replaceAll("-", "_") as TorrentSearchSort;
  const fastRules = state.fastRules ?? stored(FAST_KEY); const fileRules = state.fileRules ?? stored(FILE_KEY);
  useEffect(() => { try { localStorage.setItem(FAST_KEY, String(fastRules)); localStorage.setItem(FILE_KEY, String(fileRules)); } catch { /* unavailable */ } }, [fastRules, fileRules]);
  const update = (patch: Partial<typeof state>, replace = true) => {
    const next = serializeDiscoverSearchParams({ ...state, ...patch });
    if (replace) setParams(next, { replace: true }); else navigate(`/discover${next.size ? `?${next}` : ""}`);
  };
  const discovered = useDiscoverScenes(query, { enabled: mode === "stashbox", sortBy: stashSort });
  const runID = mode === "indexers" && state.run ? state.run : null;
  const torrents = useTorrentSearch(runID, (state.page - 1) * DISCOVERY_PAGE_SIZE, DISCOVERY_PAGE_SIZE, torrentSort);
  const preview = useTorrentSelectionPreview();
  const [previewFlags, setPreviewFlags] = useState({ fast: false, file: false, sort: torrentSort });
  useEffect(() => { preview.clear(); }, [runID]);
  const snapshot = torrents.snapshot;
  const running = snapshot?.state === TorrentSearchState.Queued || snapshot?.state === TorrentSearchState.Running;
  const activePreview = preview.value?.searchId === runID && preview.value.searchRevision === snapshot?.revision && preview.value.offset === (state.page - 1) * DISCOVERY_PAGE_SIZE && previewFlags.fast === fastRules && previewFlags.file === fileRules && previewFlags.sort === torrentSort ? preview.value : null;
  const candidates = activePreview?.candidates ?? snapshot?.candidates ?? [];
  const total = mode === "stashbox" ? discovered.results.length : activePreview?.total ?? snapshot?.total ?? 0;
  const pages = Math.max(1, Math.ceil(total / DISCOVERY_PAGE_SIZE));
  const page = state.page;
  const keywords = snapshot?.keywords ?? state.q;
  useEffect(() => { if (mode === "indexers" && snapshot && !running && !torrents.fetching && !activePreview && state.page > pages) update({ page: pages }); }, [mode, snapshot?.id, snapshot?.revision, running, torrents.fetching, state.page, pages, !!activePreview]);

  const beginSearch = async (q: string, refresh = false) => {
    if (!q.trim()) return;
    history.push(q.trim()); setHistoryVisible(false);
    const id = await torrents.start({ keywords: q.trim(), instanceIds: state.instances, refresh });
    if (id) update({ q: q.trim(), source: "indexers", run: id, page: 1 }, false);
  };
  const submit = (event: FormEvent) => { event.preventDefault(); const q = draft.trim(); if (!q) return; if (mode === "indexers") void beginSearch(q); else { history.push(q); setHistoryVisible(false); update({ q, page: 1 }, false); } };
  const applyPreview = async (targetPage = state.page, sort = torrentSort) => {
    if (!runID || running) return false;
    const ok = await preview.preview({ searchId: runID, applyFastRules: fastRules, applyFileRules: fileRules, offset: (targetPage - 1) * DISCOVERY_PAGE_SIZE, limit: DISCOVERY_PAGE_SIZE, sortBy: sort });
    if (ok) setPreviewFlags({ fast: fastRules, file: fileRules, sort }); return ok;
  };
  const move = async (targetPage: number, sort = state.sort) => {
    if (activePreview && !await applyPreview(targetPage, sort.toUpperCase().replaceAll("-", "_") as TorrentSearchSort)) return;
    update({ page: targetPage, sort });
  };
  const queueCandidate = async (id: string, code: string) => {
    const task = await torrents.queue({ candidateId: id, code });
    if (!task) return false;
    navigate(`/tasks/${encodeURIComponent(task.id)}`, { state: { backgroundLocation: location } }); return true;
  };
  const queueScene = async (item: DiscoverScenesDocumentQuery["discoverScenes"]["items"][number]) => {
    setPendingAdd(item.key);
    try { const result = await discovered.queueDiscoveredScene({ input: { sceneId: item.sceneId, stashBoxEndpoint: item.stashBoxEndpoint } }); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); else if (result.data?.queueDiscoveredScene.id) { pushToast("tone-success", t("discoverUi.queued", { title: item.title })); navigate(`/tasks/${encodeURIComponent(result.data.queueDiscoveredScene.id)}`, { state: { backgroundLocation: location } }); } }
    finally { setPendingAdd(null); }
  };
  return <>
    <DiscoveryPage query={draft} searching={mode === "stashbox" ? discovered.fetching : torrents.starting} inputFocused={focused} mode={mode} history={history.history} historyVisible={historyVisible}
      onQueryChange={value => { setDraft(value); setHistoryVisible(!value.trim() && focused); }}
      onInputFocus={() => { setFocused(true); setHistoryVisible(!draft.trim()); }} onInputBlur={() => { setFocused(false); setHistoryVisible(false); }}
      onSubmit={submit} onModeChange={source => update({ source, run: "", sort: "relevance", page: 1 })}
      onPickHistory={q => { setDraft(q); setHistoryVisible(false); if (mode === "indexers") void beginSearch(q); else { history.push(q); update({ q, page: 1 }, false); } }}
      onRemoveHistory={history.remove} onClearHistory={history.clear} onDismissHistory={() => setHistoryVisible(false)} onOpenHelp={openHelp}/>
    {mode === "indexers" && torrents.error && !state.q && !state.run ? <p className="inline-error">{describeQueryError(torrents.error)}</p> : null}
    {state.q || state.run ? <Drawer visibleDrawer="discovery" closing={false} title={t(mode === "stashbox" ? "discoverUi.stashResults" : "torrentSearch.results")} onClose={() => update({ q: "", run: "", page: 1 })}>
      <div className="drawer-stack"><article className="drawer-card"><div className="drawer-card__head"><h3>{t("discoverUi.query", { query: mode === "stashbox" ? query : keywords })}</h3></div>
        {mode === "indexers" ? <>
          <TorrentSearchSources selected={state.instances} disabled={torrents.starting || preview.busy} onChange={instances => update({ instances, run: "", page: 1 })}/>
          <div className="torrent-search__actions"><span role="status">{snapshot ? t(`torrentSearch.states.${snapshot.state}`) : t(torrents.fetching ? "torrentSearch.loading" : "torrentSearch.ready")}</span>
            {running ? <button type="button" className="ghost-button" disabled={!!snapshot?.cancelRequested || torrents.cancelling} onClick={() => void torrents.cancel()}>{t(snapshot?.cancelRequested ? "torrentSearch.cancelling" : "torrentSearch.cancel")}</button> : null}
            <button type="button" disabled={torrents.starting || !keywords.trim()} onClick={() => void beginSearch(keywords, true)}>{t(state.run ? "torrentSearch.searchAgain" : "torrentSearch.search")}</button>
            {state.run ? <button type="button" className="ghost-button" onClick={() => { preview.clear(); torrents.refresh(); }}>{t("torrentSearch.refresh")}</button> : null}
          </div>
          {torrents.error || preview.error ? <p className="inline-error">{describeQueryError(torrents.error ?? preview.error)}</p> : null}
          {snapshot?.errorCode ? <p className="inline-error">{t(`indexerErrors.${snapshot.errorCode}`, { defaultValue: t("indexerErrors.INDEXER_SEARCH_INCOMPLETE") })}</p> : null}
          {snapshot?.truncated ? <p>{t("torrentSearch.truncated")}</p> : null}
          {snapshot?.sites.length ? <details className="torrent-search__sites"><summary>{t("torrentSearch.sites")}</summary><ul>{snapshot.sites.map(site => <li key={site.instanceId}><strong>{site.name}</strong> · {t(`torrentSearch.siteStates.${site.state}`)}{site.cacheHit ? ` · ${t("torrentSearch.cached")}` : ""}{site.fetchedAt ? ` · ${new Date(site.fetchedAt).toLocaleString()}` : ""}{site.errorCode ? <span className="inline-error"> · {t(`indexerErrors.${site.errorCode}`, { defaultValue: t("indexerErrors.INDEXER_REQUEST_FAILED") })}</span> : null}</li>)}</ul></details> : null}
        </> : null}
        <SortAndPagination sortValue={mode === "stashbox" ? stashSort : torrentSort} sortOptions={mode === "stashbox" ? DISCOVER_SORT_OPTIONS : INDEXER_SORT_OPTIONS}
          onSortChange={value => { if (!preview.busy) void move(1, String(value).toLowerCase().replaceAll("_", "-")); }} page={page} totalPages={pages} total={total}
          onPrevPage={() => { if (!preview.busy) void move(Math.max(1, page - 1)); }} onNextPage={() => { if (!preview.busy) void move(Math.min(pages, page + 1)); }}
          extraContent={mode === "indexers" ? <div className="discovery-toolbar__preview">
            <label className="switch-row"><span>{t("discoverUi.fastPreview")}</span><input type="checkbox" checked={fastRules} disabled={preview.busy} onChange={event => update({ fastRules: event.target.checked, page: 1 })}/></label>
            <label className="switch-row"><span>{t("discoverUi.filePreview")}</span><input type="checkbox" checked={fileRules} disabled={preview.busy} onChange={event => update({ fileRules: event.target.checked, page: 1 })}/></label>
            <button type="button" className="ghost-button" disabled={!runID || !snapshot || running || preview.busy || (!fastRules && !fileRules) || snapshot.total === 0} onClick={() => { void applyPreview(1).then(ok => { if (ok) update({ page: 1 }); }); }}>{t(preview.busy ? "discoverUi.inspecting" : "torrentSearch.preview")}</button>
            {activePreview ? <button type="button" className="ghost-button" onClick={preview.clear}>{t("torrentSearch.clearPreview")}</button> : null}
            {fileRules ? <span>{t("torrentSearch.fileCost")}</span> : null}
            {activePreview?.meta.appliedFileRules ? <span>{t("torrentSearch.inspection", { count: activePreview.meta.inspectedCount, limit: activePreview.meta.inspectionCandidateLimit })}</span> : null}
          </div> : null}/>
        {mode === "stashbox" ? <DiscoveryDrawer query={query} searching={discovered.fetching} error={discovered.error ?? null} pendingAddId={pendingAdd}
          discoverItems={discovered.results.slice((page - 1) * DISCOVERY_PAGE_SIZE, page * DISCOVERY_PAGE_SIZE)} hasAnyResults={discovered.results.length > 0} usedStashBoxName={discovered.result?.usedStashBox?.name}
          onQueueDiscovered={item => void queueScene(item)} onTryIndexers={() => void beginSearch(state.q)}/> : <>
          {snapshot && !running && total === 0 ? <p>{t(activePreview ? "torrentSearch.noMatching" : "discoverUi.result.noResults")}</p> : null}
          <TorrentSearchResults key={runID} searchId={runID ?? ""} candidates={candidates} keywords={keywords} pending={torrents.queueing} onQueue={queueCandidate}/>
        </>}
      </article></div>
    </Drawer> : null}
  </>;
}
