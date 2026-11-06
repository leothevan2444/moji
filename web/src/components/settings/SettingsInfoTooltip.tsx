import "../../styles/settings-info.scss";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCircleInfo } from "@fortawesome/free-solid-svg-icons";
import { Tooltip } from "../common/Tooltip";

export function SettingsInfoTooltip({ info }: { info: string }) {
  return <Tooltip info={info} className="settings-info"><FontAwesomeIcon icon={faCircleInfo} aria-hidden="true" /></Tooltip>;
}
