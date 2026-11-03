import { SettingsSubpanel } from "./SettingsPanels";
import { IndexerSitesPanel } from "./IndexerSitesPanel";
import { IndexerLibraryPanel } from "./IndexerLibraryPanel";
import { IndexerExternalPanel } from "./IndexerExternalPanel";
import { IndexerManagementProvider } from "./IndexerManagementProvider";
import { IndexerCachePanel } from "./IndexerCachePanel";
import "../../styles/indexer-settings.scss";
import { IndexerNetworkPanel } from "./IndexerNetworkPanel";
export default function IndexersSettingsTab() {
  return <IndexerManagementProvider><div className="drawer-stack"><SettingsSubpanel id="sites"><IndexerSitesPanel /></SettingsSubpanel><SettingsSubpanel id="definitions"><IndexerLibraryPanel /></SettingsSubpanel><SettingsSubpanel id="connection"><IndexerExternalPanel /></SettingsSubpanel><SettingsSubpanel id="cache"><IndexerCachePanel /><IndexerNetworkPanel /></SettingsSubpanel></div></IndexerManagementProvider>;
}
