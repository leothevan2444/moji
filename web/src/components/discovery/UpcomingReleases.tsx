import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { useMutation, useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { UpcomingReleasesDocument, RefreshUpcomingReleaseDocument } from "../../graphql/generated/graphql";
import { usePerformerSubscriptionEvents } from "../../hooks/usePerformerSubscriptionEvents";
import { UpcomingReleaseCard, type UpcomingRelease } from "./UpcomingReleaseCard";
import { UpcomingReleaseDetail } from "./UpcomingReleaseDetail";
import { describeQueryError } from "../../services/queryError";
import "../../styles/upcoming.scss";

type Release = UpcomingRelease;
export function safeUpcomingSource(raw: string) {
  try { const url = new URL(raw); return ["https:", "http:"].includes(url.protocol) && !url.username && !url.password ? url.href : null; } catch { return null; }
}
export function upcomingDays(date: string, timezone: string, now = new Date(), serverToday?: string) {
  const parts = new Intl.DateTimeFormat("en-CA", { timeZone: serverToday ? "UTC" : timezone, year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(now);
  const part = (type: string) => parts.find(p => p.type === type)?.value;
  const today = serverToday ? Date.parse(`${serverToday}T00:00:00Z`) : Date.UTC(Number(part("year")), Number(part("month")) - 1, Number(part("day")));
  const parsed = Date.parse(`${date}T00:00:00Z`);
  return Number.isFinite(parsed) ? Math.round((parsed - today) / 86400000) : null;
}
export function UpcomingReleases() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  const expanded = params.get("section") === "upcoming";
  const page = Math.max(1, Number(params.get("upcomingPage")) || 1);
  const search = params.get("upcomingSearch") ?? "";
  const performerId = params.get("upcomingPerformer") ?? "";
  const daysValue = Number(params.get("upcomingDays"));
  const days = [7, 30].includes(daysValue) ? daysValue : undefined;
  const [result, execute] = useQuery({ query: UpcomingReleasesDocument, variables: { input: { page: expanded ? page : 1, pageSize: expanded ? 24 : 8, search: expanded ? search : undefined, performerId: expanded ? performerId : undefined, days: expanded ? days : undefined } }, requestPolicy: "cache-and-network" });
  const refresh = () => execute({ requestPolicy: "network-only" });
  const { event } = usePerformerSubscriptionEvents({ enabled: true, onRefresh: refresh });
  useEffect(() => { if (!event) return; const timer = window.setTimeout(refresh, 200); return () => window.clearTimeout(timer); }, [event]); // Domain events invalidate only this mounted page.
  useEffect(() => { const visible = () => { if (document.visibilityState === "visible") refresh(); }; window.addEventListener("focus", visible); document.addEventListener("visibilitychange", visible); return () => {window.removeEventListener("focus", visible); document.removeEventListener("visibilitychange", visible);}; }, []);
  const [mutation, update] = useMutation(RefreshUpcomingReleaseDocument);
  const [selected, setSelected] = useState<Release | null>(null);
  const [feedback, setFeedback] = useState<{ key: string; text: string } | null>(null);
  const data = result.data?.upcomingReleases;
  const patch = (key: string, value: string) => setParams(previous => { const next = new URLSearchParams(previous); if (value) next.set(key, value); else next.delete(key); if (key !== "upcomingPage") next.delete("upcomingPage"); return next; });
  const status = (item: Release) => {
    if (item.state === "HANDED_OFF") return t("upcoming.handedOff");
    const count = upcomingDays(item.date, data?.timezone ?? "UTC", undefined, data?.today);
    if (item.errorCode || item.state === "UNCONFIRMED" || count === null || count <= 0) return t("upcoming.unconfirmed");
    return null;
  };
  const check = async () => {
    if (!selected) return;
    setFeedback(null);
    const key = selected.key;
    const response = await update({ key });
    if (response.error) setFeedback({ key, text: describeQueryError(response.error) });
    else if (response.data) {
      const release = response.data.refreshUpcomingRelease;
      setSelected(current => current?.key === release.key ? release : current);
      setFeedback({ key, text: t("upcoming.updated") });
      refresh();
    }
  };
  return <section className="section-band" id="upcoming" aria-labelledby="upcoming-title">
    <div className="band-head"><div><h2 id="upcoming-title">{t("upcoming.title")}{data ? ` · ${data.totalCount}` : ""}</h2></div>
      <button className="ghost-button" onClick={() => patch("section", expanded ? "" : "upcoming")}>{t(expanded ? "upcoming.collapse" : "upcoming.viewAll")}</button>
    </div>
    {data && !data.automatic && data.subscribedCount > 0 ? <p className="upcoming-notice">{t("upcoming.paused")} <Link to="/settings/tasks">{t("upcoming.settings")}</Link></p> : null}
    {expanded ? <form className="upcoming-filters" onSubmit={e => e.preventDefault()}>
      <label>{t("upcoming.search")}<input value={search} onChange={e => patch("upcomingSearch", e.target.value)} /></label>
      <label>{t("upcoming.performer")}<select value={performerId} onChange={e => patch("upcomingPerformer", e.target.value)}><option value="">{t("upcoming.allPerformers")}</option>{data?.performers.map(a => <option value={a.id} key={a.id}>{a.name}</option>)}</select></label>
      <label>{t("upcoming.range")}<select value={days ?? ""} onChange={e => patch("upcomingDays", e.target.value)}><option value="">{t("upcoming.allDates")}</option><option value="7">{t("upcoming.week")}</option><option value="30">{t("upcoming.month")}</option></select></label>
    </form> : null}
    {result.error ? <p role="alert">{t("upcoming.loadFailed")} <button className="ghost-button" onClick={refresh}>{t("upcoming.retry")}</button></p> : null}
    {!data && result.fetching ? <p role="status">{t("upcoming.loading")}</p> : null}
    {data?.items.length === 0 ? <div className="preview-panel"><p>{t(data.subscribedCount === 0 ? "upcoming.noSubscriptions" : search || performerId || days ? "upcoming.noMatch" : "upcoming.empty")}</p><Link to="/performers">{t("upcoming.performers")}</Link></div> : null}
    <div className="card-grid upcoming-grid" aria-busy={result.fetching}>
      {data?.items.map(item => <UpcomingReleaseCard key={item.key} release={item} status={status(item)} onOpen={() => { setSelected(item); setFeedback(null); }} />)}
    </div>
    {expanded && data && data.totalPages > 1 ? <nav className="upcoming-pagination" aria-label={t("upcoming.pagination")}><button className="ghost-button" disabled={page <= 1} onClick={() => patch("upcomingPage", String(page - 1))}>{t("upcoming.previous")}</button><span>{page} / {data.totalPages}</span><button className="ghost-button" disabled={page >= data.totalPages} onClick={() => patch("upcomingPage", String(page + 1))}>{t("upcoming.next")}</button></nav> : null}
    {selected ? <UpcomingReleaseDetail release={selected} status={status(selected)} sourceUrl={safeUpcomingSource(selected.sourceUrl)} refreshing={mutation.fetching} feedback={feedback?.key === selected.key ? feedback.text : ""} onRefresh={() => void check()} onClose={() => setSelected(null)} /> : null}
  </section>;
}
