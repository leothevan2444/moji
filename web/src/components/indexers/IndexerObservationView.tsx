import "../../styles/indexer-observation.scss";
import { useTranslation } from "react-i18next";
import type { IndexerObservationFieldsFragment } from "../../graphql/generated/graphql";
import { formatDateTime } from "../../utils";

export function IndexerObservationView({ operation, observation }: { operation: string; observation: IndexerObservationFieldsFragment }) {
  const { t } = useTranslation();
  const kind = operation === "FETCH" ? "fetch" : "search";
  return <div className="indexer-observation">
    <span>{t(`indexerHealth.${kind}${observation.errorCode ? "Failed" : "Succeeded"}`)} · <time dateTime={observation.finishedAt}>{formatDateTime(observation.finishedAt)}</time></span>
    {observation.errorCode ? <span className="tone-danger">{t(`indexerErrors.${observation.errorCode}`, { defaultValue: t("indexers.needsAttention") })}</span> : null}
    {observation.retryAt ? <span>{t("indexerHealth.retryAt", { time: formatDateTime(observation.retryAt) })}</span> : null}
    {observation.errorCode && observation.correlationId ? <details><summary>{t("indexers.details")}</summary><code>{observation.correlationId}</code></details> : null}
  </div>;
}
