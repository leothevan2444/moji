import { useDeferredValue, useEffect, useMemo, useState, type FormEvent } from "react";
import { useLocation, useNavigate, useOutletContext, useSearchParams } from "react-router";
import { useQuery } from "urql";
import { DiscoveryPage } from "../../pages/DiscoveryPage";
import { Drawer } from "../../components/layout/Drawer";
import { DiscoveryDrawer } from "../../components/drawers/DiscoveryDrawer";
import { JackettFilterPanel } from "../../components/drawers/JackettFilterPanel";
import { SortAndPagination } from "../../components/drawers/SortAndPagination";
import { useDiscoverScenes } from "../../hooks/useDiscoverScenes";
import { useJackettSearch } from "../../hooks/useJackettSearch";
import { useJackettIndexers } from "../../hooks/useJackettIndexers";
import { usePreviewJackettSelection } from "../../hooks/usePreviewJackettSelection";
import { useSearchHistory } from "../../hooks/useSearchHistory";
import { useTaskMutations } from "../../hooks/useTaskMutations";
import { parseDiscoverSearchParams, serializeDiscoverSearchParams } from "../searchParams";
import { DISCOVERY_PAGE_SIZE, DISCOVER_SORT_OPTIONS, JACKETT_SORT_OPTIONS } from "../../constants";
import { DiscoverConfigDocumentDocument, DiscoverSortBy, JackettSortBy, type DiscoverScenesDocumentQuery, type SearchDocumentQuery } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import type { AppOutletContext } from "../AppLayout";

const FAST_KEY = "moji.discovery.previewFastRules";
const FILE_KEY = "moji.discovery.previewFileRules";
const stored = (key: string) => { try { return localStorage.getItem(key) === "true"; } catch { return false; } };

