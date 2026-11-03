import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useIndexerCache } from "../../hooks/useIndexerCache";
import { useIndexerManagement } from "../../hooks/useIndexerManagement";
import { IndexerSourceKind } from "../../graphql/generated/graphql";
import { formatBytes } from "../../utils";
import { describeQueryError } from "../../services/queryError";
import { SettingsCard, SettingsError, SettingsLoading } from "./SettingsTabCommon";
import { Drawer } from "../layout/Drawer";
import { IndexerCacheSettings } from "./IndexerCacheSettings";

export function IndexerCachePanel() {
  const { t } = useTranslation();
  const cache = useIndexerCache(); const management = useIndexerManagement();
  const [scope, setScope] = useState("");
  const [confirming, setConfirming] = useState<string | null>(null);
  if (!cache.snapshot) return cache.error ? <SettingsError title={t("indexerCache.title")} error={cache.error} onRetry={cache.refresh} /> : <SettingsLoading title={t("indexerCache.title")} />;
  const status = cache.snapshot;
  const metrics = [
    [t("indexerCache.entries"), status.entries], [t("indexerCache.usage"), `${formatBytes(status.usedBytes)} / ${formatBytes(status.maxSizeMb * 1048576)}`],
    [t("indexerCache.activeRequests"), status.activeRequests], [t("indexerCache.waiters"), status.activeWaiters]
  ];
  return <SettingsCard title={t("indexerCache.title")}>
    <IndexerCacheSettings snapshot={status} onSaved={cache.refresh} />
    {cache.error ? <p className="settings-feedback tone-danger" role="alert">{describeQueryError(cache.error)} <button type="button" onClick={cache.refresh}>{t("common.retry")}</button></p> : null}
    <dl className="indexer-cache-metrics">{metrics.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>
    <div className="indexer-actions indexer-cache-actions"><div className="settings-field"><label htmlFor="indexer-cache-scope">{t("indexerCache.scope")}</label><select id="indexer-cache-scope" value={scope} disabled={cache.clearing} onChange={event => setScope(event.target.value)}><option value="">{t("indexerCache.all")}</option>{management.sites.filter(site => site.kind === IndexerSourceKind.Builtin).map(site => <option key={site.id} value={site.id}>{site.name}</option>)}</select></div><button type="button" disabled={cache.clearing} onClick={() => setConfirming(scope)}>{t("indexerCache.clear")}</button><button type="button" disabled={cache.fetching} onClick={cache.refresh}>{t("common.refresh")}</button></div>
    {confirming !== null ? <Drawer visibleDrawer="confirm" closing={false} title={t("indexerCache.clear")} onClose={() => { if (!cache.clearing) setConfirming(null); }} closeDisabled={cache.clearing}>
      <p>{t("indexerCache.effect")}</p><p>{confirming ? management.sites.find(site => site.id === confirming)?.name : t("indexerCache.all")}</p>
      {cache.error ? <p role="alert">{describeQueryError(cache.error)}</p> : null}
      <div className="indexer-actions"><button type="button" disabled={cache.clearing} onClick={() => void cache.clear(confirming).then(ok => { if (ok) setConfirming(null); })}>{t("indexerCache.clear")}</button><button type="button" disabled={cache.clearing} onClick={() => setConfirming(null)}>{t("common.close")}</button></div>
    </Drawer> : null}
  </SettingsCard>;
}
