import { useDeferredValue, useEffect, useState } from "react";
import { useLocation, useNavigate, useOutletContext, useParams, useSearchParams } from "react-router";
import { useQuery } from "urql";
import { PerformersPage } from "../../pages/PerformersPage";
import { usePerformersWorkspace } from "../../hooks/usePerformersWorkspace";
import { parsePerformerSearchParams, serializePerformerSearchParams } from "../searchParams";
import { LibraryFilter, PerformersConfigDocumentDocument, SceneSourceFilter } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import type { AppOutletContext } from "../AppLayout";
import { useTranslation } from "react-i18next";

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
  const [selectedKeys, setSelectedKeys] = useState<string[]>([]);
  const [pendingKeys, setPendingKeys] = useState<string[]>([]);
  useEffect(() => setSelectedKeys([]), [performerId]);
  const source = state.source === "stash" ? SceneSourceFilter.Stash : state.source === "stashbox" ? SceneSourceFilter.Stashbox : SceneSourceFilter.All;
  const library = state.library === "in-library" ? LibraryFilter.InLibrary : state.library === "not-in-library" ? LibraryFilter.NotInLibrary : LibraryFilter.All;
  const [{ data: config }] = useQuery({ query: PerformersConfigDocumentDocument, requestPolicy: "cache-and-network" });
  const subscription = usePerformersWorkspace({ enabled: true, search: deferredSearch || null, page: state.page, pageSize: state.pageSize, performerId: performerId ?? null, performerSceneSearch: deferredSceneSearch || null, performerSceneSource: source, performerSceneLibrary: library, performerScenePage: state.scenePage, performerScenePageSize: state.scenePageSize });
  const update = (patch: Partial<typeof state>) => {
    const next = serializePerformerSearchParams({ ...state, ...patch });
    const base = performerId ? `/performers/${encodeURIComponent(performerId)}` : "/performers";
    navigate(`${base}${next.size ? `?${next}` : ""}`, { replace: true });
  };
  const reload = subscription.reloadSubscription;
  const toggle = async (performer: { id: string; name: string; subscribed: boolean }) => {
    setPendingSubscription(performer.id);
    try { const result = performer.subscribed ? await subscription.unsubscribePerformer({ stashPerformerID: performer.id }) : await subscription.subscribePerformer({ stashPerformerID: performer.id }); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); else { pushToast("tone-success", t(performer.subscribed ? "performerRoute.unsubscribed" : "performerRoute.subscribed", { name: performer.name })); } }
    finally { setPendingSubscription(null); }
  };
  const refreshOne = async (performer: { id: string; name: string }) => { setPendingSubscription(performer.id); try { const result = await subscription.refreshSubscribedPerformer({ stashPerformerID: performer.id }); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); else { pushToast("tone-info", t("performerRoute.checked", { name: performer.name })); } } finally { setPendingSubscription(null); } };
  const refreshAll = async () => { const result = await subscription.refreshSubscriptionsNow({}); if (result.error) pushToast("tone-danger", describeQueryError(result.error)); else { pushToast("tone-info", t("performerRoute.checkedAll")); } };
  const sceneInput = (scene: (typeof subscription.performerScenes)[number]) => ({ key: scene.key, sourceSceneId: scene.sourceSceneId, stashBoxSceneId: scene.stashBoxSceneId ?? undefined, stashBoxEndpoint: scene.stashBoxEndpoint ?? undefined, code: scene.code ?? undefined, title: scene.title ?? undefined, inLibrary: scene.inLibrary });
  const queueSelected = async () => {
    if (!performerId || !selectedKeys.length) return;
    const scenes = subscription.performerScenes.filter((scene) => selectedKeys.includes(scene.key));
    if (!scenes.length) { pushToast("tone-danger", t("performerRoute.noSelection")); return; }
    const result = await subscription.queuePerformerScenes({ input: { performerId, scenes: scenes.map(sceneInput) } });
    if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; }
    const payload = result.data?.queuePerformerScenes; if (!payload) { pushToast("tone-danger", t("performerRoute.batchNoResult")); return; }
    const { summary, results } = payload;
    pushToast(summary.queuedCount ? (summary.failedCount ? "tone-info" : "tone-success") : summary.failedCount ? "tone-danger" : "tone-info", t("performerRoute.batchSummary", { queued: summary.queuedCount, skipped: summary.skippedCount, failed: summary.failedCount }));
    const queued = new Set(results.filter((item) => item.status === "QUEUED").map((item) => item.key)); setSelectedKeys((current) => current.filter((key) => !queued.has(key)));
  };
  const queueOne = async (scene: (typeof subscription.performerScenes)[number]) => {
    if (!performerId || scene.inLibrary || scene.mojiTask || pendingKeys.includes(scene.key)) return;
    setPendingKeys((current) => [...current, scene.key]);
    try { const result = await subscription.queueSinglePerformerScene({ input: { performerId, scenes: [sceneInput(scene)] } }); if (result.error) { pushToast("tone-danger", describeQueryError(result.error)); return; } const item = result.data?.queuePerformerScenes.results[0]; if (!item) pushToast("tone-danger", t("performerRoute.createNoResult")); else { pushToast(item.status === "QUEUED" ? "tone-success" : item.status === "SKIPPED" ? "tone-info" : "tone-danger", item.status === "QUEUED" ? t("performerRoute.created", { scene: item.resolvedCode || scene.code || scene.title || t("performerRoute.scene") }) : item.message); if (item.status !== "FAILED") setSelectedKeys((current) => current.filter((key) => key !== scene.key)); } }
    finally { setPendingKeys((current) => current.filter((key) => key !== scene.key)); }
  };

  return <PerformersPage stashBaseURL={config?.settings.stash.url ?? null} stashPerformerPage={subscription.stashPerformerPage} stashPerformers={subscription.stashPerformers} subscribedPerformers={subscription.subscribedPerformers}
    fetchingStashPerformers={subscription.fetchingStashPerformers} fetchingSubscription={subscription.fetchingSubscription} performerDetail={subscription.performerDetail} performerScenePage={subscription.performerScenePage} performerScenes={subscription.performerScenes}
    fetchingPerformerDetail={subscription.fetchingPerformerDetail} fetchingPerformerScenes={subscription.fetchingPerformerScenes} refreshingSubscriptionNow={subscription.refreshingSubscriptionNow} queueingPerformerScenes={subscription.queueingPerformerScenes}
    subscriptionSearch={state.q} subscriptionPageSize={state.pageSize} selectedPerformerId={performerId ?? null} performerSceneSearch={state.sceneQ} performerSceneSourceFilter={source} performerSceneLibraryFilter={library} performerScenePageSize={state.scenePageSize}
    selectedSceneKeys={selectedKeys} pendingSceneKeys={pendingKeys} pendingSubscriptionID={pendingSubscription} subscriptionError={subscription.subscriptionError ?? null} stashPerformersError={subscription.stashPerformersError ?? null} performerDetailError={subscription.performerDetailError ?? null} performerScenesError={subscription.performerScenesError ?? null}
    onSearchChange={(q) => update({ q, page: 1 })} onPageSizeChange={(pageSize) => update({ pageSize, page: 1 })} onReload={() => void reload()} onRefreshAll={() => void refreshAll()} onToggle={(p) => void toggle(p)} onRefreshOne={(p) => void refreshOne(p)}
    onPrevPage={() => update({ page: Math.max(1, state.page - 1) })} onNextPage={() => update({ page: state.page + 1 })}
    onOpenPerformer={(id) => { const next = serializePerformerSearchParams(state); navigate(`/performers/${encodeURIComponent(id)}${next.size ? `?${next}` : ""}`, { state: { backgroundLocation: location } }); }}
    onOpenTask={(id) => navigate(`/tasks/${encodeURIComponent(id)}`, { state: { backgroundLocation: location } })} onBackToList={() => { const next = serializePerformerSearchParams({ ...state, sceneQ: "", source: "all", library: "all", scenePage: 1, scenePageSize: 24 }); navigate(`/performers${next.size ? `?${next}` : ""}`); }}
    onPerformerSceneSearchChange={(sceneQ) => update({ sceneQ, scenePage: 1 })} onPerformerSceneSourceChange={(value) => update({ source: value === SceneSourceFilter.Stash ? "stash" : value === SceneSourceFilter.Stashbox ? "stashbox" : "all", scenePage: 1 })}
    onPerformerSceneLibraryChange={(value) => update({ library: value === LibraryFilter.InLibrary ? "in-library" : value === LibraryFilter.NotInLibrary ? "not-in-library" : "all", scenePage: 1 })} onPerformerScenePageSizeChange={(scenePageSize) => update({ scenePageSize, scenePage: 1 })}
    onPrevPerformerScenePage={() => update({ scenePage: Math.max(1, state.scenePage - 1) })} onNextPerformerScenePage={() => update({ scenePage: state.scenePage + 1 })}
    onToggleSceneSelection={(key) => setSelectedKeys((current) => current.includes(key) ? current.filter((x) => x !== key) : [...current, key])} onSelectCurrentScenePage={(keys) => setSelectedKeys((current) => [...new Set([...current, ...keys])])} onClearSceneSelection={() => setSelectedKeys([])}
    onQueueSelectedScenes={() => void queueSelected()} onQueueScene={(scene) => void queueOne(scene)} />;
}
