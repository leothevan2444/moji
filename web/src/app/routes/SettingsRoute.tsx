import { useMemo } from "react";
import { Navigate, useNavigate, useOutletContext, useParams } from "react-router";
import { useQuery } from "urql";
import { SettingsDrawer } from "../../components/drawers/SettingsDrawer";
import { SettingsPageDocumentDocument } from "../../graphql/generated/graphql";
import type { SettingsTab } from "../../types";
import type { AppOutletContext } from "../AppLayout";
import "../../styles/settings.scss";
import { useTranslation } from "react-i18next";

const sectionToTab: Record<string, SettingsTab> = { connections: "connections", ingest: "ingest", automation: "automation", system: "system", logs: "logs", about: "about" };

export function Component() {
  const { t } = useTranslation();
  const { section = "connections" } = useParams();
  const navigate = useNavigate();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const settingsTab = useMemo(() => sectionToTab[section] ?? "connections", [section]);
  const [{ data }, refresh] = useQuery({
    query: SettingsPageDocumentDocument,
    requestPolicy: "cache-and-network"
  });

  if (!sectionToTab[section]) {
    return <Navigate replace to="/settings/connections" />;
  }

  return (
    <section className="section-band">
      <div className="band-head"><div><h2 tabIndex={-1}>{t("settings.title")}</h2></div></div>
      <SettingsDrawer
        settingsTab={settingsTab}
        onSettingsTabChange={(tab) => navigate(`/settings/${tab}`)}
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
