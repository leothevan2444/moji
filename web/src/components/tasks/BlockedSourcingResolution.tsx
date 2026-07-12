import { FormEvent, useState } from "react";
import { useMutation, useQuery } from "urql";
import { describeQueryError } from "../../services/queryError";
import {
  BlockedTaskTorrentCandidatesDocumentDocument,
  ResolveBlockedSourcingTaskDocumentDocument,
  type BlockedTaskTorrentCandidatesDocumentQuery,
  type BlockedTaskTorrentCandidatesDocumentQueryVariables,
  type ResolveBlockedSourcingTaskDocumentMutation,
  type ResolveBlockedSourcingTaskDocumentMutationVariables
} from "../../graphql/generated/graphql";
import { formatBytes, type DashboardTask } from "../../utils";
import { useTranslation } from "react-i18next";

interface BlockedSourcingResolutionProps {
  task: DashboardTask;
  onResolved: () => void | Promise<void>;
}

type Candidate = BlockedTaskTorrentCandidatesDocumentQuery["blockedTaskTorrentCandidates"][number];

export function BlockedSourcingResolution({ task, onResolved }: BlockedSourcingResolutionProps) {
  const { t } = useTranslation();
  const [searchEnabled, setSearchEnabled] = useState(false);
  const [magnetURL, setMagnetURL] = useState("");
  const [resolvingKey, setResolvingKey] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState("");
  const [{ data, fetching, error }, searchCandidates] = useQuery<
    BlockedTaskTorrentCandidatesDocumentQuery,
    BlockedTaskTorrentCandidatesDocumentQueryVariables
  >({
    query: BlockedTaskTorrentCandidatesDocumentDocument,
    variables: { id: task.id, limit: 50 },
    pause: !searchEnabled,
    requestPolicy: "network-only"
  });
  const [, resolveSourcingTask] = useMutation<
    ResolveBlockedSourcingTaskDocumentMutation,
    ResolveBlockedSourcingTaskDocumentMutationVariables
  >(ResolveBlockedSourcingTaskDocumentDocument);

  const resolve = async (torrentUrl: string, candidate?: Candidate) => {
    const trimmedURL = torrentUrl.trim();
    if (!trimmedURL) return;
    const key = candidate ? `${candidate.tracker}-${candidate.link}` : "manual-magnet";
    setResolvingKey(key);
    setSubmitError("");
    try {
      const result = await resolveSourcingTask({
        id: task.id,
        input: {
          torrentUrl: trimmedURL,
          title: candidate?.title,
          tracker: candidate?.tracker,
          infoHash: candidate?.infoHash,
          size: candidate?.size,
          seeders: candidate?.seeders,
          peers: candidate?.peers
        }
      });
      if (result.error) {
        setSubmitError(describeQueryError(result.error));
        return;
      }
      if (!result.data?.resolveBlockedSourcingTask?.id) {
        setSubmitError(t("taskUi.resolution.noResult"));
        return;
      }
      await onResolved();
    } finally {
      setResolvingKey(null);
    }
  };

  const submitMagnet = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    void resolve(magnetURL);
  };

  const candidates = data?.blockedTaskTorrentCandidates ?? [];

  return (
    <article className="drawer-card sourcing-resolution">
      <div className="drawer-card__head">
        <div>
          <h3>{t("taskUi.resolution.title")}</h3>
          <p>{t("taskUi.resolution.detail", { code: task.code })}</p>
        </div>
      </div>

      <form className="sourcing-resolution__magnet" onSubmit={submitMagnet}>
        <input
          value={magnetURL}
          onChange={(event) => setMagnetURL(event.target.value)}
          placeholder={t("taskUi.resolution.magnetPlaceholder")}
          aria-label={t("taskUi.resolution.magnetLabel")}
        />
        <button type="submit" disabled={!magnetURL.trim() || resolvingKey !== null}>
          {resolvingKey === "manual-magnet" ? t("taskUi.resolution.submitting") : t("taskUi.resolution.useMagnet")}
        </button>
      </form>

      <div className="sourcing-resolution__search">
        <button
          type="button"
          className="ghost-button"
          disabled={fetching || resolvingKey !== null}
          onClick={() => {
            setSubmitError("");
            if (searchEnabled) {
              searchCandidates({ requestPolicy: "network-only" });
            } else {
              setSearchEnabled(true);
            }
          }}
        >
          {fetching ? t("taskUi.resolution.searching") : searchEnabled ? t("taskUi.resolution.searchAgain") : t("taskUi.resolution.search")}
        </button>
        {searchEnabled && !fetching && !error ? <span>{t("taskUi.resolution.found", { count: candidates.length })}</span> : null}
      </div>

      {error ? <div className="task-issue tone-danger"><span>{describeQueryError(error)}</span></div> : null}
      {submitError ? <div className="task-issue tone-danger"><span>{submitError}</span></div> : null}

      {searchEnabled && !fetching && !error && candidates.length === 0 ? (
        <div className="task-issue tone-warn"><span>{t("taskUi.resolution.none")}</span></div>
      ) : null}

      {candidates.length > 0 ? (
        <div className="sourcing-resolution__candidates">
          {candidates.map((candidate) => {
            const torrentURL = candidate.magnetUri || candidate.link;
            const key = `${candidate.tracker}-${candidate.link}`;
            return (
              <div className="sourcing-candidate" key={key}>
                <div>
                  <strong>{candidate.title}</strong>
                  <span>{candidate.tracker} · {formatBytes(Number(candidate.size) || 0)} · {candidate.seeders} seeders</span>
                </div>
                <button
                  type="button"
                  disabled={!torrentURL || resolvingKey !== null}
                  onClick={() => void resolve(torrentURL, candidate)}
                >
                  {resolvingKey === key ? t("taskUi.resolution.selecting") : t("taskUi.resolution.select")}
                </button>
              </div>
            );
          })}
        </div>
      ) : null}
    </article>
  );
}
