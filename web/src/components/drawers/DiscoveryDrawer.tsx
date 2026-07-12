import { formatBytes, formatDurationSeconds, formatPublishDate } from "../../utils";
import { SkeletonCardList } from "../common/Skeleton";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faArrowUpRightFromSquare,
  faDownload,
  faMagnet
} from "@fortawesome/free-solid-svg-icons";
import type {
  DiscoverScenesDocumentQuery,
  SearchDocumentQuery
} from "../../graphql/generated/graphql";
import type { DiscoveryMode } from "../../constants";
import { useTranslation } from "react-i18next";

type DiscoverResult = DiscoverScenesDocumentQuery["discoverScenes"]["items"][number];
type DiscoverConnection = DiscoverScenesDocumentQuery["discoverScenes"];
type JackettResult = SearchDocumentQuery["jackettSearch"][number];

interface DiscoveryDrawerProps {
  mode: DiscoveryMode;
  query: string;
  searching: boolean;
  error: Error | null;
  pendingAddId: string | null;
  discoverResult: DiscoverConnection | null;
  discoverItems: DiscoverResult[];
  jackettItems: JackettResult[];
  hasAnyResults: boolean;
  usedStashBoxName?: string | null;
  onQueueDiscovered: (result: DiscoverResult) => void;
  onAddJackett: (result: JackettResult) => void;
  onTryJackett: () => void;
  onClearTrackers: () => void;
  hasActiveTrackers: boolean;
}

export function DiscoveryDrawer({
  mode,
  query,
  searching,
  error,
  pendingAddId,
  discoverItems,
  jackettItems,
  hasAnyResults,
  usedStashBoxName,
  onQueueDiscovered,
  onAddJackett,
  onTryJackett,
  onClearTrackers,
  hasActiveTrackers
}: DiscoveryDrawerProps) {
  const { t } = useTranslation();
  const isStashBox = mode === "stashbox";

  // 首屏 loading 渲染骨架；翻页过程中 searching=true 但已有数据时不渲染骨架，
  // 避免列表抖动。
  const showSkeleton = searching && !hasAnyResults;
  const showEmpty = !searching && !hasAnyResults;

  return (
    <>
      {error ? <p className="inline-error">{error.message}</p> : null}

      {showSkeleton ? (
        <SkeletonCardList count={6} />
      ) : showEmpty ? (
        <EmptyState
          isStashBox={isStashBox}
          hasQuery={query.trim() !== ""}
          hasActiveTrackers={hasActiveTrackers}
          onTryJackett={onTryJackett}
          onClearTrackers={onClearTrackers}
        />
      ) : isStashBox ? (
        <div className="discovery-results">
          {discoverItems.map((result) => (
            <article key={result.key} className="candidate-card candidate-card--stashbox">
              {result.imageUrl ? (
                <div className="candidate-card__poster candidate-card__poster--discovery">
                  <img src={result.imageUrl} alt={result.title} loading="lazy" onError={(event) => { event.currentTarget.style.display = "none"; }} />
                </div>
              ) : null}
              <div className="candidate-card__content">
                <div className="candidate-card__head">
                  <div>
                    <div className="candidate-card__title-row">
                      <h3>
                        <span className="candidate-card__title-text">{result.title}</span>
                        {result.durationSeconds ? (
                          <span className="candidate-card__meta-chip">
                            {formatDurationSeconds(result.durationSeconds)}
                          </span>
                        ) : null}
                      </h3>
                      {usedStashBoxName ? (
                        <span className="status-chip tone-info" title={t("discoverUi.result.source")}>
                          {usedStashBoxName}
                        </span>
                      ) : null}
                    </div>
                    <div className="candidate-card__meta">
                      {result.code ? (
                        <span className="candidate-card__meta-line" title={result.code}>
                          {t("discoverUi.result.code", { code: result.code })}
                        </span>
                      ) : null}
                      {result.studioName ? (
                        <span className="candidate-card__meta-line" title={result.studioName}>
                          {t("discoverUi.result.studio", { studio: result.studioName })}
                        </span>
                      ) : null}
                      {result.performerNames.length > 0 ? (
                        <span className="candidate-card__meta-line" title={result.performerNames.join(" / ")}>
                          {t("discoverUi.result.performers", { performers: result.performerNames.slice(0, 3).join(" / ") })}
                          {result.performerNames.length > 3 ? " …" : ""}
                        </span>
                      ) : null}
                    </div>
                  </div>
                </div>
                <div className="candidate-card__foot">
                  <span>{result.date || t("discoverUi.result.noDate")}</span>
                  <div className="inline-actions">
                    {result.url ? (
                      <a
                        href={result.url}
                        target="_blank"
                        rel="noreferrer"
                        className="icon-button"
                        title={t("discoverUi.result.original")}
                        aria-label={t("discoverUi.result.original")}
                      >
                        <FontAwesomeIcon icon={faArrowUpRightFromSquare} />
                      </a>
                    ) : null}
                    <button
                      type="button"
                      onClick={() => onQueueDiscovered(result)}
                      disabled={pendingAddId === result.key}
                    >
                      {pendingAddId === result.key ? t("discoverUi.result.queueing") : t("discoverUi.result.queue")}
                    </button>
                  </div>
                </div>
              </div>
            </article>
          ))}
        </div>
      ) : (
        <div className="discovery-results">
          {jackettItems.map((result) => (
            <article key={`${result.tracker}-${result.link}`} className="candidate-card candidate-card--jackett">
              <div className="candidate-card__content">
                <div className="candidate-card__head">
                  <div>
                    <h3>{result.title}</h3>
                    <p>
                      {result.tracker} · {formatBytes(Number(result.size) || 0)} · {result.seeders} seeders
                    </p>
                  </div>
                  <span className="status-chip tone-warn">Jackett</span>
                </div>
                <div className="candidate-card__foot">
                  <span>{formatPublishDate(result.publishDate)}</span>
                  <div className="inline-actions">
                    {result.magnetUri ? (
                      <a
                        href={result.magnetUri}
                        className="icon-button"
                        title={t("discoverUi.result.magnet")}
                        aria-label={t("discoverUi.result.magnet")}
                      >
                        <FontAwesomeIcon icon={faMagnet} />
                      </a>
                    ) : null}
                    <a
                      href={result.link}
                      target="_blank"
                      rel="noreferrer"
                      className="icon-button"
                      title={t("discoverUi.result.originalDownload")}
                      aria-label={t("discoverUi.result.originalDownload")}
                    >
                      <FontAwesomeIcon icon={faDownload} />
                    </a>
                    <button
                      type="button"
                      onClick={() => onAddJackett(result)}
                      disabled={pendingAddId === result.link}
                    >
                      {pendingAddId === result.link ? t("discoverUi.result.adding") : t("discoverUi.result.create")}
                    </button>
                  </div>
                </div>
              </div>
            </article>
          ))}
        </div>
      )}
    </>
  );
}

