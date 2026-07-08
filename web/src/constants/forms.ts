import {
  LogLevel,
  SubscriptionReleaseBehavior,
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

export const EMPTY_STASH_FORM = {
  url: "",
  apiKey: ""
};

export const EMPTY_INGEST_FORM = {
  deliveryMode: "PATH_MAP",
  downloads: {
    qbRoot: "",
    mojiRoot: ""
  },
  library: {
    mojiRoot: "",
    stashRoot: ""
  },
  transfer: {
    action: ""
  }
};

export const EMPTY_JACKETT_FORM = {
  url: "",
  apiKey: "",
  password: ""
};

export const EMPTY_QBITTORRENT_FORM = {
  url: "",
  username: "",
  password: "",
  defaultSavePath: "",
  category: "",
  tags: ""
};

export const EMPTY_AUTOMATION_FORM = {
  taskProgressSyncIntervalSeconds: "60",
  subscriptionPollIntervalHours: "1",
  stashBoxEndpoints: [] as string[],
  subscriptionReleasePolicy: {
    soloBehavior: SubscriptionReleaseBehavior.Download,
    groupBehavior: SubscriptionReleaseBehavior.Review,
    compilationBehavior: SubscriptionReleaseBehavior.Block,
    maxGroupPerformerCount: "3"
  },
  torrentSelection: {
    enabled: true,
    inspectionCandidateLimit: "5",
    fastRules: DEFAULT_TORRENT_SELECTION_RULES.map((rule) => structuredClone(rule)),
    torrentRules: DEFAULT_TORRENT_FILE_INSPECTION_RULES.map((rule) => structuredClone(rule))
  }
};

export const EMPTY_SYSTEM_FORM = {
  taskDeletePolicy: "KEEP_ONLY"
};

export const SUBSCRIPTION_PAGE_SIZE_OPTIONS = [12, 24, 48, 96] as const;
export const LOG_LEVEL_OPTIONS: LogLevel[] = [LogLevel.Debug, LogLevel.Info, LogLevel.Warning, LogLevel.Error];
