import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faClipboardCheck } from "@fortawesome/free-solid-svg-icons/faClipboardCheck";
import { useMutation, useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { Drawer } from "../layout/Drawer";
import { BlockedSourcingResolution } from "../tasks/BlockedSourcingResolution";
import { TaskFiles } from "../tasks/TaskFiles";
import { RetryTaskDocumentDocument, TaskDetailDocumentDocument } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import { needsTaskFileSelection, taskFailureSummary, taskGroup, taskSummary, type DashboardTask } from "../../utils";
import { taskProcessingQueue, type TaskQueueDraft } from "../../utils/taskProcessingQueue";

type TaskResult = Pick<DashboardTask, "id" | "stage" | "stageStatus">;
type Notify = (tone: "tone-success" | "tone-danger", message: string) => void;
interface Props {
  tasks: DashboardTask[];
  autoAdvance?: boolean;
  onClose: () => void;
  onCopy: (value: string, message: string) => void | Promise<void>;
  onNotify: Notify;
}

export function TaskProcessingQueueDrawer({ tasks, onClose, onCopy, onNotify, autoAdvance = true }: Props) {
  const { t } = useTranslation();
  const [session] = useState(() => taskProcessingQueue(tasks));
  const [finished, setFinished] = useState<Set<string>>(() => new Set());
  const [currentId, setCurrentId] = useState<string | undefined>(session[0]?.id);
  const [busy, setBusy] = useState(false);
  const [handled, setHandled] = useState<string | null>(null);
  const drafts = useRef(new Map<string, TaskQueueDraft>());
  const heading = useRef<HTMLHeadingElement>(null);
  const liveTasks = useMemo(() => new Map(tasks.map(task => [task.id, task])), [tasks]);
  const remaining = useMemo(() => session.filter(task => !finished.has(task.id)), [session, finished]);
  const index = Math.max(0, remaining.findIndex(task => task.id === currentId));
  const item = remaining[index];
  const task = item ? liveTasks.get(item.id) ?? item : undefined;

  const remove = useCallback((ids: string[]) => {
    if (!ids.length) return;
    const removed = new Set(ids);
    setFinished(current => new Set([...current, ...ids]));
    if (item && removed.has(item.id)) {
      if (!autoAdvance) setHandled(taskSummary(item));
      const next = remaining.slice(index + 1).find(task => !removed.has(task.id))
        ?? remaining.slice(0, index).reverse().find(task => !removed.has(task.id));
      setCurrentId(next?.id);
    }
    for (const id of ids) drafts.current.delete(id);
  }, [autoAdvance, index, item, remaining]);

  useEffect(() => {
    if (busy) return;
    remove(remaining.filter(task => {
      const live = liveTasks.get(task.id);
      return !live || taskGroup(live) !== "attention";
    }).map(task => task.id));
  }, [busy, liveTasks, remaining, remove]);

  useEffect(() => {
    const body = heading.current?.closest(".drawer-body");
    if (body) body.scrollTop = 0;
    heading.current?.focus({ preventScroll: true });
  }, [item?.id, handled]);

  const close = () => { if (!busy) onClose(); };
  const footer = <nav className="task-queue-nav" aria-label={t("taskProcessingQueue.navigation")}>
    {handled ? <button type="button" className="ghost-button" onClick={() => { if (remaining.length) setHandled(null); else close(); }}>{t(remaining.length ? "workbench.nextTask" : "common.close")} →</button> : item ? <>
      <button type="button" className="ghost-button" disabled={busy || index === 0} onClick={() => setCurrentId(remaining[index - 1].id)}>
        <span aria-hidden="true">←</span> {t("taskProcessingQueue.previous")}
      </button>
      <span className="task-queue-nav__position" role="status">{t("taskProcessingQueue.position", { current: index + 1, count: remaining.length })}</span>
      <button type="button" className="ghost-button" disabled={busy || index === remaining.length - 1} onClick={() => setCurrentId(remaining[index + 1].id)}>
        {t("taskProcessingQueue.next")} <span aria-hidden="true">→</span>
      </button>
    </> : <button type="button" className="ghost-button" onClick={close}>{t("common.close")}</button>}
  </nav>;

  return <Drawer visibleDrawer="task-queue" closing={false} title={t("taskProcessingQueue.title")}
    titleIcon={<FontAwesomeIcon icon={faClipboardCheck} />} onClose={close} closeDisabled={busy} footer={footer}>
    <div className="task-queue">
      <section className="task-queue__summary" aria-label={t("taskProcessingQueue.progressLabel")}>
        <div className="task-queue__summary-head">
          <h3>{t("taskProcessingQueue.progressLabel")}</h3>
          <p>{t("taskProcessingQueue.progress", { done: finished.size, remaining: remaining.length })}</p>
        </div>
        {session.length ? <progress aria-label={t("taskProcessingQueue.progressLabel")} max={session.length} value={finished.size} /> : null}
        <span>{t(autoAdvance ? "taskProcessingQueue.hint" : "workbench.queueHint")}</span>
      </section>
      <section className="task-queue__task">
      {handled ? <div className="task-queue__empty"><span aria-hidden="true">✓</span><h3 ref={heading} tabIndex={-1}>{t("workbench.handledTitle", { task: handled })}</h3><p>{t("workbench.handledDetail")}</p></div> : task ? <>
        <div className="task-queue__task-heading">
          <span>{t("taskProcessingQueue.currentTask")}</span>
          <h3 ref={heading} tabIndex={-1} className="task-queue__title">{taskSummary(task)}</h3>
        </div>
        <TaskQueueItem key={task.id} task={task} draft={drafts.current.get(task.id)}
          onDraftChange={draft => drafts.current.set(task.id, { ...drafts.current.get(task.id), ...draft })}
          onBusyChange={setBusy} onCopy={onCopy} onNotify={onNotify} onDone={() => remove([task.id])} />
      </> : <div className="task-queue__empty">
        <span aria-hidden="true">✓</span>
        <h3 ref={heading} tabIndex={-1}>{t(session.length ? "taskProcessingQueue.finished" : "taskProcessingQueue.empty")}</h3>
        <p>{t(session.length ? "taskProcessingQueue.finishedHelp" : "taskProcessingQueue.emptyHelp")}</p>
      </div>}
      </section>
    </div>
  </Drawer>;
}

function TaskQueueItem({ task: snapshot, draft, onDraftChange, onBusyChange, onCopy, onNotify, onDone }: {
  task: DashboardTask;
  draft?: TaskQueueDraft;
  onDraftChange: (draft: TaskQueueDraft) => void;
  onBusyChange: (busy: boolean) => void;
  onCopy: Props["onCopy"];
  onNotify: Notify;
  onDone: () => void;
}) {
  const { t } = useTranslation();
  const [{ data, fetching, error }, refresh] = useQuery({ query: TaskDetailDocumentDocument,
    variables: { id: snapshot.id }, requestPolicy: "cache-and-network" });
  const [{ fetching: retrying }, retry] = useMutation(RetryTaskDocumentDocument);
  const [message, setMessage] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [actionBusy, setActionBusy] = useState(false);
  const submission = useRef(false);
  const task = data?.task?.id === snapshot.id ? { ...snapshot, ...data.task } : snapshot;
  const failure = taskFailureSummary(task);
  const changeBusy = (value: boolean) => { setActionBusy(value); onBusyChange(value); };

  useEffect(() => {
    if (!actionBusy && data?.task?.id === snapshot.id && taskGroup({ ...snapshot, ...data.task }) !== "attention") onDone();
  }, [actionBusy, data?.task, onDone, snapshot]);

  const accept = (result: TaskResult) => {
    if (result.id !== snapshot.id) return;
    if (taskGroup({ ...task, ...result }) !== "attention") onDone();
    else refresh({ requestPolicy: "network-only" });
  };
  const retryTask = async () => {
    if (submission.current) return;
    submission.current = true;
    setSubmitting(true);
    changeBusy(true);
    setMessage("");
    try {
      const result = await retry({ id: task.id });
      if (result.error) throw result.error;
      const next = result.data?.retryTask;
      if (!next?.id) throw new Error(t("taskProcessingQueue.noResult"));
      if (taskGroup({ ...task, ...next }) === "attention") {
        setMessage(t("taskProcessingQueue.stillBlocked"));
        refresh({ requestPolicy: "network-only" });
      } else {
        onNotify("tone-success", t("taskProcessingQueue.retried", { task: taskSummary(task) }));
        accept(next);
      }
    } catch (error) {
      const message = describeQueryError(error);
      setMessage(message);
      onNotify("tone-danger", message);
    } finally {
      submission.current = false;
      setSubmitting(false);
      changeBusy(false);
    }
  };

  if (error) return <div className="empty-card"><p role="alert">{describeQueryError(error)}</p><button type="button" className="ghost-button" disabled={fetching} onClick={() => refresh({ requestPolicy: "network-only" })}>{t("common.retry")}</button></div>;
  if (fetching && !data?.task) return <div className="skeleton skeleton-card" role="status" aria-label={t("taskRoute.loading")} />;
  if (data && !data.task) return <div className="empty-card"><p>{t("taskRoute.notFound")}</p><button type="button" className="ghost-button" onClick={onDone}>{t("taskProcessingQueue.removeMissing")}</button></div>;
  if (taskGroup(task) !== "attention") return <p role="status">{t("taskProcessingQueue.advancing")}</p>;

  const source = task.stage === "SOURCING";
  const files = needsTaskFileSelection(task);
  return <div className="task-queue__content">
    <div className="task-queue__reason">
      <strong>{failure.title}</strong><p>{failure.detail}</p>
      {task.stageErrorMessage ? <details><summary>{t("taskProcessingQueue.errorDetails")}</summary><p>{task.stageErrorMessage}</p></details> : null}
    </div>
    {message ? <p className="task-queue__error" role="alert">{message}</p> : null}
    {source ? <BlockedSourcingResolution task={task} initialMagnetURL={draft?.magnet}
      onMagnetChange={magnet => onDraftChange({ magnet })} onBusyChange={changeBusy} onNotify={onNotify} onResolved={accept} /> : <>
      {!files ? <div className="task-queue__retry">
        <p>{t("taskProcessingQueue.retryHelp")}</p>
        <button type="button" className="ghost-button task-primary-action" disabled={submitting || retrying} aria-busy={submitting} onClick={() => void retryTask()}>
          {t(submitting ? "taskExperience.actions.pending" : "taskExperience.actions.retry")}
        </button>
      </div> : null}
      <TaskFiles task={task} onCopy={onCopy} initialSelection={draft?.videoIndices}
        onSelectionChange={videoIndices => onDraftChange({ videoIndices })} onBusyChange={changeBusy}
        onResolved={result => { onNotify("tone-success", t("mediaFiles.selectionSaved")); accept(result); }} />
    </>}
  </div>;
}