interface EmptyStateProps {
  isStashBox: boolean;
  hasQuery: boolean;
  hasActiveTrackers: boolean;
  onTryJackett: () => void;
  onClearTrackers: () => void;
}

function EmptyState({
  isStashBox,
  hasQuery,
  hasActiveTrackers,
  onTryJackett,
  onClearTrackers
}: EmptyStateProps) {
  const { t } = useTranslation();
  if (!hasQuery) {
    return (
      <div className="empty-card empty-card--wide">
        <h3>{t("discoverUi.result.enter")}</h3>
        <p>{t("discoverUi.result.enterDetail")}</p>
      </div>
    );
  }

  if (isStashBox) {
    return (
      <div className="empty-card empty-card--wide">
        <h3>{t("discoverUi.result.noStash")}</h3>
        <p>{t("discoverUi.result.tryJackettDetail")}</p>
        <button type="button" className="ghost-button" onClick={onTryJackett} style={{ marginTop: 12 }}>
          {t("discoverUi.result.tryJackett")}
        </button>
      </div>
    );
  }

  if (hasActiveTrackers) {
    return (
      <div className="empty-card empty-card--wide">
        <h3>{t("discoverUi.result.noFiltered")}</h3>
        <p>{t("discoverUi.result.noFilteredDetail")}</p>
        <button type="button" className="ghost-button" onClick={onClearTrackers} style={{ marginTop: 12 }}>
          {t("discoverUi.filters.clear")}
        </button>
      </div>
    );
  }

  return (
    <div className="empty-card empty-card--wide">
      <h3>{t("discoverUi.result.noResults")}</h3>
      <p>{t("discoverUi.result.noJackett")}</p>
    </div>
  );
}
