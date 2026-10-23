import { useCallback, useDeferredValue, useEffect, useRef, useState } from "react";
import { Link, useOutletContext, useSearchParams } from "react-router";
import { useMutation, useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faFilm, faRotate } from "@fortawesome/free-solid-svg-icons";
import { ProcessReleaseReviewsDocument, ReleaseReviewsDocument, ReleaseReviewAction, ReleaseReviewStatus } from "../../graphql/generated/graphql";
import { usePerformerSubscriptionEvents } from "../../hooks/usePerformerSubscriptionEvents";
import { useTaskEvents } from "../../hooks/useTaskEvents";
import { describeQueryError } from "../../services/queryError";
import { formatDateTime } from "../../utils";
import type { AppOutletContext } from "../AppLayout";
import "../../styles/release-review.scss";

const statuses = [ReleaseReviewStatus.Pending, ReleaseReviewStatus.Failed, ReleaseReviewStatus.Downloaded, ReleaseReviewStatus.InLibrary, ReleaseReviewStatus.Rejected, ReleaseReviewStatus.Skipped, ReleaseReviewStatus.All];
const downloadable = (status: ReleaseReviewStatus) => status === ReleaseReviewStatus.Pending || status === ReleaseReviewStatus.Failed;
const reconsiderable = (status: ReleaseReviewStatus) => status === ReleaseReviewStatus.Rejected || status === ReleaseReviewStatus.Skipped;
const safeURL = (url: string | null | undefined) => url && /^https?:\/\//iu.test(url) ? url : null;

