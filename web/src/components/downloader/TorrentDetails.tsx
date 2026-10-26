import { useTranslation } from "react-i18next";
import type { DownloaderDocumentQuery } from "../../graphql/generated/graphql";
import { formatBytes, formatDateTime, formatDurationSeconds } from "../../utils/formatters";

export type Torrent = DownloaderDocumentQuery["downloader"]["torrents"][number];

export function TorrentNotice({ torrent }: { torrent: Torrent }) {
  const { t } = useTranslation();
  return <>
    {torrent.state === "missing" || torrent.state === "missingFiles" ? <p className={torrent.needsAttention ? "settings-feedback tone-danger" : "home-muted"}>{t(torrent.allCompleted ? "downloader.completedSourceGone" : "downloader.missingHint")}</p> : null}
    {torrent.cleanupError && torrent.needsAttention ? <p role="alert" className="tone-danger">{t("downloader.cleanupError")}</p> : null}
    {torrent.removedAt ? <p>{t("downloader.removed", { time: formatDateTime(torrent.removedAt), reason: t(`downloader.reasons.${torrent.removalReason}`) })}</p> : null}
  </>;
}
export function TorrentDetails({ torrent, showNotice = true }: { torrent: Torrent; showNotice?: boolean }) {
  const { t } = useTranslation();
  return <div className="torrent-details">{showNotice ? <TorrentNotice torrent={torrent} /> : null}<dl className="download-details">
    <dt>{t("downloader.hash")}</dt><dd>{torrent.hash}</dd>
    <dt>{t("downloader.path")}</dt><dd>{torrent.savePath}</dd>
    <dt>{t("downloader.size")}</dt><dd>{formatBytes(torrent.size)}</dd>
    <dt>{t("downloader.downloaded")}</dt><dd>{formatBytes(torrent.downloaded)}</dd>
    <dt>{t("downloader.uploaded")}</dt><dd>{formatBytes(torrent.uploaded)}</dd>
    <dt>{t("downloader.eta")}</dt><dd>{torrent.eta > 0 && torrent.eta < 8640000 ? formatDurationSeconds(torrent.eta) : t("downloader.unknown")}</dd>
    <dt>{t("downloader.peers")}</dt><dd>{torrent.seeds} / {torrent.peers}</dd>
    <dt>{t("downloader.availability")}</dt><dd>{torrent.availability.toFixed(2)}</dd>
    <dt>{t("downloader.category")}</dt><dd>{torrent.category || "—"}</dd>
    <dt>{t("downloader.tags")}</dt><dd>{torrent.tags || "—"}</dd>
    <dt>{t("downloader.lastSeen")}</dt><dd>{formatDateTime(torrent.lastSeenAt)}</dd>
  </dl></div>;
}
