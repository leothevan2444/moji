import { PreparedTorrentDownloadActions } from "../tasks/PreparedTorrentDownloadActions";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { CandidateSeeders, candidateURL } from "../tasks/TorrentCandidateLinks";
import { formatBytes, formatPublishDate } from "../../utils";
import type { TorrentSearchFieldsFragment } from "../../graphql/generated/graphql";

type Candidate = TorrentSearchFieldsFragment["candidates"][number];
export function TorrentSearchResults({ searchId, candidates, keywords, pending, onQueue }: { searchId: string; candidates: Candidate[]; keywords: string; pending: boolean; onQueue: (id: string, code: string) => Promise<boolean> }) {
  const { t } = useTranslation(); const [selected, setSelected] = useState<string | null>(null); const [code, setCode] = useState("");
  return <div className="discovery-results">{candidates.map(candidate => <article key={candidate.id} className="candidate-card candidate-card--indexer"><div className="candidate-card__content">
    <div className="candidate-card__head"><div><h3 title={candidate.title}>{candidate.title}</h3><p>{candidate.instanceName} · {candidate.size != null ? formatBytes(Number(candidate.size)) : t("torrentSearch.unknownSize")} · <CandidateSeeders seeders={candidate.seeders}/></p></div><span className="torrent-search__kind">{candidate.kind === "BUILTIN" ? t("torrentSearch.builtin") : "Jackett"}</span></div>
    <div className="candidate-card__foot"><span>{formatPublishDate(candidate.publishedAt ?? "")}</span><div className="inline-actions">
      {candidateURL(candidate.detailsUrl) ? <a href={candidate.detailsUrl!} target="_blank" rel="noreferrer noopener" referrerPolicy="no-referrer">{t("sourcingManual.details")}</a> : null}
      <PreparedTorrentDownloadActions searchId={searchId} candidateId={candidate.id} disabled={pending}/><button type="button" disabled={pending} onClick={() => { setSelected(candidate.id); setCode(keywords); }}>{t("discoverUi.result.create")}</button>
    </div></div>
    {selected === candidate.id ? <form className="torrent-search__queue" onSubmit={event => { event.preventDefault(); void onQueue(candidate.id, code).then(ok => { if (ok) setSelected(null); }); }}>
      <label><span>{t("torrentSearch.code")}</span><input aria-label={t("torrentSearch.code")} value={code} autoFocus disabled={pending} onChange={event => setCode(event.target.value)} /></label>
      <button type="submit" disabled={pending || !code.trim()}>{t(pending ? "discoverUi.result.adding" : "torrentSearch.confirmQueue")}</button><button type="button" className="ghost-button" disabled={pending} onClick={() => setSelected(null)}>{t("torrentSearch.close")}</button>
    </form> : null}
  </div></article>)}</div>;
}
