import { useQuery } from "urql";
import { StatsPageDocumentDocument } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import { StatsDrawer } from "../../components/drawers/StatsDrawer";
import { useTranslation } from "react-i18next";

export function Component() {
  const { t } = useTranslation();
  const [{ data, fetching, error }, refresh] = useQuery({
    query: StatsPageDocumentDocument,
    requestPolicy: "cache-and-network"
  });
  const stats = data?.dashboardStats;

  return (
    <section className="section-band">
      <div className="band-head"><div><h2 tabIndex={-1}>{t("stats.title")}</h2></div></div>
      {error ? (
        <div className="empty-card"><h3>{t("stats.loadFailed")}</h3><p>{describeQueryError(error)}</p><button type="button" onClick={() => refresh({ requestPolicy: "network-only" })}>{t("common.retry")}</button></div>
      ) : fetching && !stats ? (
        <div className="skeleton skeleton-card" />
      ) : (
        <StatsDrawer active={stats?.active ?? 0} completed={stats?.completed ?? 0} pendingScans={stats?.pendingScans ?? 0} failed={stats?.failed ?? 0} />
      )}
    </section>
  );
}
