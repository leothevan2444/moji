import { useEffect, useRef, useState } from "react";
import { useMutation } from "urql";
import { useTranslation } from "react-i18next";
import { PrepareTorrentDownloadDocument, PreparedTorrentDownloadKind } from "../graphql/generated/graphql";

type Ready = { scope: string; url: string; magnet?: string; expires: number };
export function usePreparedTorrentDownload(searchId: string, candidateId: string) {
 const { t } = useTranslation();
 const scope = JSON.stringify([searchId, candidateId]);
 const active = useRef(scope); active.current = scope;
 const request = useRef<AbortController | null>(null);
 const [state, setState] = useState<{scope: string; busy: boolean; ready?: Ready; error?: unknown}>({scope, busy:false});
 const [, execute] = useMutation(PrepareTorrentDownloadDocument);
 useEffect(() => () => { request.current?.abort(); request.current = null; }, [scope]);
 const ready = state.scope === scope ? state.ready : undefined;
 useEffect(() => {
  if (!ready) return;
  const timer = setTimeout(() => setState(previous => previous.ready === ready ? {scope, busy:false} : previous), Math.max(0, ready.expires - Date.now()));
  return () => clearTimeout(timer);
 }, [ready, scope]);
 const prepare = async () => {
  if (request.current || !searchId || !candidateId) return;
  const controller = new AbortController(); request.current = controller;
  const timeout = setTimeout(() => controller.abort(), 120000);
  setState({scope, busy:true});
  try {
   const result = await execute({input:{searchId, candidateId}}, {fetchOptions:{signal:controller.signal}});
   if (result.error) throw result.error;
   const value = result.data?.prepareTorrentDownload;
   if (!value || !/^\/api\/indexers\/downloads\/[a-f0-9]{64}$/.test(value.downloadUrl)) throw new Error(t("preparedDownload.failed"));
   const expires = Math.min(Date.now() + 300000, Date.parse(value.expiresAt));
   if (!Number.isFinite(expires) || expires <= Date.now()) throw new Error(t("indexerErrors.INDEXER_RESOURCE_EXPIRED"));
   let magnet: string | undefined;
   if (value.kind === PreparedTorrentDownloadKind.Magnet) {
    // Read only this application's prepared attachment, never a site locator.
    const response = await fetch(value.downloadUrl, {signal:controller.signal, cache:"no-store", credentials:"same-origin", redirect:"error"});
    if (!response.ok || !response.headers.get("content-type")?.startsWith("text/uri-list")) throw new Error(t("preparedDownload.failed"));
    magnet = await response.text();
    if (magnet.length > 65536 || /[\r\n\0]/.test(magnet)) throw new Error(t("preparedDownload.failed"));
    try {
     const parsed = new URL(magnet);
     if (parsed.protocol !== "magnet:" || parsed.host || parsed.hash || !parsed.searchParams.get("xt")) throw new Error();
    } catch { throw new Error(t("preparedDownload.failed")); }
   }
   if (controller.signal.aborted) throw new Error(t("preparedDownload.failed"));
   if (active.current === scope) setState({scope, busy:false, ready:{scope, url:value.downloadUrl, magnet, expires}});
  } catch(error) {
   if (!controller.signal.aborted && active.current === scope) setState({scope, busy:false, error});
   else if (active.current === scope && request.current === controller) setState({scope, busy:false, error:new Error(t("preparedDownload.failed"))});
  } finally { clearTimeout(timeout); if (request.current === controller) request.current = null; }
 };
 return {prepare, ready, busy:state.scope === scope && state.busy, error:state.scope === scope ? state.error : undefined};
}
