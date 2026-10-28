import { useTranslation } from "react-i18next";
import { useOutletContext } from "react-router";
import { useMutation, useQuery } from "urql";
import type { AppOutletContext } from "../../app/AppLayout";
import { SettingsSection, TaskDeletePolicy, TaskDeletionSettingsDocument, UpdateSettingsSectionDocument } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
import { useSettingsDraft } from "./SettingsDraftStore";
import { FieldLabel, SettingsCard, SettingsDraftNotice, SettingsError, SettingsLoading } from "./SettingsTabCommon";

export default function TaskDeletionSettings() {
  const { t } = useTranslation();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const [{data,fetching,error},refresh] = useQuery({query:TaskDeletionSettingsDocument});
  const [{fetching:saving},save] = useMutation(UpdateSettingsSectionDocument);
  const [policy,setPolicy,dirty,markSaved] = useSettingsDraft("tasks:deletion",data?.settings.system.taskDeletePolicy ?? TaskDeletePolicy.KeepOnly);
  if (fetching && !data) return <SettingsLoading title={t("taskDeletion.title")} />;
  if (error && !data) return <SettingsError title={t("taskDeletion.title")} error={error} onRetry={()=>refresh({requestPolicy:"network-only"})} />;
  return <SettingsCard title={t("taskDeletion.title")}><form className="settings-form" onSubmit={event=>{
    event.preventDefault();
    void save({input:{section:SettingsSection.TaskDeletion,taskDeletePolicy:policy}}).then(result=>{
      if (result.error) {pushToast("tone-danger",describeQueryError(result.error));return;}
      markSaved(policy);pushToast("tone-success",t("taskDeletion.saved"));refresh({requestPolicy:"network-only"});
    });
  }}>
    <label className="settings-field"><FieldLabel text={t("taskDeletion.title")} info={t("taskDeletion.info")} /><select value={policy} onChange={event=>setPolicy(event.target.value as TaskDeletePolicy)}><option value={TaskDeletePolicy.KeepOnly}>{t("taskDeletion.keep")}</option><option value={TaskDeletePolicy.RemoveTorrent}>{t("taskDeletion.removeTorrent")}</option><option value={TaskDeletePolicy.RemoveTorrentAndFiles}>{t("taskDeletion.removeFiles")}</option></select></label>
    <SettingsDraftNotice dirty={dirty} /><div className="settings-actions"><button type="submit" disabled={saving}>{t(saving?"settings.saving":"settings.save")}</button></div>
  </form></SettingsCard>;
}
