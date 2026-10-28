import SettingsPolicySection from "./SettingsPolicySection";
import TaskDeletionSettings from "./TaskDeletionSettings";
export default function TaskSubscriptionSettingsTab() {
  return <div className="drawer-stack"><SettingsPolicySection area="tasks" /><SettingsPolicySection area="resources" /><TaskDeletionSettings /></div>;
}
