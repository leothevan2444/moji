import ServiceConnectionSettings from "./ServiceConnectionSettings";
import SettingsPolicySection from "./SettingsPolicySection";
export default function IndexersSettingsTab() {
  return <div className="drawer-stack"><ServiceConnectionSettings service="jackett" /><SettingsPolicySection area="indexers" /></div>;
}
