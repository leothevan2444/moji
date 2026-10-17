import { useEffect, useMemo, useState } from "react";
import { useMutation } from "urql";
import { useTranslation } from "react-i18next";
import { ResolveTaskFilesDocumentDocument } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import type { DashboardTask } from "../../utils";

export function TaskFiles({ task }: { task: DashboardTask }) {
  const { t } = useTranslation();
  const files = task.files ?? [];
  const initial = useMemo(() => files.filter(file => file.selected && file.kind === "MAIN").sort((a, b) => a.part - b.part).map(file => file.index), [files]);
  const [selected, setSelected] = useState(initial);
  const [message, setMessage] = useState("");
  const [{ fetching }, resolve] = useMutation(ResolveTaskFilesDocumentDocument);
  const signature = initial.join(",");
  useEffect(() => { setSelected(initial); setMessage(""); }, [task.id, signature]);
  const editable = task.stage === "DOWNLOADING" && task.stageStatus === "BLOCKED";
  if (!files.length) return null;
  const move = (position: number, direction: number) => setSelected(current => {
    const next = [...current];
    [next[position], next[position + direction]] = [next[position + direction], next[position]];
    return next;
  });
  const save = async () => {
    setMessage("");
    const result = await resolve({ id: task.id, videoIndices: selected });
    setMessage(result.error ? describeQueryError(result.error) : t("mediaFiles.selectionSaved"));
  };
  return <article className="drawer-card">
    <h3>{t("mediaFiles.title")}</h3>
    <p>{t("mediaFiles.summary")}</p>
    <ul className="task-file-list">
      {files.map(file => <li key={file.index}>
        {editable && ["MAIN", "UNKNOWN_VIDEO"].includes(file.kind) ? <label>
          <input type="checkbox" checked={selected.includes(file.index)} disabled={fetching} onChange={event => setSelected(current => event.target.checked ? [...current, file.index] : current.filter(index => index !== file.index))} />
          <span>{file.path}</span>
        </label> : <span>{file.path}</span>}
        <small>{t(`mediaFiles.kinds.${file.kind}`)} · {t(file.selected ? "mediaFiles.selected" : file.reason === "IDENTITY_UNCERTAIN" ? "mediaFiles.review" : "mediaFiles.skipped")} · {t(`mediaFiles.reasons.${file.reason}`, { defaultValue: t("mediaFiles.classified") })}</small>
      </li>)}
    </ul>
    {editable ? <>
      <h4>{t("mediaFiles.playbackOrder")}</h4>
      <ol>{selected.map((index, position) => <li key={index}>
        <span>{files.find(file => file.index === index)?.path}</span>
        <button type="button" disabled={fetching || position === 0} onClick={() => move(position, -1)} aria-label={t("mediaFiles.moveUp", { file: files.find(file => file.index === index)?.path })}>{t("common.moveUp")}</button>
        <button type="button" disabled={fetching || position === selected.length - 1} onClick={() => move(position, 1)} aria-label={t("mediaFiles.moveDown", { file: files.find(file => file.index === index)?.path })}>{t("common.moveDown")}</button>
      </li>)}</ol>
      <button type="button" disabled={fetching || !selected.length} onClick={() => void save()}>{t("mediaFiles.confirm")}</button>
    </> : null}
    {message ? <p role="status">{message}</p> : null}
  </article>;
}
