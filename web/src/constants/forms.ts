import {
  LogLevel,
  TitleMatchEffect,
  TitleMatchPatternMode,
  TorrentSelectionDirection,
  TorrentSelectionRuleType,
} from "../graphql/generated/graphql";

export const DEFAULT_TORRENT_SELECTION_RULES = [
  {
    id: "default-seeders",
    type: TorrentSelectionRuleType.Seeders,
    enabled: true,
    direction: TorrentSelectionDirection.Desc,
    indexerPreference: {
      trackerIds: [] as string[]
    },
    titleMatch: {
      clauses: [] as Array<{
        pattern: string;
        patternMode: TitleMatchPatternMode;
        effect: TitleMatchEffect;
      }>
    }
  },
  {
    id: "default-size",
    type: TorrentSelectionRuleType.Size,
    enabled: true,
    direction: TorrentSelectionDirection.Desc,
    indexerPreference: {
      trackerIds: [] as string[]
    },
    titleMatch: {
      clauses: [] as Array<{
        pattern: string;
        patternMode: TitleMatchPatternMode;
        effect: TitleMatchEffect;
      }>
    }
  }
];

export const EMPTY_STASH_FORM = {
  url: "",
  apiKey: ""
};

export const EMPTY_INGEST_FORM = {
  deliveryMode: "PATH_MAP",
  stashLibraryPath: "",
  transfer: {
    action: "",
    mojiSourceRoot: "",
    mojiTargetRoot: ""
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
  torrentSelection: {
    enabled: true,
    rules: DEFAULT_TORRENT_SELECTION_RULES
  }
};

export const EMPTY_SYSTEM_FORM = {
  taskDeletePolicy: "KEEP_ONLY"
};

export const TOAST_LIFETIME_MS = 10000;
export const TOAST_EXIT_MS = 480;

export const SUBSCRIPTION_PAGE_SIZE_OPTIONS = [12, 24, 48, 96] as const;
export const LOG_LEVEL_OPTIONS: LogLevel[] = [LogLevel.Debug, LogLevel.Info, LogLevel.Warning, LogLevel.Error];
