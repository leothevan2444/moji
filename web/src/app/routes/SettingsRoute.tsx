import { useMemo } from "react";
import { Navigate, useNavigate, useOutletContext, useParams } from "react-router";
import { useQuery } from "urql";
import { SettingsDrawer } from "../../components/drawers/SettingsDrawer";
import { SettingsPageDocumentDocument } from "../../graphql/generated/graphql";
import type { SettingsTab } from "../../types";
import type { AppOutletContext } from "../AppLayout";
import "../../styles/settings.scss";

const sectionToTab: Record<string, SettingsTab> = {
  connections: "连接",
  ingest: "入库",
  automation: "自动化",
  system: "系统",
  logs: "日志",
  about: "关于"
};
const tabToSection: Record<SettingsTab, string> = {
  连接: "connections",
  入库: "ingest",
  自动化: "automation",
  系统: "system",
  日志: "logs",
  关于: "about"
};

export function Component() {
  const { section = "connections" } = useParams();
  const navigate = useNavigate();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const settingsTab = useMemo(() => sectionToTab[section] ?? "连接", [section]);
  const [{ data }, refresh] = useQuery({
    query: SettingsPageDocumentDocument,
    requestPolicy: "cache-and-network"
  });

  if (!sectionToTab[section]) {
    return <Navigate replace to="/settings/connections" />;
  }

  return (
    <section className="section-band">
      <div className="band-head"><div><h2 tabIndex={-1}>配置与系统</h2></div></div>
      <SettingsDrawer
        settingsTab={settingsTab}
        onSettingsTabChange={(tab) => navigate(`/settings/${tabToSection[tab]}`)}
        runtimeSettings={data?.settings ?? null}
        runtimeStatus={data?.settingsStatus ?? null}
        appVersion={data?.version ?? ""}
        drawer="settings"
        renderedDrawer="settings"
        pushToast={pushToast}
        refreshDashboard={refresh}
      />
    </section>
  );
}
