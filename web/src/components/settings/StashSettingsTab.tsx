import { SettingsSubpanel } from "./SettingsPanels";
import StashBoxCacheSettings from "./StashBoxCacheSettings";
import ServiceConnectionSettings from "./ServiceConnectionSettings";
import SettingsPolicySection from "./SettingsPolicySection";
import LibraryDeliverySettings from "./LibraryDeliverySettings";
export default function StashSettingsTab() {
  return <div className="drawer-stack"><SettingsSubpanel id="connection"><ServiceConnectionSettings service="stash" /></SettingsSubpanel><SettingsSubpanel id="delivery"><LibraryDeliverySettings /></SettingsSubpanel><SettingsSubpanel id="metadata"><SettingsPolicySection area="metadata" /><StashBoxCacheSettings /></SettingsSubpanel></div>;
}
