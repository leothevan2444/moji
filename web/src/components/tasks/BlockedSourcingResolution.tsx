import { type FormEvent, useId, useRef, useState } from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faArrowUpFromBracket } from "@fortawesome/free-solid-svg-icons/faArrowUpFromBracket";
import { faRotateRight } from "@fortawesome/free-solid-svg-icons/faRotateRight";
import { faSpinner } from "@fortawesome/free-solid-svg-icons/faSpinner";
import { useMutation, useQuery } from "urql";
import { describeQueryError } from "../../services/queryError";
import {
  BlockedTaskTorrentCandidatesDocumentDocument,
  ResolveBlockedSourcingTaskDocumentDocument,
  type BlockedTaskTorrentCandidatesDocumentQuery,
  type BlockedTaskTorrentCandidatesDocumentQueryVariables,
  type ResolveBlockedSourcingTaskDocumentMutation,
  type ResolveBlockedSourcingTaskDocumentMutationVariables,
  type ResolveBlockedSourcingTaskInput
} from "../../graphql/generated/graphql";
import { formatBytes, type DashboardTask } from "../../utils";
import { useTranslation } from "react-i18next";

interface BlockedSourcingResolutionProps {
  task: DashboardTask;
  onResolved: (task: Pick<DashboardTask, "id" | "stage" | "stageStatus">) => void | Promise<void>;
  onNotify?: (tone: "tone-success" | "tone-danger", message: string) => void;
  onBusyChange?: (busy: boolean) => void;
  initialMagnetURL?: string;
  onMagnetChange?: (value: string) => void;
}

type Candidate = BlockedTaskTorrentCandidatesDocumentQuery["blockedTaskTorrentCandidates"][number];

export function BlockedSourcingResolution({ task, onResolved, onNotify, onBusyChange, initialMagnetURL = "", onMagnetChange }: BlockedSourcingResolutionProps) {
  const { t } = useTranslation();
  const headingId = useId();
  const fileInput = useRef<HTMLInputElement>(null);
  const submitting = useRef(false);
  const [magnetURL, setMagnetURL] = useState(initialMagnetURL);
  const [resolvingKey, setResolvingKey] = useState<string | null>(null);
  const [uploadName, setUploadName] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [{ data, fetching, error }, searchCandidates] = useQuery<
    BlockedTaskTorrentCandidatesDocumentQuery,
    BlockedTaskTorrentCandidatesDocumentQueryVariables
  >({
    query: BlockedTaskTorrentCandidatesDocumentDocument,
    variables: { id: task.id, limit: 50 },
    requestPolicy: "network-only"
  });
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

  const resolve = async (torrentUrl: string, candidate?: Candidate, key = "manual-magnet") => {
    if (!torrentUrl.trim()) return;
    await submit({ torrentUrl: torrentUrl.trim(), title: candidate?.title, tracker: candidate?.tracker,
      infoHash: candidate?.infoHash, size: candidate?.size, seeders: candidate?.seeders, peers: candidate?.peers }, key);
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
  const candidates = data?.blockedTaskTorrentCandidates ?? [];
  const busy = resolvingKey !== null;

  return <article className="drawer-card sourcing-resolution" aria-label={t("taskExperience.actions.resolve")}>
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

    <section className="sourcing-resolution__results" aria-labelledby={headingId} aria-busy={fetching}>
      <div className="sourcing-resolution__search">
        <div className="sourcing-resolution__result-heading">
          <h3 id={headingId}>{t("sourcingManual.candidates")}</h3>
          {data && !fetching && !error ? <span>{t("taskUi.resolution.found", { count: candidates.length })}</span> : null}
        </div>
        <button type="button" className="ghost-button" disabled={fetching || busy} onClick={() => searchCandidates({ requestPolicy: "network-only" })}>
          <FontAwesomeIcon icon={fetching ? faSpinner : faRotateRight} aria-hidden="true" />
          {t(fetching ? "taskUi.resolution.searching" : "taskUi.resolution.searchAgain")}
        </button>
      </div>
      {fetching ? <div className="sourcing-resolution__loading" role="status">
        <span>{t("sourcingManual.searchingFor", { code: task.code })}</span>
        {!candidates.length ? <div className="sourcing-resolution__skeletons" aria-hidden="true"><span /><span /><span /></div> : null}
      </div> : null}
      {error ? <div className="sourcing-resolution__error" role="alert">{describeQueryError(error)}</div> : null}
      {data && !fetching && !error && candidates.length === 0 ? <p className="sourcing-resolution__empty">{t("taskUi.resolution.none")}</p> : null}
      {candidates.length > 0 ? <ul className="sourcing-resolution__candidates">
        {candidates.map((candidate, index) => {
          const torrentURL = candidate.magnetUri || candidate.link;
          const key = `${candidate.tracker}-${candidate.infoHash || torrentURL}-${index}`;
          return <li className="sourcing-candidate" key={key}>
            <span className="sourcing-candidate__rank" aria-hidden="true">{String(index + 1).padStart(2, "0")}</span>
            <div className="sourcing-candidate__copy">
              <strong title={candidate.title}>{candidate.title}</strong>
              <div className="sourcing-candidate__meta">
                {candidate.tracker ? <span className="sourcing-candidate__tracker" title={candidate.tracker}>{candidate.tracker}</span> : null}
                <span>{formatBytes(Number(candidate.size) || 0)}</span>
                <span className={candidate.seeders > 0 ? "has-seeders" : undefined}>{t("sourcingManual.seeders", { count: candidate.seeders })}</span>
              </div>
            </div>
            <button type="button" className="ghost-button task-primary-action" disabled={!torrentURL || busy} aria-busy={resolvingKey === key}
              aria-label={t("sourcingManual.selectCandidate", { title: candidate.title })} onClick={() => void resolve(torrentURL, candidate, key)}>
              {t(resolvingKey === key ? "taskUi.resolution.selecting" : "taskUi.resolution.select")}
            </button>
          </li>;
        })}
      </ul> : null}
    </section>
  </article>;
}
