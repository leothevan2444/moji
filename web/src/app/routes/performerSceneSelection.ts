import type { QueuePerformerSceneInput, QueuePerformerScenesInput } from "../../graphql/generated/graphql";
import { useEffect, useState } from "react";

export type PerformerSceneSelectionMap = ReadonlyMap<string, QueuePerformerSceneInput>;

export type SceneSnapshotSource = {
  key: string;
  sourceSceneId: string;
  stashBoxSceneId?: string | null;
  stashBoxEndpoint?: string | null;
  code?: string | null;
  title?: string | null;
  inLibrary: boolean;
};

const emptySelection: PerformerSceneSelectionMap = new Map();

export function performerSceneQueueSnapshot(scene: SceneSnapshotSource): QueuePerformerSceneInput {
  return {
    key: scene.key,
    sourceSceneId: scene.sourceSceneId,
    stashBoxSceneId: scene.stashBoxSceneId ?? undefined,
    stashBoxEndpoint: scene.stashBoxEndpoint ?? undefined,
    code: scene.code ?? undefined,
    title: scene.title ?? undefined,
    inLibrary: scene.inLibrary
  };
}

export function togglePerformerSceneSelection(current: PerformerSceneSelectionMap, scene: SceneSnapshotSource) {
  const next = new Map(current);
  if (next.has(scene.key)) next.delete(scene.key);
  else next.set(scene.key, performerSceneQueueSnapshot(scene));
  return next;
}

export function addPerformerSceneSelections(current: PerformerSceneSelectionMap, scenes: SceneSnapshotSource[]) {
  const next = new Map(current);
  for (const scene of scenes) next.set(scene.key, performerSceneQueueSnapshot(scene));
  return next;
}

export function buildPerformerSceneQueueInput(performerId: string, selected: PerformerSceneSelectionMap): QueuePerformerScenesInput {
  return { performerId, scenes: [...selected.values()] };
}

export function retainUnqueuedPerformerScenes(current: PerformerSceneSelectionMap, results: ReadonlyArray<{ key: string; status: string }>) {
  const queued = new Set(results.filter((item) => item.status === "QUEUED").map((item) => item.key));
  return new Map([...current].filter(([key]) => !queued.has(key)));
}

export function usePerformerSceneSelection(performerId: string | null, visibleScenes: SceneSnapshotSource[]) {
  const [state, setState] = useState<{ performerId: string | null; selected: PerformerSceneSelectionMap }>(() => ({ performerId, selected: new Map() }));
  const selected = state.performerId === performerId ? state.selected : emptySelection;

  useEffect(() => setState({ performerId, selected: new Map() }), [performerId]);

  const update = (transform: (current: PerformerSceneSelectionMap) => PerformerSceneSelectionMap) => {
    setState((current) => ({
      performerId,
      selected: transform(current.performerId === performerId ? current.selected : emptySelection)
    }));
  };

  return {
    selected,
    selectedKeys: [...selected.keys()],
    toggle(key: string) {
      const scene = visibleScenes.find((item) => item.key === key);
      if (scene) update((current) => togglePerformerSceneSelection(current, scene));
    },
    addVisible(keys: string[]) {
      const keySet = new Set(keys);
      update((current) => addPerformerSceneSelections(current, visibleScenes.filter((scene) => keySet.has(scene.key))));
    },
    clear() {
      update(() => new Map());
    },
    applyResults(results: ReadonlyArray<{ key: string; status: string }>) {
      update((current) => retainUnqueuedPerformerScenes(current, results));
    }
  };
}
