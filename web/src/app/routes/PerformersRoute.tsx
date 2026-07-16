import { lazy, Suspense, useDeferredValue, useEffect, useState } from "react";
import { useLocation, useNavigate, useOutletContext, useParams, useSearchParams } from "react-router";
import { useQuery } from "urql";
import { Drawer } from "../../components/layout/Drawer";
import { PerformersPage } from "../../pages/PerformersPage";
import { usePerformersWorkspace } from "../../hooks/usePerformersWorkspace";
import { parsePerformerSearchParams, serializePerformerSearchParams } from "../searchParams";
import { LibraryFilter, PerformerBatchFieldsFragmentDoc, PerformersConfigDocumentDocument, SceneSourceFilter } from "../../graphql/generated/graphql";
import { useFragment as readFragment } from "../../graphql/generated/fragment-masking";
import { describeQueryError } from "../../services/queryError";
import type { AppOutletContext } from "../AppLayout";
import { useTranslation } from "react-i18next";
import { mergePerformerSelection } from "../../utils";
import type { PerformerBatchConfirmAction } from "../../components/drawers/PerformerBatchConfirmDrawer";
import type { PerformerBatchResultView } from "../../components/drawers/PerformerBatchResultDrawer";
import { buildPerformerSceneQueueInput, usePerformerSceneSelection } from "./performerSceneSelection";
import { describePerformerSceneQueueResult } from "../../services/performerSceneQueueResult";

const PerformerBatchConfirmDrawer = lazy(() => import("../../components/drawers/PerformerBatchConfirmDrawer").then((module) => ({ default: module.PerformerBatchConfirmDrawer })));
const PerformerBatchResultDrawer = lazy(() => import("../../components/drawers/PerformerBatchResultDrawer").then((module) => ({ default: module.PerformerBatchResultDrawer })));

