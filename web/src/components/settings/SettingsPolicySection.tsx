import { useTranslation } from "react-i18next";
import { useOutletContext } from "react-router";
import { useQuery } from "urql";
import type { AppOutletContext } from "../../app/AppLayout";
import { SettingsPolicySectionDocument } from "../../graphql/generated/graphql";
import { PolicySettingsEditor, POLICY_SECTION_TITLE_KEYS, type SettingsPolicyArea } from "./PolicySettingsEditor";
import { SettingsError, SettingsLoading } from "./SettingsTabCommon";
import "../../styles/settings.scss";

export default function SettingsPolicySection({ area }: { area: SettingsPolicyArea }) {
  const { t } = useTranslation();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const [{ data, fetching, error }, refresh] = useQuery({ query: SettingsPolicySectionDocument });
  const title = t(POLICY_SECTION_TITLE_KEYS[area]);
  if (fetching && !data) return <SettingsLoading title={title} />;
  if (error && !data) return <SettingsError title={title} error={error} onRetry={() => refresh({ requestPolicy: "network-only" })} />;
  if (!data) return null;

  return <PolicySettingsEditor area={area} runtimeSettings={data.settings} runtimeStatus={data.settingsStatus} pushToast={pushToast} refreshSection={refresh} />;
}
