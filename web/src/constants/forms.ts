import {
  LogLevel,
  TorrentFileMatchEffect,
  TitleMatchEffect,
  TitleMatchPatternMode,
  TorrentSelectionDirection,
  TorrentSelectionRuleType,
} from "../graphql/generated/graphql";

export const DEFAULT_TORRENT_SELECTION_RULES = [
  {
    id: "default-seeders",
    name: "Seeders",
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
    },
    torrentFileNameMatch: {
      clauses: [] as Array<{
        pattern: string;
        patternMode: TitleMatchPatternMode;
        effect: TorrentFileMatchEffect;
      }>
    }
  },
  {
    id: "default-size",
    name: "Size",
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
    },
    torrentFileNameMatch: {
      clauses: [] as Array<{
        pattern: string;
        patternMode: TitleMatchPatternMode;
        effect: TorrentFileMatchEffect;
      }>
    }
  }
];

export const DEFAULT_TORRENT_FILE_INSPECTION_RULES = [
  {
    id: "default-torrent-single-video",
    name: "Single Video",
    type: TorrentSelectionRuleType.TorrentSingleVideo,
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
    },
    torrentFileNameMatch: {
      clauses: [] as Array<{
        pattern: string;
        patternMode: TitleMatchPatternMode;
        effect: TorrentFileMatchEffect;
      }>
    }
  },
  {
    id: "default-torrent-file-name-match",
    name: "Torrent File Name Match",
    type: TorrentSelectionRuleType.TorrentFileNameMatch,
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
    },
    torrentFileNameMatch: {
      clauses: [] as Array<{
        pattern: string;
        patternMode: TitleMatchPatternMode;
        effect: TorrentFileMatchEffect;
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
  torrentSelection: {
    enabled: true,
    inspectionCandidateLimit: "5",
    rules: [...DEFAULT_TORRENT_SELECTION_RULES, ...DEFAULT_TORRENT_FILE_INSPECTION_RULES]
  }
};

export const EMPTY_SYSTEM_FORM = {
  taskDeletePolicy: "KEEP_ONLY"
};

export const SUBSCRIPTION_PAGE_SIZE_OPTIONS = [12, 24, 48, 96] as const;
export const LOG_LEVEL_OPTIONS: LogLevel[] = [LogLevel.Debug, LogLevel.Info, LogLevel.Warning, LogLevel.Error];
