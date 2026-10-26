import i18n from "../i18n/i18n";

// All qB status copy must match its official WebUI, including [F] and spacing.
// See docs/downloader.md and __fixtures__/qbittorrent-statuses.json for sources.
const states = [
  "error", "missingFiles", "uploading", "stalledUP", "forcedUP", "queuedUP",
  "checkingUP", "downloading", "stalledDL", "metaDL", "forcedMetaDL", "forcedDL",
  "queuedDL", "checkingDL", "queuedForChecking", "checkingResumeData", "moving",
  "stoppedDL", "stoppedUP", "pausedDL", "pausedUP", "allocating", "unknown",
  // These describe Moji observations, not qB API states or official translations.
  "missing", "removed",
] as const;
const canonicalStates = new Map(states.map(state => [state.toLowerCase(), state]));

/** The shared status label for torrent lists, task cards and lifecycle details. */
export function qbittorrentStateLabel(
  state?: string | null,
  translate: (key: string) => string = key => i18n.t(key),
): string {
  const key = canonicalStates.get(state?.trim().toLowerCase() ?? "") ?? "unknown";
  return translate(`downloader.states.${key}`);
}