export function Component() {
  const { t } = useTranslation();
  const { pushToast } = useOutletContext<AppOutletContext>();
  const [params, setParams] = useSearchParams();
  const search = params.get("q") ?? "";
  const deferredSearch = useDeferredValue(search);
  const statusValue = params.get("status") as ReleaseReviewStatus | null;
  const status = statusValue && statuses.includes(statusValue) ? statusValue : ReleaseReviewStatus.Pending;
  const performerId = params.get("performer") || null;
  const vr = params.get("vr") === "true" ? true : params.get("vr") === "false" ? false : null;
  const page = Math.max(1, Number.parseInt(params.get("page") ?? "1", 10) || 1);
  const [{ data, fetching, error }, reload] = useQuery({ query: ReleaseReviewsDocument, variables: { input: { search: deferredSearch || null, performerId, status, vr, page, pageSize: 24 } }, requestPolicy: "network-only" });
  const [mutation, process] = useMutation(ProcessReleaseReviewsDocument);
  const [selected, setSelected] = useState(new Map<string, { code: string; status: ReleaseReviewStatus }>());
  const [busy, setBusy] = useState<string[]>([]);
  const [results, setResults] = useState<Array<{ key: string; code: string; outcome: string; reasonCode: string; taskId?: string | null }>>([]);
  const running = useRef(false);
  const refresh = useCallback(() => reload({ requestPolicy: "network-only" }), [reload]);
  const events = usePerformerSubscriptionEvents({ enabled: true, onRefresh: refresh });
  useEffect(() => { if (!events.event) return; const timer = window.setTimeout(refresh, 200); return () => window.clearTimeout(timer); }, [events.event, refresh]);
  useTaskEvents({ onEvent: event => { if (event.type === "CREATED") refresh(); }, onReconnect: refresh, onSequenceGap: refresh });
  useEffect(() => {
    const onVisible = () => { if (document.visibilityState === "visible") refresh(); };
    document.addEventListener("visibilitychange", onVisible);
    return () => document.removeEventListener("visibilitychange", onVisible);
  }, [refresh]);
  const connection = data?.releaseReviews;
  const items = connection?.items ?? [];
  const pending = mutation.fetching || busy.length > 0;
  const change = (updates: Record<string, string>) => {
    setParams(current => { const next = new URLSearchParams(current); next.delete("page"); for (const [key, value] of Object.entries(updates)) { if (value) next.set(key, value); else next.delete(key); } return next; }, { replace: true });
  };
  const toggle = (item: (typeof items)[number]) => setSelected(current => {
    const next = new Map(current);
    if (next.has(item.key)) next.delete(item.key);
    else if (next.size < 100) next.set(item.key, { code: item.code || item.title, status: item.status });
    else pushToast("tone-info", t("releaseReview.limit"));
    return next;
  });
  const selectVisible = () => setSelected(current => {
    const next = new Map(current);
    for (const item of items) { if (next.size >= 100) break; if (downloadable(item.status) || reconsiderable(item.status)) next.set(item.key, { code: item.code || item.title, status: item.status }); }
    return next;
  });
  const eligible = (action: ReleaseReviewAction) => [...selected].filter(([, item]) => action === ReleaseReviewAction.Reconsider ? reconsiderable(item.status) : downloadable(item.status)).map(([key]) => key);
  const run = async (keys: string[], action: ReleaseReviewAction) => {
    if (!keys.length || running.current) return;
    running.current = true; setBusy(keys); setResults([]);
    try {
      const response = await process({ input: { keys, action } });
      if (response.error) { pushToast("tone-danger", describeQueryError(response.error)); return; }
      const payload = response.data?.processReleaseReviews;
      if (!payload) { pushToast("tone-danger", t("releaseReview.noResult")); return; }
      setResults(payload.results.map(result => ({ ...result, code: selected.get(result.key)?.code || items.find(item => item.key === result.key)?.code || result.key, taskId: result.release?.taskId })));
      const succeeded = new Set(payload.results.filter(result => result.outcome === "SUCCEEDED").map(result => result.key));
      setSelected(current => {
        const next = new Map(current);
        for (const key of succeeded) next.delete(key);
        for (const result of payload.results) { if (next.has(result.key) && result.release) next.set(result.key, { ...next.get(result.key)!, status: result.release.status }); }
        return next;
      });
      pushToast(payload.failedCount ? "tone-danger" : payload.skippedCount ? "tone-info" : "tone-success", t("releaseReview.resultSummary", { succeeded: payload.succeededCount, skipped: payload.skippedCount, failed: payload.failedCount }));
      refresh();
    } finally { running.current = false; setBusy([]); }
  };

  return <section className="section-band release-review">
    <div className="band-head"><div><h2>{t("releaseReview.title")}</h2><p>{t("releaseReview.intro")}</p></div><Link className="ghost-button release-review-link" to="/performers">{t("performerUi.back")}</Link></div>
    <div className="release-review__counts"><span>{t("releaseReview.pendingCount", { count: connection?.pendingCount ?? 0 })}</span><span>{t("releaseReview.failedCount", { count: connection?.failedCount ?? 0 })}</span></div>
    <div className="release-review__filters">
      <label>{t("releaseReview.search")}<input value={search} onChange={event => change({ q: event.target.value })} placeholder={t("releaseReview.searchPlaceholder")} /></label>
      <label>{t("releaseReview.performer")}<select value={performerId ?? ""} onChange={event => change({ performer: event.target.value })}><option value="">{t("releaseReview.allPerformers")}</option>{connection?.performers.map(actor => <option key={actor.id} value={actor.id}>{actor.name}</option>)}</select></label>
      <label>{t("releaseReview.vr")}<select value={vr === null ? "" : String(vr)} onChange={event => change({ vr: event.target.value })}><option value="">{t("releaseReview.allFormats")}</option><option value="true">{t("releaseReview.onlyVR")}</option><option value="false">{t("releaseReview.nonVR")}</option></select></label>
      <label>{t("releaseReview.status")}<select value={status} onChange={event => change({ status: event.target.value })}>{statuses.map(value => <option key={value} value={value}>{t(`releaseReview.states.${value}`)}</option>)}</select></label>
      <button type="button" className="ghost-button" onClick={refresh} disabled={fetching}><FontAwesomeIcon icon={faRotate} className={fetching ? "is-spinning" : undefined} /> {t("common.refresh")}</button>
    </div>
    {error ? <p role="alert" className="tone-danger">{describeQueryError(error)}</p> : null}
    {fetching ? <p role="status">{t("common.loading")}</p> : null}
    <div className="release-review__batch">
      <span>{t("releaseReview.selected", { count: selected.size })}</span>
      <button className="ghost-button" type="button" onClick={selectVisible} disabled={pending || fetching}>{t("releaseReview.selectPage")}</button>
      <button className="ghost-button" type="button" onClick={() => setSelected(new Map())} disabled={pending || !selected.size}>{t("releaseReview.clear")}</button>
      <button type="button" disabled={pending || !eligible(ReleaseReviewAction.Download).length} onClick={() => void run(eligible(ReleaseReviewAction.Download), ReleaseReviewAction.Download)}>{t("releaseReview.downloadSelected", { count: eligible(ReleaseReviewAction.Download).length })}</button>
      <button type="button" className="ghost-button" disabled={pending || !eligible(ReleaseReviewAction.Reject).length} onClick={() => void run(eligible(ReleaseReviewAction.Reject), ReleaseReviewAction.Reject)}>{t("releaseReview.rejectSelected", { count: eligible(ReleaseReviewAction.Reject).length })}</button>
      {eligible(ReleaseReviewAction.Reconsider).length ? <button type="button" className="ghost-button" disabled={pending} onClick={() => void run(eligible(ReleaseReviewAction.Reconsider), ReleaseReviewAction.Reconsider)}>{t("releaseReview.reconsiderSelected", { count: eligible(ReleaseReviewAction.Reconsider).length })}</button> : null}
    </div>
    {results.length ? <div className="release-review__results" role="status"><h3>{t("releaseReview.results")}</h3>{results.map(result => <p key={result.key}><strong>{result.code}</strong> · {t(`releaseReview.resultsByCode.${result.reasonCode}`)} {result.taskId ? <Link to={`/tasks/${encodeURIComponent(result.taskId)}`}>{t("releaseReview.openTask")}</Link> : null}</p>)}</div> : null}
    <div className="release-review__list" aria-busy={fetching}>
      {!fetching && !error && !items.length ? <article className="empty-card"><h3>{t("releaseReview.empty")}</h3><p>{t("releaseReview.emptyHelp")}</p></article> : null}
      {items.map(item => <article key={item.key} className="release-review__item">
        <input type="checkbox" aria-label={t("releaseReview.select", { code: item.code || item.title })} checked={selected.has(item.key)} disabled={pending || fetching || !(downloadable(item.status) || reconsiderable(item.status))} onChange={() => toggle(item)} />
        <div className="release-review__cover">{item.imageUrl ? <img src={item.imageUrl} alt="" loading="lazy" onError={event => { event.currentTarget.hidden = true; }} /> : <FontAwesomeIcon icon={faFilm} />}</div>
        <div className="release-review__info"><div className="release-review__heading"><h3>{item.code || item.title}</h3>{item.vr ? <span className="tag">VR</span> : null}<span className="tag">{t(`releaseReview.states.${item.status}`)}</span></div><p>{item.title}</p><p>{item.performerNames.join("、") || item.performers.map(actor => actor.name).join("、")} · {item.date || t("releaseReview.unknownDate")}</p><p>{t("releaseReview.reason", { reason: t(`releaseReview.reasons.${item.reasonCode}`, { defaultValue: t("releaseReview.reasons.unknown") }) })}</p><p className="release-review__meta">{t("releaseReview.seenAt", { time: formatDateTime(item.seenAt) })}{item.reviewedAt ? ` · ${t("releaseReview.reviewedAt", { time: formatDateTime(item.reviewedAt) })}` : ""}</p>{item.error ? <details><summary>{t("releaseReview.error")}</summary><p>{item.error}</p></details> : null}{safeURL(item.url) ? <a href={safeURL(item.url)!} target="_blank" rel="noreferrer">{t("releaseReview.openSource")}</a> : null}</div>
        <div className="release-review__actions">{busy.includes(item.key) ? <span role="status">{t("common.loading")}</span> : null}{downloadable(item.status) ? <><button type="button" disabled={pending || fetching} onClick={() => void run([item.key], ReleaseReviewAction.Download)}>{t(item.status === ReleaseReviewStatus.Failed ? "releaseReview.retry" : "releaseReview.download")}</button><button type="button" className="ghost-button" disabled={pending || fetching} onClick={() => void run([item.key], ReleaseReviewAction.Reject)}>{t("releaseReview.reject")}</button></> : null}{reconsiderable(item.status) ? <button type="button" className="ghost-button" disabled={pending || fetching} onClick={() => void run([item.key], ReleaseReviewAction.Reconsider)}>{t("releaseReview.reconsider")}</button> : null}{item.taskId ? <Link className="ghost-button release-review-link" to={`/tasks/${encodeURIComponent(item.taskId)}`}>{t("releaseReview.openTask")}</Link> : null}</div>
      </article>)}
    </div>
    <div className="release-review__pagination"><button className="ghost-button" disabled={fetching || (connection?.page ?? 1) <= 1} onClick={() => change({ page: String((connection?.page ?? 1) - 1) })}>{t("releaseReview.prev")}</button><span>{t("releaseReview.page", { page: connection?.page ?? page, pages: Math.max(1, connection?.totalPages ?? 1), count: connection?.totalCount ?? 0 })}</span><button className="ghost-button" disabled={fetching || (connection?.page ?? 1) >= (connection?.totalPages ?? 0)} onClick={() => change({ page: String((connection?.page ?? 1) + 1) })}>{t("releaseReview.next")}</button></div>
  </section>;
}
