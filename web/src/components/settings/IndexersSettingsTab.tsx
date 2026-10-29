import { SettingsSubpanel } from "./SettingsPanels";
import ServiceConnectionSettings from "./ServiceConnectionSettings";
import SettingsPolicySection from "./SettingsPolicySection";
export default function IndexersSettingsTab() {
  return <div className="drawer-stack"><SettingsSubpanel id="connection"><ServiceConnectionSettings service="jackett" /></SettingsSubpanel><SettingsSubpanel id="queryOrder"><SettingsPolicySection area="indexers" /></SettingsSubpanel></div>;
}
