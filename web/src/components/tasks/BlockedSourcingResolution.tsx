import { TaskIndexerSearch } from "./TaskIndexerSearch";
import { type FormEvent, useRef, useState } from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowUpFromBracket } from "@fortawesome/free-solid-svg-icons/faArrowUpFromBracket";
import { faSpinner } from "@fortawesome/free-solid-svg-icons/faSpinner";
import { useMutation } from "urql";
import { describeQueryError } from "../../services/queryError";
import {
  ResolveBlockedSourcingTaskDocumentDocument,
  type ResolveBlockedSourcingTaskDocumentMutation,
  type ResolveBlockedSourcingTaskDocumentMutationVariables,
  type ResolveBlockedSourcingTaskInput
} from "../../graphql/generated/graphql";
import { type DashboardTask } from "../../utils";
import { useTranslation } from "react-i18next";

interface BlockedSourcingResolutionProps {
  task: DashboardTask;
  onResolved: (task: Pick<DashboardTask, "id" | "stage" | "stageStatus">) => void | Promise<void>;
  onNotify?: (tone: "tone-success" | "tone-danger", message: string) => void;
  onBusyChange?: (busy: boolean) => void;
  initialMagnetURL?: string;
  onMagnetChange?: (value: string) => void;
}

export function BlockedSourcingResolution({ task, onResolved, onNotify, onBusyChange, initialMagnetURL = "", onMagnetChange }: BlockedSourcingResolutionProps) {
  const { t } = useTranslation();
  const fileInput = useRef<HTMLInputElement>(null);
  const submitting = useRef(false);
  const [magnetURL, setMagnetURL] = useState(initialMagnetURL);
  const [resolvingKey, setResolvingKey] = useState<string | null>(null);
  const [uploadName, setUploadName] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [, resolveSourcingTask] = useMutation<
    ResolveBlockedSourcingTaskDocumentMutation,
    ResolveBlockedSourcingTaskDocumentMutationVariables
  >(ResolveBlockedSourcingTaskDocumentDocument);

  const reportError = (message: string) => {
    setSubmitError(message);
    onNotify?.("tone-danger", message);
  };
  const submit = async (input: ResolveBlockedSourcingTaskInput, key: string) => {
    if (submitting.current) return;
    submitting.current = true;
    onBusyChange?.(true);
    setResolvingKey(key);
    setSubmitError("");
    try {
      const result = await resolveSourcingTask({ id: task.id, input });
      if (result.error) {
        reportError(describeQueryError(result.error));
        return;
      }
      if (!result.data?.resolveBlockedSourcingTask?.id) {
        reportError(t("taskUi.resolution.noResult"));
        return;
      }
      onNotify?.("tone-success", t("sourcingManual.accepted"));
      await onResolved(result.data.resolveBlockedSourcingTask);
    } catch (error) {
      reportError(describeQueryError(error));
    } finally {
      submitting.current = false;
      setResolvingKey(null);
      onBusyChange?.(false);
    }
  };

  const resolve = async (torrentUrl: string, key = "manual-magnet") => {
    if (!torrentUrl.trim()) return;
    await submit({ torrentUrl: torrentUrl.trim() }, key);
  };
  const submitMagnet = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    void resolve(magnetURL);
  };
  const selectFile = (file: File | undefined) => {
    if (!file || submitting.current) return;
    setSubmitError("");
    if (!file.name.toLowerCase().endsWith(".torrent") || file.size === 0 || file.size > 8 * 1024 * 1024) {
      reportError(t("sourcingManual.invalidFile"));
      return;
    }
    setUploadName(file.name);
    void submit({ torrentFile: file }, "torrent-file");
  };
  const [indexedBusy, setIndexedBusy] = useState(false);
  const busy = resolvingKey !== null || indexedBusy;

  return <article className="drawer-card sourcing-resolution" aria-label={t("taskExperience.actions.resolve")}>
    <section className="sourcing-resolution__manual">
    <p className="sourcing-resolution__intro">{t("sourcingManual.intro")}</p>
    <form className="sourcing-resolution__magnet" onSubmit={submitMagnet}>
      <input disabled={busy} value={magnetURL} onChange={event => { setMagnetURL(event.target.value); onMagnetChange?.(event.target.value); }}
        placeholder={t("taskUi.resolution.magnetPlaceholder")} aria-label={t("taskUi.resolution.magnetLabel")} />
      <button type="submit" className="ghost-button task-primary-action" disabled={!magnetURL.trim() || busy} aria-busy={resolvingKey === "manual-magnet"}>
        {t(resolvingKey === "manual-magnet" ? "taskUi.resolution.submitting" : "taskUi.resolution.useMagnet")}
      </button>
      <button type="button" className="ghost-button sourcing-resolution__upload" disabled={busy} aria-busy={resolvingKey === "torrent-file"}
        title={t("sourcingManual.fileHelp")} onClick={() => fileInput.current?.click()}>
        <FontAwesomeIcon icon={resolvingKey === "torrent-file" ? faSpinner : faArrowUpFromBracket} aria-hidden="true" />
        {t(resolvingKey === "torrent-file" ? "sourcingManual.validating" : "sourcingManual.fileLabel")}
      </button>
      <input ref={fileInput} type="file" hidden accept=".torrent,application/x-bittorrent" disabled={busy}
        aria-label={t("sourcingManual.fileLabel")} onChange={event => {
          const file = event.currentTarget.files?.[0];
          event.currentTarget.value = ""; // Allow retrying the same file after validation fails.
          selectFile(file);
        }} />
    </form>
    {resolvingKey === "torrent-file" ? <p className="sourcing-resolution__feedback" role="status">{t("sourcingManual.validatingFile", { name: uploadName })}</p> : null}
    {submitError ? <div className="sourcing-resolution__error" role="alert">{submitError}</div> : null}

    </section>
    <TaskIndexerSearch task={task} disabled={busy} onResolved={onResolved} onNotify={onNotify}
      onBusyChange={value => { setIndexedBusy(value); onBusyChange?.(value); }} />
  </article>;
}