export function Component() {
  const { t } = useTranslation();
  const { performerId } = useParams();
  const [params] = useSearchParams();
  const state = parsePerformerSearchParams(params);
  const deferredSearch = useDeferredValue(state.q);
  const deferredSceneSearch = useDeferredValue(state.sceneQ);
  const navigate = useNavigate();
  const location = useLocation();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const [pendingSubscription, setPendingSubscription] = useState<string | null>(null);
  const [pendingKeys, setPendingKeys] = useState<string[]>([]);
  const [performerSelectionMode, setPerformerSelectionMode] = useState(false);
  const [selectedPerformerIds, setSelectedPerformerIds] = useState<string[]>([]);
  const [batchConfirm, setBatchConfirm] = useState<{ action: PerformerBatchConfirmAction; ids: string[] } | null>(null);
  const [batchResult, setBatchResult] = useState<PerformerBatchResultView | null>(null);
  const [lastRefreshedAt, setLastRefreshedAt] = useState<string | null>(null);
  const [reloadingList, setReloadingList] = useState(false);
  useEffect(() => { setPerformerSelectionMode(false); setSelectedPerformerIds([]); }, [performerId]);
  const source = state.source === "stash" ? SceneSourceFilter.Stash : state.source === "stashbox" ? SceneSourceFilter.Stashbox : SceneSourceFilter.All;
  const library = state.library === "in-library" ? LibraryFilter.InLibrary : state.library === "not-in-library" ? LibraryFilter.NotInLibrary : LibraryFilter.All;
  const [{ data: config }] = useQuery({ query: PerformersConfigDocumentDocument, requestPolicy: "cache-and-network" });
  const subscription = usePerformersWorkspace({ enabled: true, search: deferredSearch || null, page: state.page, pageSize: state.pageSize, performerId: performerId ?? null, performerSceneSearch: deferredSceneSearch || null, performerSceneSource: source, performerSceneLibrary: library, performerScenePage: state.scenePage, performerScenePageSize: state.scenePageSize });
  const sceneSelection = usePerformerSceneSelection(performerId ?? null, subscription.performerScenes);
  const update = (patch: Partial<typeof state>) => {
    const next = serializePerformerSearchParams({ ...state, ...patch });
    const base = performerId ? `/performers/${encodeURIComponent(performerId)}` : "/performers";
    navigate(`${base}${next.size ? `?${next}` : ""}`, { replace: true });
  };
  const reload = async () => {
    setReloadingList(true);
    try {
      const results = await subscription.reloadSubscription();
      const failure = results.find((result) => result.error)?.error;
      if (failure) pushToast("tone-danger", describeQueryError(failure));
      else { setLastRefreshedAt(new Date().toISOString()); pushToast("tone-success", t("performerRoute.reloaded")); }
    } finally { setReloadingList(false); }
  };
  const toggle = async (performer: { id: string; name: string; subscribed: boolean }) => {
    setPendingSubscription(performer.id);
    try { const result = performer.subscribed ? await subscription.unsubscribePerformer({ stashPerformerID: performer.id }) : await subscription.subscribePerformer({ stashPerformerID: performer.id }); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); else { pushToast("tone-success", t(performer.subscribed ? "performerRoute.unsubscribed" : "performerRoute.subscribed", { name: performer.name })); } }
    finally { setPendingSubscription(null); }
  };
  const refreshOne = async (performer: { id: string; name: string }) => { setPendingSubscription(performer.id); try { const result = await subscription.refreshSubscribedPerformer({ stashPerformerID: performer.id }); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); else { pushToast("tone-info", t("performerRoute.checked", { name: performer.name })); } } finally { setPendingSubscription(null); } };
  const refreshAll = async () => { const result = await subscription.refreshSubscriptionsNow({}); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); else { pushToast("tone-info", t("performerRoute.checkedAll")); } };
  const showPerformerBatchResult = (payload: PerformerBatchResultView | undefined) => {
    if (!payload) { pushToast("tone-danger", t("performerBatch.noResult")); return; }
    if (payload.summary.failedCount || payload.summary.skippedCount) setBatchResult(payload);
    else pushToast("tone-success", t("performerBatch.complete", { count: payload.summary.succeededCount }));
  };
  const batchPayload = (value: any): PerformerBatchResultView | undefined => value ? readFragment(PerformerBatchFieldsFragmentDoc, value) : undefined;
  const runBatchSubscribe = async (ids: string[]) => {
    const result = await subscription.subscribePerformers({ ids });
    if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
    showPerformerBatchResult(batchPayload(result.data?.subscribePerformers));
    await subscription.reloadStashPerformers();
  };
  const runBatchUnsubscribe = async (ids: string[]) => {
    const result = await subscription.unsubscribePerformers({ ids });
    if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
    const payload = batchPayload(result.data?.unsubscribePerformers);
    if (payload) {
      const removed = new Set(payload.results.filter((item) => item.status === "SUCCEEDED").map((item) => item.performerId));
      setSelectedPerformerIds((current) => current.filter((id) => !removed.has(id)));
    }
    showPerformerBatchResult(payload);
    setBatchConfirm(null);
    await subscription.reloadStashPerformers();
  };
  const runBatchRefresh = async (ids: string[]) => {
    const result = await subscription.refreshSubscribedPerformers({ ids });
    if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
    showPerformerBatchResult(batchPayload(result.data?.refreshSubscribedPerformers));
    setBatchConfirm(null);
    await subscription.reloadStashPerformers();
  };
  const togglePerformerSelection = (id: string) => setSelectedPerformerIds((current) => current.includes(id) ? current.filter((item) => item !== id) : current.length >= 100 ? (pushToast("tone-info", t("performerBatch.limit", { count: 100 })), current) : [...current, id]);
  const selectVisiblePerformers = (ids: string[]) => {
    const merged = [...new Set([...selectedPerformerIds, ...ids])];
    if (merged.length > 100) pushToast("tone-info", t("performerBatch.limit", { count: 100 }));
    setSelectedPerformerIds((current) => mergePerformerSelection(current, ids));
  };
  const queueSelected = async () => {
    if (!performerId || !sceneSelection.selected.size) return;
    const result = await subscription.queuePerformerScenes({ input: buildPerformerSceneQueueInput(performerId, sceneSelection.selected) });
    if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
    const payload = result.data?.queuePerformerScenes; if (!payload) { pushToast("tone-danger", t("performerRoute.batchNoResult")); return; }
    const { summary, results } = payload;
    pushToast(summary.queuedCount ? (summary.failedCount ? "tone-info" : "tone-success") : summary.failedCount ? "tone-danger" : "tone-info", t("performerRoute.batchSummary", { queued: summary.queuedCount, skipped: summary.skippedCount, failed: summary.failedCount }));
    sceneSelection.applyResults(results);
  };
  const queueOne = async (scene: (typeof subscription.performerScenes)[number]) => {
    if (!performerId || scene.inLibrary || scene.mojiTask || pendingKeys.includes(scene.key)) return;
    setPendingKeys((current) => [...current, scene.key]);
    try { const result = await subscription.queueSinglePerformerScene({ input: { performerId, sceneKeys: [scene.key] } }); if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; } const item = result.data?.queuePerformerScenes.results[0]; if (!item) pushToast("tone-danger", t("performerRoute.createNoResult")); else { pushToast(item.status === "QUEUED" ? "tone-success" : item.status === "SKIPPED" ? "tone-info" : "tone-danger", item.status === "QUEUED" ? t("performerRoute.created", { scene: item.resolvedCode || scene.code || scene.title || t("performerRoute.scene") }) : describePerformerSceneQueueResult(item.reasonCode)); if (item.status === "QUEUED") sceneSelection.applyResults([item]); } }
    finally { setPendingKeys((current) => current.filter((key) => key !== scene.key)); }
  };

  return <><PerformersPage stashBaseURL={config?.settings.stash.url ?? null} stashPerformerPage={subscription.stashPerformerPage} stashPerformers={subscription.stashPerformers} subscribedPerformers={subscription.subscribedPerformers}
    fetchingStashPerformers={subscription.fetchingStashPerformers} fetchingSubscription={subscription.fetchingSubscription} performerDetail={subscription.performerDetail} performerScenePage={subscription.performerScenePage} performerScenes={subscription.performerScenes}
    fetchingPerformerDetail={subscription.fetchingPerformerDetail} fetchingPerformerScenes={subscription.fetchingPerformerScenes} refreshingSubscriptionNow={subscription.refreshingSubscriptionNow} refreshingList={reloadingList || subscription.fetchingStashPerformers} performerBatchPending={subscription.subscribingPerformers || subscription.unsubscribingPerformers || subscription.refreshingSubscribedPerformers} queueingPerformerScenes={subscription.queueingPerformerScenes}
    subscriptionSearch={state.q} subscriptionPageSize={state.pageSize} selectedPerformerId={performerId ?? null} performerSceneSearch={state.sceneQ} performerSceneSourceFilter={source} performerSceneLibraryFilter={library} performerScenePageSize={state.scenePageSize}
    selectedSceneKeys={sceneSelection.selectedKeys} selectedPerformerIds={selectedPerformerIds} performerSelectionMode={performerSelectionMode} lastRefreshedAt={lastRefreshedAt} pendingSceneKeys={pendingKeys} pendingSubscriptionID={pendingSubscription} subscriptionError={subscription.subscriptionError ?? null} stashPerformersError={subscription.stashPerformersError ?? null} performerDetailError={subscription.performerDetailError ?? null} performerScenesError={subscription.performerScenesError ?? null}
    onSearchChange={(q) => update({ q, page: 1 })} onPageSizeChange={(pageSize) => update({ pageSize, page: 1 })} onReload={() => void reload()} onRefreshAll={() => void refreshAll()} onTogglePerformerSelectionMode={() => { if (performerSelectionMode) setSelectedPerformerIds([]); setPerformerSelectionMode((current) => !current); }} onTogglePerformerSelection={togglePerformerSelection} onSelectVisiblePerformers={selectVisiblePerformers} onClearPerformerSelection={() => setSelectedPerformerIds([])} onBatchSubscribePerformers={(ids) => void runBatchSubscribe(ids)} onBatchUnsubscribePerformers={(ids) => setBatchConfirm({ action: "unsubscribe", ids })} onBatchRefreshPerformers={(ids) => setBatchConfirm({ action: "refresh", ids })} onToggle={(p) => void toggle(p)} onRefreshOne={(p) => void refreshOne(p)}
    onPrevPage={() => update({ page: Math.max(1, state.page - 1) })} onNextPage={() => update({ page: state.page + 1 })}
    onOpenPerformer={(id) => { const next = serializePerformerSearchParams(state); navigate(`/performers/${encodeURIComponent(id)}${next.size ? `?${next}` : ""}`, { state: { backgroundLocation: location } }); }}
    onOpenTask={(id) => navigate(`/tasks/${encodeURIComponent(id)}`, { state: { backgroundLocation: location } })} onBackToList={() => { const next = serializePerformerSearchParams({ ...state, sceneQ: "", source: "all", library: "all", scenePage: 1, scenePageSize: 24 }); navigate(`/performers${next.size ? `?${next}` : ""}`); }}
    onPerformerSceneSearchChange={(sceneQ) => update({ sceneQ, scenePage: 1 })} onPerformerSceneSourceChange={(value) => update({ source: value === SceneSourceFilter.Stash ? "stash" : value === SceneSourceFilter.Stashbox ? "stashbox" : "all", scenePage: 1 })}
    onPerformerSceneLibraryChange={(value) => update({ library: value === LibraryFilter.InLibrary ? "in-library" : value === LibraryFilter.NotInLibrary ? "not-in-library" : "all", scenePage: 1 })} onPerformerScenePageSizeChange={(scenePageSize) => update({ scenePageSize, scenePage: 1 })}
    onPrevPerformerScenePage={() => update({ scenePage: Math.max(1, state.scenePage - 1) })} onNextPerformerScenePage={() => update({ scenePage: state.scenePage + 1 })}
    onToggleSceneSelection={sceneSelection.toggle} onSelectCurrentScenePage={sceneSelection.addVisible} onClearSceneSelection={sceneSelection.clear}
    onQueueSelectedScenes={() => void queueSelected()} onQueueScene={(scene) => void queueOne(scene)} />
    {batchConfirm ? <Drawer visibleDrawer="performer-batch-confirm" closing={false} title={t("performerBatch.confirmTitle")} onClose={() => { if (!(subscription.unsubscribingPerformers || subscription.refreshingSubscribedPerformers)) setBatchConfirm(null); }}><Suspense fallback={null}><PerformerBatchConfirmDrawer action={batchConfirm.action} count={batchConfirm.ids.length} pending={subscription.unsubscribingPerformers || subscription.refreshingSubscribedPerformers} onConfirm={() => batchConfirm.action === "unsubscribe" ? void runBatchUnsubscribe(batchConfirm.ids) : void runBatchRefresh(batchConfirm.ids)} onCancel={() => setBatchConfirm(null)} /></Suspense></Drawer> : null}
    {batchResult ? <Drawer visibleDrawer="performer-batch-result" closing={false} title={t("performerBatch.resultTitle")} onClose={() => setBatchResult(null)}><Suspense fallback={null}><PerformerBatchResultDrawer payload={batchResult} /></Suspense></Drawer> : null}
  </>;
}
