import { SettingsSubpanel } from "./SettingsPanels";
import SettingsPolicySection from "./SettingsPolicySection";
import TaskDeletionSettings from "./TaskDeletionSettings";
export default function TaskSubscriptionSettingsTab() {
  return <div className="drawer-stack"><SettingsSubpanel id={["schedule", "releases"]}><SettingsPolicySection area="tasks" /></SettingsSubpanel><SettingsSubpanel id="resources"><SettingsPolicySection area="resources" /></SettingsSubpanel><SettingsSubpanel id="deletion"><TaskDeletionSettings /></SettingsSubpanel></div>;
}
