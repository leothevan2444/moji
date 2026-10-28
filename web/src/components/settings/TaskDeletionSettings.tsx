import { useTranslation } from "react-i18next";
import { useQuery } from "urql";
import { TaskDeletePolicy, TaskDeletionSettingsDocument } from "../../graphql/generated/graphql";
import { useSettingsAutosave } from "./useSettingsAutosave";
import { FieldLabel, SettingsCard, SettingsSaveStatus, SettingsError, SettingsLoading } from "./SettingsTabCommon";

export default function TaskDeletionSettings() {
  const { t } = useTranslation();
  const [{data,fetching,error},refresh] = useQuery({query:TaskDeletionSettingsDocument});
  const autosave = useSettingsAutosave("tasks:deletion",data?.settings.system.taskDeletePolicy ?? TaskDeletePolicy.KeepOnly,policy=>({tasks:{task_delete_policy:policy}}));
  const {value:policy,setValue:setPolicy}=autosave;
  if (fetching && !data) return <SettingsLoading title={t("taskDeletion.title")} />;
  if (error && !data) return <SettingsError title={t("taskDeletion.title")} error={error} onRetry={()=>refresh({requestPolicy:"network-only"})} />;
  return <SettingsCard title={t("taskDeletion.title")}><form className="settings-form" onSubmit={event=>event.preventDefault()}>
    <label className="settings-field"><FieldLabel text={t("taskDeletion.title")} info={t("taskDeletion.info")} /><select value={policy} onChange={event=>setPolicy(event.target.value as TaskDeletePolicy)}><option value={TaskDeletePolicy.KeepOnly}>{t("taskDeletion.keep")}</option><option value={TaskDeletePolicy.RemoveTorrent}>{t("taskDeletion.removeTorrent")}</option><option value={TaskDeletePolicy.RemoveTorrentAndFiles}>{t("taskDeletion.removeFiles")}</option></select></label>
    <SettingsSaveStatus {...autosave} />
  </form></SettingsCard>;
}
