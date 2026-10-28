import { useQuery } from "urql";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";
import { TaskProcessingStateDocument } from "../../graphql/generated/graphql";

export function TaskProcessingNotice() {
  const { t } = useTranslation();
  const [{ data }] = useQuery({ query: TaskProcessingStateDocument, requestPolicy: "cache-and-network" });
  if (data?.settings.automation.taskProcessingEnabled !== false) return null;
  return <aside className="task-processing-notice" role="status">
    <div>
      <strong>{t("taskSettings.pausedTitle")}</strong>
      <p>{t("taskSettings.pausedDetail")}</p>
    </div>
    <Link className="ghost-button" to="/settings/tasks">{t("taskSettings.openSettings")}</Link>
  </aside>;
}
