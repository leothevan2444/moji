import { useTranslation } from "react-i18next";
import type { IndexerSite } from "../../hooks/useIndexerManagement";
import { IndexerObservationView } from "../indexers/IndexerObservationView";

export function IndexerSiteHistory({ health }: { health: IndexerSite["health"] }) {
  const { t } = useTranslation();
  const entries = (["search", "fetch"] as const).flatMap(kind => {
    const observation = health[kind];
    return observation ? [{ kind, observation }] : [];
  }).sort((a, b) => Number(Boolean(b.observation.errorCode)) - Number(Boolean(a.observation.errorCode)));
  if (!entries.length) return null;
  const summary = [t("indexers.history"), ...entries.map(({ kind, observation }) => t(`indexerHealth.${kind}${observation.errorCode ? "Failed" : "Succeeded"}`))].join(" · ");
  return <details className="indexer-history">
    <summary className={entries.some(entry => entry.observation.errorCode) ? "tone-danger" : undefined} title={summary}>{summary}</summary>
    {entries.map(({ kind, observation }) => <IndexerObservationView key={kind} operation={kind.toUpperCase()} observation={observation} />)}
  </details>;
}
