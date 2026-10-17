import { useTranslation } from "react-i18next";
import { useQuery } from "urql";
import { AboutSettingsTabDocument } from "../../graphql/generated/graphql";
import { SettingsCard, SettingsError, SettingsLoading } from "./SettingsTabCommon";
import { APP_VERSION } from "../../version";

export default function AboutSettingsTab() {
  const { t } = useTranslation();
  const [{ data, fetching, error }, refresh] = useQuery({ query: AboutSettingsTabDocument, requestPolicy: "cache-first" });
  if (fetching && !data) return <SettingsLoading title={t("settings.tabs.about")} />;
  if (error && !data) return <SettingsError title={t("settings.tabs.about")} error={error} onRetry={() => refresh({ requestPolicy: "network-only" })} />;
  return <SettingsCard title={t("systemUi.about")}><dl className="settings-grid"><div><dt>{t("systemUi.version")}</dt><dd>{data?.version || APP_VERSION}</dd></div></dl></SettingsCard>;
}
