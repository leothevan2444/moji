import StashBoxCacheSettings from "./StashBoxCacheSettings";
import ServiceConnectionSettings from "./ServiceConnectionSettings";
import SettingsPolicySection from "./SettingsPolicySection";
import LibraryDeliverySettings from "./LibraryDeliverySettings";
export default function StashSettingsTab() {
  return <div className="drawer-stack"><ServiceConnectionSettings service="stash" /><SettingsPolicySection area="metadata" /><LibraryDeliverySettings /><StashBoxCacheSettings /></div>;
}
