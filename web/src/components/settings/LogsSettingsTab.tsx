import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useOutletContext } from "react-router";
import { useQuery } from "urql";
import type { AppOutletContext } from "../../app/AppLayout";
import { LogLevel, LogsDocumentDocument } from "../../graphql/generated/graphql";
import { mergeLogEntries, type StreamedLogEntry } from "../../hooks/useLogEvents";
import { describeQueryError } from "../../services/queryError";
import { formatLogEntries } from "../../utils";
import { LogViewer } from "./LogViewer";
import { LogEventStream } from "./LogEventStream";
import { SettingsCard } from "./SettingsTabCommon";
import "../../styles/settings-logs.scss";

const rank: Record<LogLevel, number> = { [LogLevel.Debug]: 0, [LogLevel.Info]: 1, [LogLevel.Warning]: 2, [LogLevel.Error]: 3 };

export default function LogsSettingsTab() {
  const { t } = useTranslation();
  const { pushToast, copyText } = useOutletContext<AppOutletContext>();
  const [level, setLevel] = useState(LogLevel.Info);
  const [streamed, setStreamed] = useState<StreamedLogEntry[]>([]);
  const [downloading, setDownloading] = useState(false);
  const [{ data, fetching, error }, refresh] = useQuery({ query: LogsDocumentDocument, variables: { minLevel: level }, requestPolicy: "network-only" });
  const logs = useMemo(() => mergeLogEntries(data?.logs ?? [], streamed.filter((entry) => rank[entry.level] >= rank[level])), [data, level, streamed]);
  const download = async () => {
    setDownloading(true);
    try {
      const response = await fetch("/api/logs/current");
      if (!response.ok) throw new Error(t("logsUi.downloadHttpError", { status: response.status }));
      const url = URL.createObjectURL(await response.blob());
      const link = document.createElement("a"); link.href = url; link.download = "moji.log"; link.click(); URL.revokeObjectURL(url);
    } catch (cause) { pushToast("tone-danger", cause instanceof Error ? cause.message : t("logsUi.downloadFailed")); }
    finally { setDownloading(false); }
  };

  return <SettingsCard title={t("settings.tabs.logs")}>
    <LogEventStream
      pause={false}
      onEntries={(entries) => setStreamed((current) => mergeLogEntries([], [...entries, ...current]))}
      onResync={() => {
        setStreamed([]);
        void refresh({ requestPolicy: "network-only" });
      }}
    />
    <div className="toolbar-inline toolbar-inline--logs">
      <select aria-label={t("logsUi.level")} value={level} onChange={(event) => {
        setLevel(event.target.value as LogLevel);
        setStreamed([]);
      }}>
        <option value={LogLevel.Debug}>DEBUG</option>
        <option value={LogLevel.Info}>INFO</option>
        <option value={LogLevel.Warning}>WARNING</option>
        <option value={LogLevel.Error}>ERROR</option>
      </select>
      <button type="button" className="ghost-button" onClick={() => void refresh({ requestPolicy: "network-only" })}>
        {fetching ? t("logsUi.refreshing") : t("logsUi.refresh")}
      </button>
      <button type="button" className="ghost-button" disabled={downloading} onClick={() => void download()}>
        {downloading ? t("logsUi.downloading") : t("logsUi.download")}
      </button>
    </div>
    {error ? <p className="settings-feedback tone-danger">{describeQueryError(error)}</p> : null}
    <LogViewer key={level} entries={logs} loading={fetching} onCopy={(entries) => void copyText(formatLogEntries(entries), t("logsUi.copy"))} />
  </SettingsCard>;
}
