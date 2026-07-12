import { FormEvent, useMemo, useRef } from "react";
import { SegmentedTab } from "../components/common/SegmentedTab";
import { SearchHistoryDropdown } from "../components/common/SearchHistoryDropdown";
import { useGlobalSlashShortcut, useRotatingPlaceholder } from "../hooks";
import {
  DISCOVERY_MODE_OPTIONS,
  SEARCH_PLACEHOLDERS,
  type DiscoveryMode
} from "../constants";
import { useTranslation } from "react-i18next";

interface DiscoveryPageProps {
  query: string;
  searching: boolean;
  inputFocused: boolean;
  mode: DiscoveryMode;
  history: string[];
  historyVisible: boolean;
  onQueryChange: (value: string) => void;
  onInputFocus: () => void;
  onInputBlur: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onModeChange: (mode: DiscoveryMode) => void;
  onPickHistory: (entry: string) => void;
  onRemoveHistory: (entry: string) => void;
  onClearHistory: () => void;
  onDismissHistory: () => void;
  onOpenHelp: () => void;
}

export function DiscoveryPage({
  query,
  searching,
  inputFocused,
  mode,
  history,
  historyVisible,
  onQueryChange,
  onInputFocus,
  onInputBlur,
  onSubmit,
  onModeChange,
  onPickHistory,
  onRemoveHistory,
  onClearHistory,
  onDismissHistory,
  onOpenHelp
}: DiscoveryPageProps) {
  const { t } = useTranslation();
  const inputRef = useRef<HTMLInputElement>(null);
  const placeholders = useMemo(() => SEARCH_PLACEHOLDERS.map((key) => t(key)), [t]);
  const rawPlaceholder = useRotatingPlaceholder(placeholders, 4500, inputFocused);
  useGlobalSlashShortcut(inputRef);

  // placeholder 反映当前 mode，但 tab 已经在左侧 prefix 显示，故不再重复
  const placeholder = rawPlaceholder;
  const submitTitle = t(mode === "stashbox" ? "discoverUi.searchStashBox" : "discoverUi.searchJackett");

  const canSubmit = !searching && query.trim() !== "";

  return (
    <>
      <section className="section-band">
        <div className="band-head">
          <div>
            <h2>{t("discoverUi.search")}</h2>
          </div>
        </div>

        <form className="discovery-shell" onSubmit={onSubmit}>
          <div className="discovery-shell__prefix">
            <SegmentedTab
              value={mode}
              options={DISCOVERY_MODE_OPTIONS}
              onChange={onModeChange}
              size="sm"
              ariaLabel={t("discoverUi.searchMode")}
            />
          </div>
          <div className="discovery-shell__field">
            <input
              ref={inputRef}
              value={query}
              onChange={(event) => onQueryChange(event.target.value)}
              onFocus={onInputFocus}
              onBlur={onInputBlur}
              placeholder={placeholder}
              aria-label={t("discoverUi.keyword")}
              spellCheck={false}
              autoComplete="off"
            />
            <SearchHistoryDropdown
              history={history}
              visible={historyVisible}
              onPick={onPickHistory}
              onRemove={onRemoveHistory}
              onClear={onClearHistory}
              onDismiss={onDismissHistory}
            />
          </div>
          <button
            type="submit"
            className="discovery-shell__submit"
            disabled={!canSubmit}
            aria-label={submitTitle}
            title={submitTitle}
          >
            {searching ? (
              <span className="discovery-shell__submit-spinner" aria-hidden="true" />
            ) : (
              <svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
                <path
                  d="M2 8h11M9 4l4 4-4 4"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.8"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            )}
            <span className="discovery-shell__submit-label">{submitTitle}</span>
          </button>
        </form>
      </section>

      <section className="section-band section-band--preview">
        <div className="band-head">
          <div>
            <h2>{t("discoverUi.recommendation")}</h2>
          </div>
          <p className="band-note">{t("discoverUi.recommendationNote")}</p>
        </div>
        <div className="preview-panel">
          <div>
            <h3>{t("discoverUi.recommendationDisabled")}</h3>
            <p>{t("discoverUi.recommendationDetail")}</p>
          </div>
          <button type="button" className="ghost-button" onClick={onOpenHelp}>
            {t("discoverUi.viewHelp")}
          </button>
        </div>
      </section>
    </>
  );
}
