import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faFilm } from "@fortawesome/free-solid-svg-icons/faFilm";
import { faArrowUpRightFromSquare } from "@fortawesome/free-solid-svg-icons/faArrowUpRightFromSquare";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "urql";
import { TaskStashScenesDocumentDocument } from "../../graphql/generated/graphql";
import { TaskValue } from "./TaskValue";
import type { DashboardTask } from "../../utils";

function ScenePoster({ src }: { src?: string | null }) {
  const [failed, setFailed] = useState(false);
  return <div className="task-stash-scene__poster" aria-hidden="true">
    {src && !failed ? <img src={src} alt="" loading="lazy" onError={() => setFailed(true)} /> : <FontAwesomeIcon icon={faFilm} />}
  </div>;
}

export function TaskStashScenes({ task, onCopy }: { task: DashboardTask; onCopy?: (value: string, message: string) => void | Promise<void> }) {
  const { t } = useTranslation();
  const ids = [...new Set(task.stashSceneIds ?? [])];
  const [result, reload] = useQuery({
    query: TaskStashScenesDocumentDocument, variables: { id: task.id },
    pause: ids.length === 0, requestPolicy: "network-only"
  });
  const scenes = result.data?.taskStashScenes ?? [];
  const missing = ids.length - scenes.length;
  return <article className="drawer-card task-stash-scenes" aria-label={t("taskStash.title")}>
    <div className="drawer-card__head"><h3>{t("taskStash.title")}</h3></div>
    {ids.length === 0 ? <p className="task-stash-scenes__message">{t(task.stage === "COMPLETED" ? "taskStash.empty" : "taskStash.pending")}</p> : <>
      {result.fetching && !scenes.length ? <p className="task-stash-scenes__message" role="status">{t("taskStash.loading")}</p> : null}
      {result.error ? <div className="task-stash-scenes__feedback" role="status">
        <span>{t("taskStash.failed")}</span>
        <button type="button" className="ghost-button" disabled={result.fetching} onClick={() => reload({ requestPolicy: "network-only" })}>{t("taskStash.retry")}</button>
      </div> : null}
      {scenes.length ? <ul className="task-stash-scenes__list">
        {scenes.map(scene => {
          const title = scene.title || scene.code || t("taskStash.untitled");
          return <li key={scene.id} className="task-stash-scene">
            <ScenePoster key={scene.imageUrl} src={scene.imageUrl} />
            <div className="task-stash-scene__content">
              {scene.code && scene.code !== title ? <span className="task-stash-scene__code">{scene.code}</span> : null}
              <h4 title={title}>{title}</h4>
              {scene.date || scene.studio ? <div className="task-stash-scene__meta">
                {scene.date ? <time dateTime={scene.date} aria-label={t("taskStash.releaseDate")}>{scene.date}</time> : null}
                {scene.studio ? <span title={scene.studio}>{scene.studio}</span> : null}
              </div> : null}
              {scene.performers.length ? <p className="task-stash-scene__performers" title={scene.performers.join(" · ")}>{scene.performers.join(" · ")}</p> : null}
              {scene.url ? <a className="task-stash-scene__link" href={scene.url} target="_blank" rel="noopener noreferrer" aria-label={t("taskStash.openLabel", { title })}>
                {t("taskStash.open")} <FontAwesomeIcon icon={faArrowUpRightFromSquare} />
              </a> : null}
            </div>
          </li>;
        })}
      </ul> : null}
      {!result.fetching && !result.error && missing > 0 ? <p className="task-stash-scenes__message">{t(scenes.length ? "taskStash.missing" : "taskStash.empty", { count: missing })}</p> : null}
    </>}
    {task.stashScanPath?.trim() ? <dl className="task-stash-scenes__location">
      <TaskValue label={t("taskInfo.libraryLocation")} value={task.stashScanPath.trim()} onCopy={onCopy} />
    </dl> : null}
  </article>;
}
