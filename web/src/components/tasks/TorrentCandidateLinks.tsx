import { useTranslation } from "react-i18next";

// Only use the URLs supplied by the indexer. Never derive a details URL from
// a download URL, which may carry a private tracker passkey or Jackett API key.
export function candidateURL(value: string | null | undefined, magnet = false): string | undefined {
  if (!value?.trim()) return undefined;
  try {
    const url = new URL(value.trim());
    if (url.username || url.password) return undefined;
    return (magnet ? url.protocol === "magnet:" : ["http:", "https:"].includes(url.protocol)) ? value.trim() : undefined;
  } catch { return undefined; }
}

export function CandidateSeeders({ seeders }: { seeders?: number | null }) {
  const { t } = useTranslation();
  return <span className={seeders != null && seeders > 0 ? "has-seeders" : undefined}>
    {seeders != null && Number.isInteger(seeders) && seeders >= 0 ? t("sourcingManual.seeders", { count: seeders }) : t("sourcingManual.unknownSeeders")}
  </span>;
}

export function TorrentCandidateLinks({ candidate }: { candidate: { magnetUri?: string; link?: string; details?: string } }) {
  const { t } = useTranslation();
  const links = [
    { href: candidateURL(candidate.magnetUri, true) || candidateURL(candidate.link, true), label: "sourcingManual.magnet", magnet: true },
    { href: candidateURL(candidate.link), label: "sourcingManual.download", magnet: false },
    { href: candidateURL(candidate.details), label: "sourcingManual.details", magnet: false }
  ].filter(link => link.href);
  return <div className="torrent-candidate-links">
    {links.length ? links.map(link => <a key={link.label} href={link.href} target={link.magnet ? undefined : "_blank"}
      rel="noopener noreferrer" referrerPolicy="no-referrer">{t(link.label)}</a>) : <span>{t("sourcingManual.noLinks")}</span>}
  </div>;
}
