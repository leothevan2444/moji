import { faFireFlameCurved } from "@fortawesome/free-solid-svg-icons/faFireFlameCurved";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";

export function ActivityLevelIcons({rank, label, title, crossedOut = false}: {rank: number; label: string; title: string; crossedOut?: boolean}) {
  return <span className={`performer-activity-icons${crossedOut ? " is-unknown" : ""}`} role="img" aria-label={label} title={title}>
    {[1,2,3,4,5].map(level => <FontAwesomeIcon key={level} icon={faFireFlameCurved} className={level <= rank ? "is-active" : undefined} aria-hidden="true" />)}
  </span>;
}
