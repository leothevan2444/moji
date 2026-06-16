interface StatsDrawerProps {
  active: number;
  completed: number;
  pendingScans: number;
  failed: number;
}

export function StatsDrawer({ active, completed, pendingScans, failed }: StatsDrawerProps) {
  return (
    <div className="drawer-stack">
      <div className="stat-strip">
        <article className="stat-card">
          <span>活跃任务</span>
          <strong>{active}</strong>
        </article>
        <article className="stat-card">
          <span>完成任务</span>
          <strong>{completed}</strong>
        </article>
        <article className="stat-card">
          <span>待扫描</span>
          <strong>{pendingScans}</strong>
        </article>
        <article className="stat-card">
          <span>失败</span>
          <strong>{failed}</strong>
        </article>
      </div>

      <article className="drawer-card">
        <h3>指标占位</h3>
        <p>后续可在这里接入速度、队列、成功率和时段趋势图。</p>
        <div className="mini-bars" aria-hidden="true">
          <span style={{ height: "35%" }} />
          <span style={{ height: "65%" }} />
          <span style={{ height: "50%" }} />
          <span style={{ height: "80%" }} />
          <span style={{ height: "42%" }} />
          <span style={{ height: "70%" }} />
        </div>
      </article>
    </div>
  );
}
