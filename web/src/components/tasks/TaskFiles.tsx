import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCopy } from "@fortawesome/free-solid-svg-icons/faCopy";
import { faFileVideo } from "@fortawesome/free-solid-svg-icons/faFileVideo";
import { faClosedCaptioning } from "@fortawesome/free-solid-svg-icons/faClosedCaptioning";
import { faFile } from "@fortawesome/free-solid-svg-icons/faFile";
import { faArrowUp } from "@fortawesome/free-solid-svg-icons/faArrowUp";
import { faArrowDown } from "@fortawesome/free-solid-svg-icons/faArrowDown";
import { useEffect, useId, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useMutation } from "urql";
import { useTranslation } from "react-i18next";
import { ResolveTaskFilesDocumentDocument } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import { TaskValue } from "./TaskValue";
import { formatBytes, isMagnetLink, needsTaskFileSelection, type DashboardTask } from "../../utils";

type TaskFile = NonNullable<DashboardTask["files"]>[number];
const isVideo = (file: TaskFile) => ["MAIN", "UNKNOWN_VIDEO"].includes(file.kind);
const filename = (path: string) => path.split(/[\\/]/).pop() || path;
const fileSize = (file: TaskFile) => Math.max(0, Number(file.size) || 0);

export function TaskFiles({ task, onCopy }: { task: DashboardTask; onCopy?: (value: string, message: string) => void | Promise<void> }) {
  const { t } = useTranslation();
  const headingId = useId();
  const files = task.files ?? [];
  const initial = useMemo(() => files.filter(file => file.selected && file.kind === "MAIN").sort((a, b) => a.part - b.part).map(file => file.index), [files]);
  const [selected, setSelected] = useState(initial);
  const [message, setMessage] = useState("");
  const [failed, setFailed] = useState(false);
  const focusAfterUpdate = useRef<HTMLElement | null>(null);
  const [{ fetching }, resolve] = useMutation(ResolveTaskFilesDocumentDocument);
  const signature = initial.join(",");
  useEffect(() => { setSelected(initial); setMessage(""); setFailed(false); }, [task.id, signature]);
  useLayoutEffect(() => {
    const element = focusAfterUpdate.current;
    if (element?.isConnected) {
      const target = element.matches(":disabled") ? element.closest("li")?.querySelector<HTMLInputElement>("input") : element;
      target?.focus({ preventScroll: true });
    }
    focusAfterUpdate.current = null;
  }, [selected]);
  const editable = needsTaskFileSelection(task);
  if (!files.length && !task.torrentName?.trim() && !task.torrentHash?.trim() && !task.torrentUrl?.trim()) return null;
  const selectedVideoFiles = selected.map(index => files.find(file => file.index === index)).filter((file): file is TaskFile => Boolean(file));
  const candidates = [...selectedVideoFiles, ...files.filter(file => isVideo(file) && !selected.includes(file.index))];
  const included = files.filter(file => file.selected).sort((a, b) => (a.kind === "MAIN" ? 0 : 1) - (b.kind === "MAIN" ? 0 : 1) || a.part - b.part);
  const subtitles = files.filter(file => file.selected && file.kind === "SUBTITLE");
  const movieCount = files.filter(file => file.selected && file.kind === "MAIN").length;
  const skipped = files.filter(file => !file.selected && !(editable && isVideo(file)));
  const groups = editable
    ? [{ key: "choose", label: t("fileList.choose"), files: candidates }, { key: "included", label: t("fileList.subtitles"), files: subtitles }, { key: "skipped", label: t("mediaFiles.skipped"), files: skipped }]
    : [{ key: "included", label: t("fileList.downloads"), files: included }, { key: "skipped", label: t("mediaFiles.skipped"), files: skipped }];
  const move = (position: number, direction: number, element: HTMLElement) => {
    focusAfterUpdate.current = element;
    setSelected(current => {
      const next = [...current];
      [next[position], next[position + direction]] = [next[position + direction], next[position]];
      return next;
    });
  };
  const save = async () => {
    setMessage("");
    try {
      const result = await resolve({ id: task.id, videoIndices: selected });
      setFailed(Boolean(result.error));
      setMessage(result.error ? describeQueryError(result.error) : t("mediaFiles.selectionSaved"));
    } catch (error) {
      setFailed(true);
      setMessage(describeQueryError(error));
    }
  };

  return <article key={task.id} className="drawer-card task-files" aria-labelledby={headingId}>
    <div className="task-files__heading">
      <h3 id={headingId}>{t("fileList.title")}</h3>
      {files.length > 0 ? <span className="task-files__count">{t("fileList.fileCount", { count: files.length })} · {formatBytes(files.reduce((sum, file) => sum + fileSize(file), 0))}</span> : null}
    </div>
    <dl className="torrent-info">
      <TaskValue className="torrent-info__name" label={t("taskDetail.torrentName")} value={task.torrentName?.trim()} onCopy={onCopy} />
      {task.torrentHash?.trim() ? <TaskValue label={t("taskDetail.torrentHash")} value={task.torrentHash.trim()} onCopy={onCopy} /> : null}
      {task.torrentUrl?.trim() ? <TaskValue label={t(isMagnetLink(task.torrentUrl) ? "taskDetail.magnet" : "taskExperience.torrentUrl")} value={task.torrentUrl.trim()} onCopy={onCopy} /> : null}
    </dl>
    {files.length > 0 ? <div className="task-files__groups">
      {groups.filter(group => group.files.length > 0).map(group => <section key={group.key} className={`task-file-group task-file-group--${group.key}`} aria-label={group.label}>
        <div className="task-file-group__heading">
          <h4>{group.label}<span>{group.files.length}</span></h4>
          <span>{formatBytes(group.files.reduce((sum, file) => sum + fileSize(file), 0))}</span>
        </div>
        <ul className="task-file-list">
          {group.files.map(file => {
            const selectable = editable && isVideo(file);
            const position = selectable ? selected.indexOf(file.index) : -1;
            const included = selectable ? position >= 0 : file.selected;
            const name = filename(file.path);
            const type = file.kind === "SUBTITLE" ? t("fileList.subtitle") : isVideo(file) ? t("fileList.movie") : t("fileList.other");
            return <li key={file.index} className={`task-file-row ${included ? "is-included" : group.key === "choose" ? "is-review" : "is-skipped"}`}>
              <div className="task-file-row__marker">
                {selectable ? <label className="task-file-row__select">
                  <input type="checkbox" checked={included} disabled={fetching} aria-label={name} onChange={event => {
                    focusAfterUpdate.current = event.currentTarget;
                    const checked = event.currentTarget.checked;
                    setSelected(current => checked ? [...current, file.index] : current.filter(index => index !== file.index));
                  }} />
                </label> : <span className="task-file-row__icon"><FontAwesomeIcon icon={file.kind === "SUBTITLE" ? faClosedCaptioning : isVideo(file) ? faFileVideo : faFile} /></span>}
              </div>
              <div className="task-file-row__identity">
                <span className="task-file-row__name" title={file.path}>{name}</span>
                <span className="task-file-row__meta">
                  <span>{formatBytes(fileSize(file))}</span>
                  <span>{group.key === "skipped" ? t(`mediaFiles.reasons.${file.reason}`, { defaultValue: t("fileList.other") }) : type}</span>
                </span>
              </div>
              {selectable && included ? <div className="task-file-row__order" role="group" aria-label={t("fileList.orderFor", { file: name })}>
                <span className="task-file-row__part" aria-label={t("fileList.part", { position: position + 1 })}>{String(position + 1).padStart(2, "0")}</span>
                {selected.length > 1 ? <>
                  <button type="button" className="ghost-button" disabled={fetching || position === 0} onClick={event => move(position, -1, event.currentTarget)} aria-label={t("mediaFiles.moveUp", { file: file.path })}><FontAwesomeIcon icon={faArrowUp} /></button>
                  <button type="button" className="ghost-button" disabled={fetching || position === selected.length - 1} onClick={event => move(position, 1, event.currentTarget)} aria-label={t("mediaFiles.moveDown", { file: file.path })}><FontAwesomeIcon icon={faArrowDown} /></button>
                </> : null}
              </div> : !editable && file.kind === "MAIN" && included && file.part > 0 && movieCount > 1 ? <span className="task-file-row__part" aria-label={t("fileList.part", { position: file.part })}>{String(file.part).padStart(2, "0")}</span> : null}
              {onCopy ? <button type="button" className="ghost-button task-file-row__copy" aria-label={t("taskExperience.copy", { label: file.path })} onClick={() => void onCopy(file.path, t("taskExperience.copied", { label: file.path }))}><FontAwesomeIcon icon={faCopy} /></button> : null}
            </li>;
          })}
        </ul>
      </section>)}
    </div> : null}
    {editable && files.length > 0 ? <div className="task-files__footer">
      <span>{t("fileList.selected", { count: selected.length })}<strong>{formatBytes(selectedVideoFiles.reduce((sum, file) => sum + fileSize(file), 0))}</strong></span>
      <button type="button" className="task-primary-action" disabled={fetching || !selected.length} aria-busy={fetching} onClick={() => void save()}>{t(fetching ? "taskExperience.actions.pending" : "fileList.confirm")}</button>
    </div> : null}
    {message ? <p className={`task-files__message${failed ? " is-error" : ""}`} role={failed ? "alert" : "status"}>{message}</p> : null}
  </article>;
}
