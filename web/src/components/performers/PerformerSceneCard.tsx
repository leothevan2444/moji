import { faArrowUpRightFromSquare } from "@fortawesome/free-solid-svg-icons/faArrowUpRightFromSquare";
import { faCheck } from "@fortawesome/free-solid-svg-icons/faCheck";
import { faFilm } from "@fortawesome/free-solid-svg-icons/faFilm";
import { faPlus } from "@fortawesome/free-solid-svg-icons/faPlus";
import { faTags } from "@fortawesome/free-solid-svg-icons/faTags";
import { faUsers } from "@fortawesome/free-solid-svg-icons/faUsers";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useTranslation } from "react-i18next";

import {
  TaskStage,
  TaskStageStatus
} from "../../graphql/generated/graphql";
import i18n from "../../i18n/i18n";
import {
  taskPrimaryState
} from "../../utils";

import type { PerformersPageProps, StashPerformerSceneEntry } from "./types";
function performerSceneTaskLabel(task: NonNullable<StashPerformerSceneEntry["mojiTask"]>) {
  const state = taskPrimaryState(task);
  if (task.stage === TaskStage.Downloading && task.stageStatus !== TaskStageStatus.Blocked && task.progress > 0) return i18n.t("performerUi.downloadProgress", { progress: Math.min(100, Math.max(0, Math.round(task.progress * 100))) });
  return state.label;
}
function performerSceneTaskTone(task: NonNullable<StashPerformerSceneEntry["mojiTask"]>) { return taskPrimaryState(task).tone; }

function performerSceneSourceLabel(scene: StashPerformerSceneEntry) {
  if (scene.hasStashSource && scene.hasStashBoxSource) return i18n.t("performerUi.dualSource");
  if (scene.hasStashBoxSource) return "StashBox";
  return "Stash";
}

export function PerformerSceneCard({ scene, page }: { scene: StashPerformerSceneEntry; page: Pick<PerformersPageProps, "selectedSceneKeys" | "pendingSceneKeys" | "onToggleSceneSelection" | "onOpenTask" | "onQueueScene">; }) {
  const { t } = useTranslation();
  const { selectedSceneKeys, pendingSceneKeys, onToggleSceneSelection, onOpenTask, onQueueScene } = page;
  const selected = selectedSceneKeys.includes(scene.key);
  const pendingSingleQueue = pendingSceneKeys.includes(scene.key);
  const selectable = !scene.inLibrary && !scene.mojiTask && !pendingSingleQueue;
  return (
    <article
      key={scene.key}
      className={`candidate-card performer-scene-card has-media ${selected ? "is-selected" : ""} ${selectable ? "" : "is-unselectable"}`}
      onClick={() => {
        if (selectable) onToggleSceneSelection(scene.key);
      }}
    >
      <div className="performer-scene-card__media">
        {scene.imageUrl ? (
          <img
            src={scene.imageUrl}
            alt={scene.title || scene.code || "scene"}
            loading="lazy"
            onError={(event) => { event.currentTarget.style.display = "none"; }}
            className="performer-scene-card__image"
          />
        ) : (
          <div className="performer-scene-card__image-placeholder" aria-hidden="true">
            <FontAwesomeIcon icon={faFilm} />
          </div>
        )}
        <span
          className="performer-scene-card__source-badge"
          title={t("performerUi.source", { sources: scene.sourceLabels.join(" + ") })}
        >
          {performerSceneSourceLabel(scene)}
        </span>
      </div>
      {selectable ? (
        <button
          type="button"
          className="performer-scene-card__selector"
          aria-pressed={selected}
          aria-label={t(selected ? "performerUi.deselectName" : "performerUi.selectName", { name: scene.code || scene.title || scene.sourceSceneId })}
          title={t(selected ? "performerUi.deselect" : "performerUi.selectScene")}
          onClick={(event) => {
            event.stopPropagation();
            onToggleSceneSelection(scene.key);
          }}
        >
          {selected ? <FontAwesomeIcon icon={faCheck} /> : null}
        </button>
      ) : null}
      <div className="performer-scene-card__content">
        <div className="performer-scene-card__status-row">
          <h3 title={scene.code || scene.title || scene.sourceSceneId}>{scene.code || scene.title || scene.sourceSceneId}</h3>
          <time className="performer-scene-card__date" dateTime={scene.date ?? undefined}>
            {scene.date || t("performerUi.noDate")}
          </time>
        </div>
        <p className="performer-scene-card__meta" title={scene.studioName || t("performerUi.unknownStudio")}>
          {scene.studioName || t("performerUi.unknownStudio")}
        </p>
        <p className="performer-scene-card__title" title={scene.title || t("performerUi.noTitle")}>{scene.title || t("performerUi.noTitle")}</p>
        <div className="performer-scene-card__counts">
          <span
            title={scene.performers.length > 0 ? t("performerUi.cast", { names: scene.performers.map((item) => item.name).join(" / ") }) : t("performerUi.castCount", { count: scene.performerCount })}
            aria-label={t("performerUi.castCount", { count: scene.performerCount })}
          >
            <FontAwesomeIcon icon={faUsers} />
            {scene.performerCount}
          </span>
          <span
            title={scene.tags.length > 0 ? t("performerUi.tags", { names: scene.tags.map((item) => item.name).join(" / ") }) : t("performerUi.tagCount", { count: scene.tagCount })}
            aria-label={t("performerUi.tagCount", { count: scene.tagCount })}
          >
            <FontAwesomeIcon icon={faTags} />
            {scene.tagCount}
          </span>
        </div>
        <div className="performer-scene-card__bottom-actions">
          {scene.inLibrary ? (
            <span className="status-chip performer-scene-card__business-state tone-success" title={t("performerUi.sceneInLibrary")}>
              {t("performerUi.inLibrary")}
            </span>
          ) : scene.mojiTask ? (
            <button
              type="button"
              className={`status-chip performer-scene-card__business-state performer-scene-card__task ${performerSceneTaskTone(scene.mojiTask)}`}
              title={t("performerUi.taskTitle", { stage: performerSceneTaskLabel(scene.mojiTask), status: scene.mojiTask.stageStatusLabel })}
              onClick={(event) => {
                event.stopPropagation();
                onOpenTask(scene.mojiTask!.id);
              }}
            >
              {performerSceneTaskLabel(scene.mojiTask)}
            </button>
          ) : (
            <button
              type="button"
              className="status-chip performer-scene-card__business-state performer-scene-card__create-task tone-warn"
              title={t("performerUi.createScene")}
              disabled={pendingSingleQueue}
              onClick={(event) => {
                event.stopPropagation();
                onQueueScene(scene);
              }}
            >
              {pendingSingleQueue ? (
                t("performerUi.creating")
              ) : (
                <>
                  <FontAwesomeIcon icon={faPlus} />
                  {t("performerUi.createTask")}
                </>
              )}
            </button>
          )}
          {scene.url ? (
            <a
              className="profile-action-icon performer-scene-card__source-link"
              href={scene.url}
              target="_blank"
              rel="noreferrer"
              title={t("performerUi.original")}
              aria-label={t("performerUi.originalName", { name: scene.code || scene.title || scene.sourceSceneId })}
              onClick={(event) => event.stopPropagation()}
            >
              <FontAwesomeIcon icon={faArrowUpRightFromSquare} />
            </a>
          ) : null}
        </div>
      </div>
    </article>
  );

}
