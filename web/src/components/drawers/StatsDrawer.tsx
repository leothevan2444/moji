interface StatsDrawerProps {
  active: number;
  completed: number;
  pendingScans: number;
  failed: number;
}
import { useTranslation } from "react-i18next";

export function StatsDrawer({ active, completed, pendingScans, failed }: StatsDrawerProps) {
  const { t } = useTranslation();
  return (
    <div className="drawer-stack">
      <div className="stat-strip">
        <article className="stat-card">
          <span>{t("stats.active")}</span>
          <strong>{active}</strong>
        </article>
        <article className="stat-card">
          <span>{t("stats.completed")}</span>
          <strong>{completed}</strong>
        </article>
        <article className="stat-card">
          <span>{t("stats.pending")}</span>
          <strong>{pendingScans}</strong>
        </article>
        <article className="stat-card">
          <span>{t("stats.failed")}</span>
          <strong>{failed}</strong>
        </article>
      </div>

      <article className="drawer-card">
        <h3>{t("stats.placeholder")}</h3>
        <p>{t("stats.placeholderDetail")}</p>
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
