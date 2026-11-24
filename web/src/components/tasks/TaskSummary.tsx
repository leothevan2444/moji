import { useEffect, useRef, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "urql";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowUpRightFromSquare } from "@fortawesome/free-solid-svg-icons/faArrowUpRightFromSquare";
import { TaskStashScenesDocumentDocument, type TaskStashScenesDocumentQuery } from "../../graphql/generated/graphql";
import { useAppTimezone } from "../../hooks/useAppTimezone";
import { SceneArtwork } from "../common/SceneArtwork";
import { TaskTimeline } from "./TaskTimeline";
import { TaskSelectionReport } from "./TaskSelectionReport";
import { TaskValue } from "./TaskValue";
import { formatDateTime, taskPresentation, taskSourceLabel, type DashboardTask } from "../../utils";

type StashScene = TaskStashScenesDocumentQuery["taskStashScenes"][number];
type Identity = Pick<StashScene, "id" | "title" | "date" | "studio" | "performers" | "imageUrl" | "url"> & {fromStash:boolean; missingFields?:string[]};

export function TaskSummary({task,onCopy,action}:{task:DashboardTask;action?:ReactNode;onCopy:(value:string,message:string)=>void|Promise<void>}) {
  useAppTimezone();
  const {t} = useTranslation();
  const film = task.film;
  const ids = [...new Set(task.stashSceneIds ?? [])];
  const knownIds = new Set(ids);
  const linked = ids.length > 0;
  const [result,reload] = useQuery({query:TaskStashScenesDocumentDocument,variables:{id:task.id},pause:!linked,requestPolicy:"network-only"});
  const [confirmed,setConfirmed] = useState<StashScene[]>([]);
  const refreshKey = [task.stage,task.stageStatus,task.stashIngestPhase,...ids].join(":");
  const previous = useRef({refreshKey,linked});
  const refreshing = linked && (result.fetching || previous.current.refreshKey !== refreshKey);
  useEffect(() => {
    if (previous.current.refreshKey !== refreshKey && previous.current.linked && linked) reload({requestPolicy:"network-only"});
    previous.current = {refreshKey,linked};
  },[refreshKey,linked,reload]);
  useEffect(() => {
    if (result.data && !result.error && !result.fetching) setConfirmed(result.data.taskStashScenes);
  },[result.data,result.error,result.fetching]);
  // A failed read keeps the last confirmed Stash view. A successful empty read
  // clears it, so removed scenes never masquerade as a source-metadata fallback.
  const scenes = (result.error ? confirmed : result.data?.taskStashScenes ?? confirmed).filter(scene=>knownIds.has(scene.id));
  const missing = Math.max(0,ids.length-scenes.length);
  let identities: Identity[];
  if (!linked) {
    identities = [{id:"source",title:film?.title??"",date:film?.date??"",studio:film?.studio??"",performers:film?.performers??[],imageUrl:film?.imageUrl??null,url:null,fromStash:false}];
  } else if (scenes.length) {
    identities = scenes.map(scene=>({...scene,fromStash:true,missingFields:[
      !scene.title?.trim() && "title",!scene.code?.trim() && "code",!scene.date && "date",!scene.studio && "studio",!scene.performers.length && "performers"
    ].filter((field):field is string=>Boolean(field))}));
  } else {
    identities = [{id:"unavailable",title:t("taskDetail.overview"),date:"",studio:"",performers:[],imageUrl:null,url:null,fromStash:true}];
  }
  const performers = [...new Set(identities.flatMap(identity=>identity.performers).filter(name=>name.trim()))];
  const presentation = taskPresentation(task);

  return (
    <article className="drawer-card task-detail-overview" aria-label={t("taskDetail.overview")}>
      <div className="task-detail-identities">
        {identities.map((identity) => {
          const title = identity.title?.trim() || t(identity.fromStash ? "taskStash.untitled" : "taskDetail.overview");
          return <div key={identity.id} className="task-detail-summary">
            {identity.fromStash || identity.imageUrl || identity.title ? <div className="task-detail-summary__artwork"><SceneArtwork src={identity.imageUrl} alt="" /></div> : null}
            <div className="task-detail-heading">
              <div className="task-detail-summary__title">
                <h3>{title}</h3>
                {identity.url ? <a className="task-detail-summary__link" href={identity.url} target="_blank" rel="noopener noreferrer" aria-label={t("taskStash.openLabel", { title })} title={t("taskStash.open")}><FontAwesomeIcon icon={faArrowUpRightFromSquare} aria-hidden="true" /></a> : null}
              </div>
              {identity.date || identity.studio ? <div className="task-detail-summary__metadata">
                {identity.date ? <time dateTime={identity.date}>{identity.date}</time> : null}
                {identity.studio ? <span>{identity.studio}</span> : null}
              </div> : null}
              {identity.performers.length ? <p className="task-detail-summary__performers" title={identity.performers.join(" / ")}>{identity.performers.join(" / ")}</p> : null}

              {identity.missingFields?.length ? <p className="task-detail-summary__notice">{t("taskModel.ingest.missingFields", {fields: identity.missingFields.map(field => t(`taskModel.ingest.fields.${field}`)).join(t("taskModel.ingest.fieldSeparator"))})}</p> : null}
            </div>
          </div>;
        })}
      </div>
      {linked && refreshing && !scenes.length ? <p className="task-detail-summary__notice" role="status">{t("taskStash.loading")}</p> : null}
      {linked && result.error ? <div className="task-detail-summary__feedback" role="status"><span>{t("taskStash.failed")}</span><button type="button" className="ghost-button" disabled={result.fetching} onClick={() => reload({requestPolicy:"network-only"})}>{t("taskStash.retry")}</button></div> : null}
      {linked && !refreshing && !result.error && missing > 0 ? <p className="task-detail-summary__notice">{t(scenes.length ? "taskStash.missing" : "taskStash.notFound", {count:missing})}</p> : null}
      {!linked && task.stage === "COMPLETED" ? <p className="task-detail-summary__notice">{t("taskStash.empty")}</p> : null}
      <TaskTimeline key={task.id} steps={presentation.lifecycle} sourcingDetails={task.selectionReport ? <TaskSelectionReport report={task.selectionReport} taskCode={task.code} /> : undefined} status={presentation} action={action} schedule={{last:task.sourcingLastCheckedAt,next:task.sourcingNextCheckAt}} animateIngest={task.stageStatus === "RUNNING"} transfer={{ action: task.transferAction, target: task.mojiTransferPath }} onCopy={onCopy}
        downloadProgress={presentation.showDownloadProgress ? { percent: presentation.progressPercent, label: presentation.progressLabel, active: presentation.animateDownloadProgress } : undefined} />
      <details key={`metadata:${task.id}`} className="task-detail-extra">
        <summary>{t("taskDetail.moreInformation")}</summary>
        <dl className="settings-grid task-detail-metadata">
          <div><dt>{t("taskDetail.source")}</dt><dd>{taskSourceLabel(task.source)}</dd></div>
          <div><dt>{t("taskDetail.sourcingMode")}</dt><dd>{t(task.sourcingMode === "USER_SELECTED" ? "filmCreation.specifiedResource" : "filmCreation.automatic")}</dd></div>
          {presentation.metaLine ? <div><dt>{t("taskDetail.category")}</dt><dd>{presentation.metaLine}</dd></div> : null}
          {film?.endpoint ? <TaskValue label={t("taskDetail.originalMetadataSource")} value={film.endpoint} onCopy={onCopy} /> : null}
          {task.stashScanPath?.trim() ? <TaskValue label={t("taskInfo.libraryLocation")} value={task.stashScanPath.trim()} onCopy={onCopy} /> : null}
          {performers.length > 3 ? <div className="task-detail-metadata__wide"><dt>{t("taskDetail.performers")}</dt><dd>{performers.join(" / ")}</dd></div> : null}
          <div><dt>{t("taskDetail.created")}</dt><dd>{formatDateTime(task.createdAt)}</dd></div>
          <div><dt>{t("taskDetail.updated")}</dt><dd>{formatDateTime(task.updatedAt)}</dd></div>
        </dl>
      </details>
    </article>
  );
}
