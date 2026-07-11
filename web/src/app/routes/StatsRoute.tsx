import { useQuery } from "urql";
import { StatsPageDocumentDocument } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import { StatsDrawer } from "../../components/drawers/StatsDrawer";

export function Component() {
  const [{ data, fetching, error }, refresh] = useQuery({
    query: StatsPageDocumentDocument,
    requestPolicy: "cache-and-network"
  });
  const stats = data?.dashboardStats;

  return (
    <section className="section-band">
      <div className="band-head"><div><h2 tabIndex={-1}>运行概览</h2></div></div>
      {error ? (
        <div className="empty-card"><h3>统计加载失败</h3><p>{describeQueryError(error)}</p><button type="button" onClick={() => refresh({ requestPolicy: "network-only" })}>重试</button></div>
      ) : fetching && !stats ? (
        <div className="skeleton skeleton-card" />
      ) : (
        <StatsDrawer active={stats?.active ?? 0} completed={stats?.completed ?? 0} pendingScans={stats?.pendingScans ?? 0} failed={stats?.failed ?? 0} />
      )}
    </section>
  );
}