export function Component() {
  const navigate = useNavigate();
  const location = useLocation();
  const [params, setParams] = useSearchParams();
  const state = parseDiscoverSearchParams(params);
  const { pushToast, openHelp } = useOutletContext<AppOutletContext>();
  const [draft, setDraft] = useState(state.q);
  const [focused, setFocused] = useState(false);
  const [historyVisible, setHistoryVisible] = useState(false);
  const [pendingAdd, setPendingAdd] = useState<string | null>(null);
  const history = useSearchHistory();
  useEffect(() => setDraft(state.q), [state.q]);
  const query = useDeferredValue(state.q);
  const mode = state.source;
  const stashSort = state.sort.toUpperCase().replaceAll("-", "_") as DiscoverSortBy;
  const jackettSort = state.sort.toUpperCase().replaceAll("-", "_") as JackettSortBy;
  const fastRules = state.fastRules ?? stored(FAST_KEY);
  const fileRules = state.fileRules ?? stored(FILE_KEY);
  useEffect(() => { try { localStorage.setItem(FAST_KEY, String(fastRules)); localStorage.setItem(FILE_KEY, String(fileRules)); } catch { /* unavailable */ } }, [fastRules, fileRules]);
  const update = (patch: Partial<typeof state>, replace = true) => {
    const next = serializeDiscoverSearchParams({ ...state, ...patch });
    if (replace) setParams(next, { replace: true }); else navigate(`/discover${next.size ? `?${next}` : ""}`);
  };

  const [{ data: config }] = useQuery({ query: DiscoverConfigDocumentDocument, requestPolicy: "cache-and-network" });
  const { result: discoveredResult, results: discovered, fetching: discovering, error: discoverError, queueDiscoveredScene } = useDiscoverScenes(query, { enabled: mode === "stashbox", sortBy: stashSort });
  const { results: jackett, fetching: searching, error: searchError } = useJackettSearch(query, { enabled: mode === "jackett", trackers: state.trackers, sortBy: jackettSort });
  const { results: preview, previewMeta, fetching: previewing, error: previewError } = usePreviewJackettSelection(query, jackett, { enabled: mode === "jackett", applyFastRules: fastRules, applyFileRules: fileRules, inspectionCandidateLimit: config?.settings.automation.torrentSelection.inspectionCandidateLimit ?? 5 });
  const { indexers, fetching: fetchingIndexers } = useJackettIndexers(mode === "jackett");
  const { addTorrent } = useTaskMutations();
  const activeJackett = useMemo(() => (!fastRules && !fileRules) || preview.length === 0 ? jackett : preview, [fastRules, fileRules, jackett, preview]);
  const visible = mode === "stashbox" ? discovered : activeJackett;
  const pages = Math.max(1, Math.ceil(visible.length / DISCOVERY_PAGE_SIZE));
  const page = Math.min(state.page, pages);
  const slice = <T,>(items: T[]) => items.slice((page - 1) * DISCOVERY_PAGE_SIZE, page * DISCOVERY_PAGE_SIZE);

  const submit = (event: FormEvent) => { event.preventDefault(); const q = draft.trim(); if (!q) return; history.push(q); setHistoryVisible(false); update({ q, page: 1 }, false); };
  const addJackett = async (item: SearchDocumentQuery["jackettSearch"][number]) => {
    setPendingAdd(item.link);
    try { const result = await addTorrent({ input: { url: item.magnetUri || item.link } }); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); else if (result.data?.addTorrent.id) navigate(`/tasks/${encodeURIComponent(result.data.addTorrent.id)}`, { state: { backgroundLocation: location } }); }
    finally { setPendingAdd(null); }
  };
  const queueScene = async (item: DiscoverScenesDocumentQuery["discoverScenes"]["items"][number]) => {
    setPendingAdd(item.key);
    try { const result = await queueDiscoveredScene({ input: { sceneId: item.sceneId, stashBoxEndpoint: item.stashBoxEndpoint } }); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); else if (result.data?.queueDiscoveredScene.id) { pushToast("tone-success", `已将 ${item.title} 加入任务队列。`); navigate(`/tasks/${encodeURIComponent(result.data.queueDiscoveredScene.id)}`, { state: { backgroundLocation: location } }); } }
    finally { setPendingAdd(null); }
  };

  return <>
    <DiscoveryPage query={draft} searching={mode === "stashbox" ? discovering : searching} inputFocused={focused} mode={mode} history={history.history} historyVisible={historyVisible}
      onQueryChange={(value) => { setDraft(value); setHistoryVisible(!value.trim() && focused); }}
      onInputFocus={() => { setFocused(true); setHistoryVisible(!draft.trim()); }} onInputBlur={() => { setFocused(false); setHistoryVisible(false); }}
      onSubmit={submit} onModeChange={(source) => update({ source, sort: "relevance", page: 1 })}
      onPickHistory={(q) => { setDraft(q); history.push(q); setHistoryVisible(false); update({ q, page: 1 }, false); }}
      onRemoveHistory={history.remove} onClearHistory={history.clear} onDismissHistory={() => setHistoryVisible(false)} onOpenHelp={openHelp} />
    {state.q ? <Drawer visibleDrawer="discovery" closing={false} title={mode === "stashbox" ? "StashBox 搜索结果" : "Jackett 搜索结果"} onClose={() => update({ q: "", page: 1 })}>
      <div className="drawer-stack"><article className="drawer-card"><div className="drawer-card__head"><h3>搜索词：{query}</h3></div>
        {mode === "jackett" ? <JackettFilterPanel indexers={indexers} fetching={fetchingIndexers} enabledIds={state.trackers} onToggle={(id) => update({ trackers: state.trackers.includes(id) ? state.trackers.filter((x) => x !== id) : [...state.trackers, id], page: 1 })} onClear={() => update({ trackers: [], page: 1 })} /> : null}
        <SortAndPagination sortValue={mode === "stashbox" ? stashSort : jackettSort} sortOptions={mode === "stashbox" ? DISCOVER_SORT_OPTIONS : JACKETT_SORT_OPTIONS}
          onSortChange={(value) => update({ sort: String(value).toLowerCase().replaceAll("_", "-"), page: 1 })} page={page} totalPages={pages} total={visible.length}
          onPrevPage={() => update({ page: Math.max(1, page - 1) })} onNextPage={() => update({ page: Math.min(pages, page + 1) })}
          extraContent={mode === "jackett" ? <div className="discovery-toolbar__preview"><label className="switch-row"><span>快速规则预览</span><input type="checkbox" checked={fastRules} onChange={(e) => update({ fastRules: e.target.checked, page: 1 })} /></label><label className="switch-row"><span>文件结构规则预览</span><input type="checkbox" checked={fileRules} onChange={(e) => update({ fileRules: e.target.checked, page: 1 })} /></label>{fileRules ? <span>{previewing ? "正在检查文件结构..." : `已检查 ${previewMeta?.inspectedCount ?? 0} / ${previewMeta?.inspectableCount ?? 0} 条可检查候选`}</span> : null}</div> : null} />
        <DiscoveryDrawer mode={mode} query={query} searching={mode === "stashbox" ? discovering : searching || previewing} error={(mode === "stashbox" ? discoverError : previewError ?? searchError) ?? null} pendingAddId={pendingAdd}
          discoverResult={discoveredResult} discoverItems={slice(discovered)} jackettItems={slice(activeJackett)} hasAnyResults={visible.length > 0} usedStashBoxName={discoveredResult?.usedStashBox?.name ?? null}
          onQueueDiscovered={(item) => void queueScene(item)} onAddJackett={(item) => void addJackett(item)} onTryJackett={() => update({ source: "jackett", sort: "relevance", page: 1 })} onClearTrackers={() => update({ trackers: [], page: 1 })} hasActiveTrackers={state.trackers.length > 0} />
      </article></div>
    </Drawer> : null}
  </>;
}
