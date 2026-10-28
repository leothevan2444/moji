import {
  LogLevel,
  SubscriptionReleaseBehavior,
  SubscriptionReleaseDateRange,
  TorrentFileMatchEffect,
  TitleMatchEffect,
  TitleMatchPatternMode,
  TorrentSelectionDirection,
  TorrentSelectionRuleType,
} from "../graphql/generated/graphql";

function makeDefaultTorrentSelectionRule(type: TorrentSelectionRuleType) {
  return {
    type,
    enabled: true,
    indexerPreference: {
      trackerIds: [] as string[]
    },
    titleMatch: {
      clauses: [] as Array<{
        pattern: string;
        patternMode: TitleMatchPatternMode;
        effect: TitleMatchEffect;
      }>
    },
    publishDate: {
      direction: TorrentSelectionDirection.Desc
    },
    seeders: {
      direction: TorrentSelectionDirection.Desc
    },
    size: {
      direction: TorrentSelectionDirection.Desc
    },
    torrentFileNameMatch: {
      clauses: [] as Array<{
        pattern: string;
        patternMode: TitleMatchPatternMode;
        effect: TorrentFileMatchEffect;
      }>
    }
  };
}

export const DEFAULT_TORRENT_SELECTION_RULES = [
  makeDefaultTorrentSelectionRule(TorrentSelectionRuleType.IndexerPreference),
  makeDefaultTorrentSelectionRule(TorrentSelectionRuleType.TitleMatch),
  makeDefaultTorrentSelectionRule(TorrentSelectionRuleType.PublishDate),
  makeDefaultTorrentSelectionRule(TorrentSelectionRuleType.TitleSimilarity),
  makeDefaultTorrentSelectionRule(TorrentSelectionRuleType.Seeders),
  makeDefaultTorrentSelectionRule(TorrentSelectionRuleType.Size)
];

export const DEFAULT_TORRENT_FILE_INSPECTION_RULES = [
  makeDefaultTorrentSelectionRule(TorrentSelectionRuleType.TorrentSingleVideo),
  makeDefaultTorrentSelectionRule(TorrentSelectionRuleType.TorrentFileNameMatch)
];

export const EMPTY_POLICY_SETTINGS_FORM = {
  taskProcessingEnabled: true,
  sourcingRetryIntervalHours: "6",
  taskProgressSyncIntervalSeconds: "60",
  subscriptionPollIntervalHours: "1",
  stashBoxEndpoints: [] as string[],
  subscriptionReleasePolicy: {
    soloBehavior: SubscriptionReleaseBehavior.Download,
    groupBehavior: SubscriptionReleaseBehavior.Review,
    compilationBehavior: SubscriptionReleaseBehavior.Block,
    vrBehavior: SubscriptionReleaseBehavior.Review,
    maxGroupPerformerCount: "3",
    releaseDateRange: SubscriptionReleaseDateRange.All
  },
  torrentSelection: {
    enabled: true,
    inspectionCandidateLimit: "5",
    fastRules: DEFAULT_TORRENT_SELECTION_RULES.map((rule) => structuredClone(rule)),
    torrentRules: DEFAULT_TORRENT_FILE_INSPECTION_RULES.map((rule) => structuredClone(rule))
  }
};

export const SUBSCRIPTION_PAGE_SIZE_OPTIONS = [12, 24, 48, 96] as const;
export const LOG_LEVEL_OPTIONS: LogLevel[] = [LogLevel.Debug, LogLevel.Info, LogLevel.Warning, LogLevel.Error];
