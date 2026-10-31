import { useLayoutEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { LogLevel } from "../../graphql/generated/graphql";
import type { LogEntry } from "../../hooks/useLogEvents";
import { formatDateTime } from "../../utils";

interface Props {
  entries: LogEntry[];
  loading?: boolean;
  onCopy: (entries: LogEntry[]) => void;
}

export function LogViewer({ entries, onCopy, loading = false }: Props) {
  const { t } = useTranslation();
  const viewport = useRef<HTMLDivElement>(null);
  // Keep the actual rows stable while reading, even when the live cache evicts old rows.
  const [history, setHistory] = useState<LogEntry[] | null>(null);
  const visible = useMemo(() => [...(history ?? entries)].reverse(), [history, entries]);
  const following = history === null;
  const hasNew = history !== null && entries.some(entry => entry.sequence > (history[0]?.sequence ?? 0));

  useLayoutEffect(() => {
    if (following && viewport.current) viewport.current.scrollTop = viewport.current.scrollHeight;
  }, [visible, following]);

  return <>
    <div className="log-viewer-status">
      <span>{t("logsUi.loaded", { count: visible.length })} · {t(following ? "logsUi.following" : "logsUi.readingHistory")}</span>
      <div className="log-viewer-actions">
        <button type="button" className="ghost-button" onClick={() => onCopy(visible)}>{t("logsUi.copy")}</button>
        <button type="button" className="ghost-button" onClick={() => setHistory(following ? entries : null)}>
          {t(following ? "logsUi.pauseFollow" : hasNew ? "logsUi.resumeNew" : "logsUi.resumeFollow")}
        </button>
      </div>
    </div>
    <div ref={viewport} className="log-stream" role="log" aria-label={t("settings.tabs.logs")} aria-busy={loading} aria-live={following ? "polite" : "off"} tabIndex={0}
      onScroll={() => {
        const node = viewport.current;
        if (following && node && node.scrollHeight - node.clientHeight - node.scrollTop > 24) setHistory(entries);
      }}>
      {visible.length === 0 ? <p className="muted">{t(loading ? "logsUi.refreshing" : "logsUi.empty")}</p> : visible.map((entry) => <div key={entry.sequence} className={`log-line ${entry.level === LogLevel.Error ? "log-line--error" : entry.level === LogLevel.Warning ? "log-line--warn" : entry.level === LogLevel.Debug ? "log-line--debug" : "log-line--info"}`}>
        <time className="log-line__time" dateTime={entry.time}>{formatDateTime(entry.time)}</time>
        <span className="log-line__level">[{entry.level}]</span>
        <span className="log-line__message">{entry.message}</span>
      </div>)}
    </div>
    <p className="settings-hint">{t("logsUi.scope")}</p>
  </>;
}
