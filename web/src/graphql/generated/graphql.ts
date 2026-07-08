/* eslint-disable */
import { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  Long: { input: any; output: any; }
};

export type AutomationSettings = {
  __typename?: 'AutomationSettings';
  /** Endpoint URLs in the user-defined order used for subscription lookups. Endpoints not listed here are still queried, in their Stash order, appended after the listed ones. An empty list means use Stash's order as-is. */
  stashBoxEndpoints: Array<Scalars['String']['output']>;
  subscriptionPollIntervalHours: Scalars['Int']['output'];
  subscriptionReleasePolicy: SubscriptionReleasePolicy;
  taskProgressSyncIntervalSeconds: Scalars['Int']['output'];
  torrentSelection: TorrentSelectionSettings;
};

export type AutomationStatus = {
  __typename?: 'AutomationStatus';
  subscriptionPollEnabled: Scalars['Boolean']['output'];
  subscriptionPollIntervalHours: Scalars['Int']['output'];
  taskProgressSyncEnabled: Scalars['Boolean']['output'];
  taskProgressSyncIntervalSeconds: Scalars['Int']['output'];
};

export type DashboardStats = {
  __typename?: 'DashboardStats';
  active: Scalars['Int']['output'];
  completed: Scalars['Int']['output'];
  downloading: Scalars['Int']['output'];
  failed: Scalars['Int']['output'];
  pendingScans: Scalars['Int']['output'];
  total: Scalars['Int']['output'];
};

export type DirectionRule = {
  __typename?: 'DirectionRule';
  direction: TorrentSelectionDirection;
};

export type DirectionRuleInput = {
  direction: TorrentSelectionDirection;
};

export type DiscoverSceneConnection = {
  __typename?: 'DiscoverSceneConnection';
  fallbackCount: Scalars['Int']['output'];
  items: Array<DiscoveredScene>;
  searchedQuery: Scalars['String']['output'];
  usedStashBox?: Maybe<MatchedStashBox>;
};

export type DiscoverScenesInput = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  query: Scalars['String']['input'];
  sortBy?: InputMaybe<DiscoverSortBy>;
};

export enum DiscoverSortBy {
  DateAsc = 'DATE_ASC',
  DateDesc = 'DATE_DESC',
  DurationDesc = 'DURATION_DESC',
  Relevance = 'RELEVANCE',
  TitleAsc = 'TITLE_ASC'
}

export type DiscoveredScene = {
  __typename?: 'DiscoveredScene';
  code?: Maybe<Scalars['String']['output']>;
  date?: Maybe<Scalars['String']['output']>;
  derivedQuery: Scalars['String']['output'];
  durationSeconds?: Maybe<Scalars['Int']['output']>;
  imageUrl?: Maybe<Scalars['String']['output']>;
  key: Scalars['ID']['output'];
  performerNames: Array<Scalars['String']['output']>;
  sceneId: Scalars['ID']['output'];
  stashBoxEndpoint: Scalars['String']['output'];
  stashBoxName: Scalars['String']['output'];
  studioName?: Maybe<Scalars['String']['output']>;
  title: Scalars['String']['output'];
  url?: Maybe<Scalars['String']['output']>;
};

export type DownloadCandidate = {
  __typename?: 'DownloadCandidate';
  infoHash: Scalars['String']['output'];
  link: Scalars['String']['output'];
  magnetUri: Scalars['String']['output'];
  peers: Scalars['Int']['output'];
  seeders: Scalars['Int']['output'];
  size: Scalars['Long']['output'];
  title: Scalars['String']['output'];
  tracker: Scalars['String']['output'];
};

export type DownloadMediaInput = {
  categories?: InputMaybe<Array<Scalars['Int']['input']>>;
  category?: InputMaybe<Scalars['String']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  paused?: InputMaybe<Scalars['Boolean']['input']>;
  query: Scalars['String']['input'];
  savePath?: InputMaybe<Scalars['String']['input']>;
  tags?: InputMaybe<Scalars['String']['input']>;
  trackers?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type DownloadsIngestSettings = {
  __typename?: 'DownloadsIngestSettings';
  mojiRoot: Scalars['String']['output'];
  qbRoot: Scalars['String']['output'];
};

export type DownloadsIngestSettingsInput = {
  mojiRoot: Scalars['String']['input'];
  qbRoot: Scalars['String']['input'];
};

/** Basic service health */
export type Health = {
  __typename?: 'Health';
  message: Scalars['String']['output'];
  ok: Scalars['Boolean']['output'];
};

export type IndexerPreferenceRule = {
  __typename?: 'IndexerPreferenceRule';
  trackerIds: Array<Scalars['String']['output']>;
};

export type IndexerPreferenceRuleInput = {
  trackerIds: Array<Scalars['String']['input']>;
};

export type IngestSettings = {
  __typename?: 'IngestSettings';
  deliveryMode: Scalars['String']['output'];
  downloads: DownloadsIngestSettings;
  library: LibraryIngestSettings;
  transfer: TransferIngestSettings;
};

export type IngestStatus = {
  __typename?: 'IngestStatus';
  /** Whether the ingest pipeline is fully wired for the selected mode. Becomes true only when the mode-specific fields are all filled in; reaches false as soon as any required field is cleared. */
  configured: Scalars['Boolean']['output'];
};

export type JackettIndexer = {
  __typename?: 'JackettIndexer';
  /** Mirrors Jackett's Configured flag — indexers not yet configured are hidden by Jackett's UI. */
  enabled: Scalars['Boolean']['output'];
  id: Scalars['String']['output'];
  name: Scalars['String']['output'];
};

export type JackettSearchInput = {
  categories?: InputMaybe<Array<Scalars['Int']['input']>>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  query: Scalars['String']['input'];
  sortBy?: InputMaybe<JackettSortBy>;
  trackers?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type JackettSearchResult = {
  __typename?: 'JackettSearchResult';
  categoryDesc: Scalars['String']['output'];
  details: Scalars['String']['output'];
  infoHash: Scalars['String']['output'];
  link: Scalars['String']['output'];
  magnetUri: Scalars['String']['output'];
  peers: Scalars['Int']['output'];
  publishDate: Scalars['String']['output'];
  seeders: Scalars['Int']['output'];
  size: Scalars['Long']['output'];
  title: Scalars['String']['output'];
  tracker: Scalars['String']['output'];
  trackerId: Scalars['String']['output'];
};

export type JackettSettings = {
  __typename?: 'JackettSettings';
  /** Currently configured Jackett API key. Returned in plaintext for the settings UI; never logged. */
  apiKey: Scalars['String']['output'];
  apiKeyConfigured: Scalars['Boolean']['output'];
  configured: Scalars['Boolean']['output'];
  /** Currently configured Jackett dashboard password. Returned in plaintext for the settings UI; never logged. */
  password: Scalars['String']['output'];
  passwordConfigured: Scalars['Boolean']['output'];
  url: Scalars['String']['output'];
};

export enum JackettSortBy {
  DateDesc = 'DATE_DESC',
  Relevance = 'RELEVANCE',
  SeedersDesc = 'SEEDERS_DESC',
  SizeDesc = 'SIZE_DESC'
}

export type JackettStats = {
  __typename?: 'JackettStats';
  /** Subset of indexerCount that are marked as configured in Jackett. */
  configuredIndexerCount: Scalars['Int']['output'];
  /** Total indexers reported by Jackett (configured + unconfigured). */
  indexerCount: Scalars['Int']['output'];
  /** Most recent error message from any Jackett-side refresh. Null = OK. */
  lastError?: Maybe<Scalars['String']['output']>;
  /** First non-empty Error field from the most recent search, if any. */
  lastIndexerError?: Maybe<Scalars['String']['output']>;
  /** Worst indexer latency (ms) from the most recent /all/results search. 0 if no search has happened. */
  lastIndexerLatencyMs: Scalars['Int']['output'];
  /** Timestamp of the most recent /all/results search. Null if no search has happened yet. */
  lastIndexerSearchAt?: Maybe<Scalars['String']['output']>;
  /** ISO 8601 timestamp of the most recent successful refresh. Null until the first probe completes successfully. */
  okAt?: Maybe<Scalars['String']['output']>;
};

export enum LibraryFilter {
  All = 'ALL',
  InLibrary = 'IN_LIBRARY',
  NotInLibrary = 'NOT_IN_LIBRARY'
}

export type LibraryIngestSettings = {
  __typename?: 'LibraryIngestSettings';
  mojiRoot: Scalars['String']['output'];
  stashRoot: Scalars['String']['output'];
};

export type LibraryIngestSettingsInput = {
  mojiRoot: Scalars['String']['input'];
  stashRoot: Scalars['String']['input'];
};

export type LogEntry = {
  __typename?: 'LogEntry';
  level: LogLevel;
  message: Scalars['String']['output'];
  time: Scalars['String']['output'];
};

export enum LogLevel {
  Debug = 'Debug',
  Error = 'Error',
  Info = 'Info',
  Warning = 'Warning'
}

export type MatchedStashBox = {
  __typename?: 'MatchedStashBox';
  endpoint: Scalars['String']['output'];
  name: Scalars['String']['output'];
  performerId: Scalars['ID']['output'];
  performerName: Scalars['String']['output'];
};

export type Mutation = {
  __typename?: 'Mutation';
  /** Add a torrent URL or magnet and create a persisted Moji task */
  addTorrent: Task;
  /** Delete a persisted Moji task record */
  deleteTask: Task;
  /** Search torrent candidates and create a Moji download task */
  downloadMedia: Task;
  /**
   * Add a torrent to qBittorrent via magnet or http(s) URL
   * @deprecated Use addTorrent for persisted Moji task tracking.
   */
  qbittorrentAdd: Scalars['Boolean']['output'];
  /** Queue a discovered StashBox scene into the standard download workflow */
  queueDiscoveredScene: Task;
  /** Refresh a subscribed performer against the configured release source */
  refreshSubscribedPerformer: SubscribedPerformer;
  /** Re-fetch the Stash-Box list from the configured Stash server. The updated list and load status are reflected in the returned Settings snapshot. */
  refreshSubscriptionStashBoxes: Settings;
  /** Refresh all subscribed performers against the configured release source */
  refreshSubscriptionsNow: Array<SubscribedPerformer>;
  /** Start a Stash metadata scan */
  stashMetadataScan: Scalars['ID']['output'];
  /** Mark a Stash performer as subscribed by Moji */
  subscribePerformer: SubscribedPerformer;
  /** Synchronize Moji task progress from qBittorrent */
  syncTaskProgress: Array<Task>;
  /** Trigger Stash metadata scans for completed Moji tasks */
  triggerStashScans: Array<Task>;
  /** Trigger a Stash metadata scan for a single completed Moji task */
  triggerTaskStashScan: Task;
  /** Remove Moji subscription mark from a Stash performer */
  unsubscribePerformer: Scalars['Boolean']['output'];
  /** Update automation settings and persist them to backend config */
  updateAutomationSettings: Settings;
  /** Update ingest settings and persist them to backend config */
  updateIngestSettings: Settings;
  /** Update Jackett settings and persist them to backend config */
  updateJackettSettings: Settings;
  /** Update qBittorrent settings and persist them to backend config */
  updateQBittorrentSettings: Settings;
  /** Update Stash settings and persist them to backend config */
  updateStashSettings: Settings;
  /** Update system settings and persist them to backend config */
  updateSystemSettings: Settings;
};


export type MutationAddTorrentArgs = {
  input: QBittorrentAddInput;
};


export type MutationDeleteTaskArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDownloadMediaArgs = {
  input: DownloadMediaInput;
};


export type MutationQbittorrentAddArgs = {
  input: QBittorrentAddInput;
};


export type MutationQueueDiscoveredSceneArgs = {
  input: QueueDiscoveredSceneInput;
};


export type MutationRefreshSubscribedPerformerArgs = {
  stashPerformerID: Scalars['ID']['input'];
};


export type MutationStashMetadataScanArgs = {
  input: StashMetadataScanInput;
};


export type MutationSubscribePerformerArgs = {
  stashPerformerID: Scalars['ID']['input'];
};


export type MutationTriggerTaskStashScanArgs = {
  id: Scalars['ID']['input'];
};


export type MutationUnsubscribePerformerArgs = {
  stashPerformerID: Scalars['ID']['input'];
};


export type MutationUpdateAutomationSettingsArgs = {
  input: UpdateAutomationSettingsInput;
};


export type MutationUpdateIngestSettingsArgs = {
  input: UpdateIngestSettingsInput;
};


export type MutationUpdateJackettSettingsArgs = {
  input: UpdateJackettSettingsInput;
};


export type MutationUpdateQBittorrentSettingsArgs = {
  input: UpdateQBittorrentSettingsInput;
};


export type MutationUpdateStashSettingsArgs = {
  input: UpdateStashSettingsInput;
};


export type MutationUpdateSystemSettingsArgs = {
  input: UpdateSystemSettingsInput;
};

export type PreviewJackettSelectionCandidateInput = {
  categoryDesc: Scalars['String']['input'];
  details: Scalars['String']['input'];
  infoHash: Scalars['String']['input'];
  link: Scalars['String']['input'];
  magnetUri: Scalars['String']['input'];
  peers: Scalars['Int']['input'];
  publishDate: Scalars['String']['input'];
  seeders: Scalars['Int']['input'];
  size: Scalars['Long']['input'];
  title: Scalars['String']['input'];
  tracker: Scalars['String']['input'];
  trackerId: Scalars['String']['input'];
};

export type PreviewJackettSelectionInput = {
  applyFastRules: Scalars['Boolean']['input'];
  applyFileRules: Scalars['Boolean']['input'];
  inspectionCandidateLimit?: InputMaybe<Scalars['Int']['input']>;
  query: Scalars['String']['input'];
  results: Array<PreviewJackettSelectionCandidateInput>;
};

export type PreviewJackettSelectionMeta = {
  __typename?: 'PreviewJackettSelectionMeta';
  appliedFastRules: Scalars['Boolean']['output'];
  appliedFileRules: Scalars['Boolean']['output'];
  inspectableCount: Scalars['Int']['output'];
  inspectedCount: Scalars['Int']['output'];
};

export type PreviewJackettSelectionResult = {
  __typename?: 'PreviewJackettSelectionResult';
  previewMeta: PreviewJackettSelectionMeta;
  results: Array<JackettSearchResult>;
};

export type QbTorrent = {
  __typename?: 'QBTorrent';
  addedOn: Scalars['Long']['output'];
  category: Scalars['String']['output'];
  dlspeed: Scalars['Long']['output'];
  eta: Scalars['Long']['output'];
  hash: Scalars['String']['output'];
  name: Scalars['String']['output'];
  progress: Scalars['Float']['output'];
  size: Scalars['Long']['output'];
  state: Scalars['String']['output'];
  tags: Scalars['String']['output'];
  upspeed: Scalars['Long']['output'];
};

export type QBittorrentAddInput = {
  category?: InputMaybe<Scalars['String']['input']>;
  paused?: InputMaybe<Scalars['Boolean']['input']>;
  savePath?: InputMaybe<Scalars['String']['input']>;
  tags?: InputMaybe<Scalars['String']['input']>;
  url: Scalars['String']['input'];
};

export type QBittorrentSettings = {
  __typename?: 'QBittorrentSettings';
  category: Scalars['String']['output'];
  configured: Scalars['Boolean']['output'];
  defaultSavePath: Scalars['String']['output'];
  /** Currently configured qBittorrent password. Returned in plaintext for the settings UI; never logged. */
  password: Scalars['String']['output'];
  passwordConfigured: Scalars['Boolean']['output'];
  tags: Scalars['String']['output'];
  url: Scalars['String']['output'];
  username: Scalars['String']['output'];
  usernameConfigured: Scalars['Boolean']['output'];
};

export type QBittorrentStats = {
  __typename?: 'QBittorrentStats';
  /** Count of torrents matching qBittorrent filter "active". */
  activeTorrentCount: Scalars['Int']['output'];
  /** Whether qBittorrent's alternative speed limits are enabled. */
  altSpeedLimitEnabled: Scalars['Boolean']['output'];
  /** qBittorrent connection status: connected | firewalled | disconnected. */
  connectionStatus: Scalars['String']['output'];
  /** Global download rate in bytes/sec. */
  downloadSpeed: Scalars['Int']['output'];
  /** Most recent error message from any qBittorrent-side refresh. Null = OK. */
  lastError?: Maybe<Scalars['String']['output']>;
  /** ISO 8601 timestamp of the most recent successful refresh. Null until the first probe completes successfully. */
  okAt?: Maybe<Scalars['String']['output']>;
  /** Global upload rate in bytes/sec. */
  uploadSpeed: Scalars['Int']['output'];
};

export type Query = {
  __typename?: 'Query';
  /** Get aggregate task stats for the dashboard and task center */
  dashboardStats: DashboardStats;
  /** Search scene metadata from preferred StashBox sources */
  discoverScenes: DiscoverSceneConnection;
  health: Health;
  /** List the indexers Jackett currently exposes. Returns [] when Jackett is not configured. */
  jackettIndexers: Array<JackettIndexer>;
  /** Search torrents via Jackett as a fallback-only power-user tool */
  jackettSearch: Array<JackettSearchResult>;
  /** Retrieve recent Moji logs for troubleshooting */
  logs: Array<LogEntry>;
  /** Preview automatic torrent-selection ordering on an existing Jackett result set */
  previewJackettSelection: PreviewJackettSelectionResult;
  /** List torrents from qBittorrent */
  qbittorrentTorrents: Array<QbTorrent>;
  /** Get editable configuration for the Settings surface */
  settings: Settings;
  /** Get runtime state for the Settings surface */
  settingsStatus: SettingsStatus;
  /** Get a Stash background job by id */
  stashJob?: Maybe<StashJob>;
  /** Fetch Stash performer detail with Moji / StashBox context */
  stashPerformerDetail: StashPerformerDetail;
  /** List deduplicated performer scenes from Stash and the preferred StashBox */
  stashPerformerScenes: StashPerformerSceneConnection;
  /** List Stash performers with current Moji subscription state */
  stashPerformers: StashPerformerConnection;
  /** List performers currently subscribed by Moji */
  subscribedPerformers: Array<SubscribedPerformer>;
  /** Get a Moji download task by id */
  task?: Maybe<Task>;
  /** List Moji download tasks, newest first */
  tasks: Array<Task>;
  version: Scalars['String']['output'];
};


export type QueryDiscoverScenesArgs = {
  input: DiscoverScenesInput;
};


export type QueryJackettSearchArgs = {
  input: JackettSearchInput;
};


export type QueryLogsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  minLevel?: InputMaybe<LogLevel>;
};


export type QueryPreviewJackettSelectionArgs = {
  input: PreviewJackettSelectionInput;
};


export type QueryQbittorrentTorrentsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
};


export type QueryStashJobArgs = {
  id: Scalars['ID']['input'];
};


export type QueryStashPerformerDetailArgs = {
  id: Scalars['ID']['input'];
};


export type QueryStashPerformerScenesArgs = {
  id: Scalars['ID']['input'];
  input: StashPerformerScenesInput;
};


export type QueryStashPerformersArgs = {
  page?: InputMaybe<Scalars['Int']['input']>;
  pageSize?: InputMaybe<Scalars['Int']['input']>;
  search?: InputMaybe<Scalars['String']['input']>;
};


export type QueryTaskArgs = {
  id: Scalars['ID']['input'];
};

export type QueueDiscoveredSceneInput = {
  sceneId: Scalars['ID']['input'];
  stashBoxEndpoint: Scalars['String']['input'];
};

export enum SceneSource {
  Stash = 'STASH',
  Stashbox = 'STASHBOX'
}

export enum SceneSourceFilter {
  All = 'ALL',
  Stash = 'STASH',
  Stashbox = 'STASHBOX'
}

export type ServiceStatus = {
  __typename?: 'ServiceStatus';
  /** True iff the minimum connection fields are present, so the backend can attempt to talk to the upstream service. */
  configured: Scalars['Boolean']['output'];
  /** True iff the upstream service is configured AND a recent probe succeeded. A probe result older than ~4 minutes, or a failed probe, returns false. The proximate cause of a non-ready state lives on the corresponding *Stats.lastError. */
  ready: Scalars['Boolean']['output'];
};

export type Settings = {
  __typename?: 'Settings';
  automation: AutomationSettings;
  ingest: IngestSettings;
  jackett: JackettSettings;
  qbittorrent: QBittorrentSettings;
  stash: StashSettings;
  system: SystemSettings;
};

export type SettingsStatus = {
  __typename?: 'SettingsStatus';
  automation: AutomationStatus;
  ingest: IngestStatus;
  jackett: ServiceStatus;
  /** Runtime stats for the Jackett indexer aggregator. Refreshed by the stats collector. */
  jackettStats: JackettStats;
  qbittorrent: ServiceStatus;
  /** Runtime stats for the qBittorrent download client. Refreshed by the stats collector. */
  qbittorrentStats: QBittorrentStats;
  stash: ServiceStatus;
  stashLibraries: Array<StashLibrary>;
  stashLibrariesLoadError?: Maybe<Scalars['String']['output']>;
  /** Runtime stats for the Stash server. Refreshed by the stats collector. */
  stashStats: StashStats;
  subscription: SubscriptionStatus;
};

export type StashBoxEndpoint = {
  __typename?: 'StashBoxEndpoint';
  apiKeyConfigured: Scalars['Boolean']['output'];
  endpoint: Scalars['String']['output'];
  name: Scalars['String']['output'];
};

export type StashJob = {
  __typename?: 'StashJob';
  addTime: Scalars['String']['output'];
  description: Scalars['String']['output'];
  endTime?: Maybe<Scalars['String']['output']>;
  error?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  progress?: Maybe<Scalars['Float']['output']>;
  startTime?: Maybe<Scalars['String']['output']>;
  status: Scalars['String']['output'];
  subTasks?: Maybe<Array<Scalars['String']['output']>>;
};

export type StashLibrary = {
  __typename?: 'StashLibrary';
  path: Scalars['String']['output'];
};

export type StashMetadataScanInput = {
  paths?: InputMaybe<Array<Scalars['String']['input']>>;
  rescan?: InputMaybe<Scalars['Boolean']['input']>;
  scanGenerateClipPreviews?: InputMaybe<Scalars['Boolean']['input']>;
  scanGenerateCovers?: InputMaybe<Scalars['Boolean']['input']>;
  scanGenerateImagePreviews?: InputMaybe<Scalars['Boolean']['input']>;
  scanGeneratePhashes?: InputMaybe<Scalars['Boolean']['input']>;
  scanGeneratePreviews?: InputMaybe<Scalars['Boolean']['input']>;
  scanGenerateSprites?: InputMaybe<Scalars['Boolean']['input']>;
  scanGenerateThumbnails?: InputMaybe<Scalars['Boolean']['input']>;
};

export type StashPerformer = {
  __typename?: 'StashPerformer';
  aliasList: Array<Scalars['String']['output']>;
  favorite: Scalars['Boolean']['output'];
  id: Scalars['ID']['output'];
  imagePath?: Maybe<Scalars['String']['output']>;
  name: Scalars['String']['output'];
  sceneCount: Scalars['Int']['output'];
  subscribed: Scalars['Boolean']['output'];
};

export type StashPerformerConnection = {
  __typename?: 'StashPerformerConnection';
  hasNextPage: Scalars['Boolean']['output'];
  hasPrevPage: Scalars['Boolean']['output'];
  items: Array<StashPerformer>;
  page: Scalars['Int']['output'];
  pageSize: Scalars['Int']['output'];
  totalCount: Scalars['Int']['output'];
  totalPages: Scalars['Int']['output'];
};

export type StashPerformerDetail = {
  __typename?: 'StashPerformerDetail';
  birthdate?: Maybe<Scalars['String']['output']>;
  country?: Maybe<Scalars['String']['output']>;
  dedupedSceneCount: Scalars['Int']['output'];
  disambiguation?: Maybe<Scalars['String']['output']>;
  ethnicity?: Maybe<Scalars['String']['output']>;
  eyeColor?: Maybe<Scalars['String']['output']>;
  heightCm?: Maybe<Scalars['Int']['output']>;
  matchedStashBox?: Maybe<MatchedStashBox>;
  performer: StashPerformer;
  rating100?: Maybe<Scalars['Int']['output']>;
  stashBoxSceneCount: Scalars['Int']['output'];
  stashSceneCount: Scalars['Int']['output'];
  totalSceneCount: Scalars['Int']['output'];
  urls: Array<Scalars['String']['output']>;
};

export type StashPerformerScene = {
  __typename?: 'StashPerformerScene';
  code?: Maybe<Scalars['String']['output']>;
  date?: Maybe<Scalars['String']['output']>;
  hasStashBoxSource: Scalars['Boolean']['output'];
  hasStashSource: Scalars['Boolean']['output'];
  imageUrl?: Maybe<Scalars['String']['output']>;
  inLibrary: Scalars['Boolean']['output'];
  key: Scalars['ID']['output'];
  matchedStashSceneId?: Maybe<Scalars['ID']['output']>;
  primarySource: SceneSource;
  sourceLabels: Array<Scalars['String']['output']>;
  sourceSceneId: Scalars['ID']['output'];
  stashBoxEndpoint?: Maybe<Scalars['String']['output']>;
  stashBoxSceneId?: Maybe<Scalars['ID']['output']>;
  stashIds: Array<StashSceneId>;
  studioName?: Maybe<Scalars['String']['output']>;
  title?: Maybe<Scalars['String']['output']>;
  url?: Maybe<Scalars['String']['output']>;
};

export type StashPerformerSceneConnection = {
  __typename?: 'StashPerformerSceneConnection';
  dedupedCount: Scalars['Int']['output'];
  hasNextPage: Scalars['Boolean']['output'];
  hasPrevPage: Scalars['Boolean']['output'];
  items: Array<StashPerformerScene>;
  page: Scalars['Int']['output'];
  pageSize: Scalars['Int']['output'];
  stashBoxCount: Scalars['Int']['output'];
  stashSceneCount: Scalars['Int']['output'];
  totalCount: Scalars['Int']['output'];
  totalPages: Scalars['Int']['output'];
};

export type StashPerformerScenesInput = {
  inLibrary?: InputMaybe<LibraryFilter>;
  page?: InputMaybe<Scalars['Int']['input']>;
  pageSize?: InputMaybe<Scalars['Int']['input']>;
  search?: InputMaybe<Scalars['String']['input']>;
  source?: InputMaybe<SceneSourceFilter>;
};

export type StashSceneId = {
  __typename?: 'StashSceneID';
  endpoint: Scalars['String']['output'];
  stashId: Scalars['String']['output'];
};

export type StashSettings = {
  __typename?: 'StashSettings';
  /** Currently configured Stash API key. Returned in plaintext for the settings UI; never logged. */
  apiKey: Scalars['String']['output'];
  apiKeyConfigured: Scalars['Boolean']['output'];
  configured: Scalars['Boolean']['output'];
  url: Scalars['String']['output'];
};

/** Per-service runtime stats. okAt is the timestamp of the most recent successful refresh; lastError is the message from the most recent failed refresh (if any). When lastError is non-null, other numeric fields still reflect the last known-good snapshot. */
export type StashStats = {
  __typename?: 'StashStats';
  /** Most recent error message from any Stash-side refresh. Null = OK. */
  lastError?: Maybe<Scalars['String']['output']>;
  /** ISO 8601 timestamp of the most recent successful refresh. Null until the first probe completes successfully. */
  okAt?: Maybe<Scalars['String']['output']>;
  /** Number of Moji-owned tasks currently in RUNNING or READY stash-scan state. */
  pendingMojiScanCount: Scalars['Int']['output'];
  /** Total scenes in the Stash library. Null if not yet fetched or query failed. */
  sceneCount?: Maybe<Scalars['Int']['output']>;
  /** Stash server version string, e.g. 0.27.2. Null if not yet fetched. */
  version?: Maybe<Scalars['String']['output']>;
};

export type SubscribedPerformer = {
  __typename?: 'SubscribedPerformer';
  lastCheckedAt?: Maybe<Scalars['String']['output']>;
  lastError?: Maybe<Scalars['String']['output']>;
  pendingReleaseCount: Scalars['Int']['output'];
  performer: StashPerformer;
  processedReleaseCount: Scalars['Int']['output'];
  recentReleases: Array<SubscriptionRelease>;
};

export type SubscriptionRelease = {
  __typename?: 'SubscriptionRelease';
  classification: SubscriptionReleaseClassification;
  code?: Maybe<Scalars['String']['output']>;
  date?: Maybe<Scalars['String']['output']>;
  decision: SubscriptionReleaseDecision;
  decisionReason: Scalars['String']['output'];
  key: Scalars['ID']['output'];
  performerCount: Scalars['Int']['output'];
  performerNames: Array<Scalars['String']['output']>;
  query: Scalars['String']['output'];
  seenAt: Scalars['String']['output'];
  source: Scalars['String']['output'];
  taskID?: Maybe<Scalars['ID']['output']>;
  title: Scalars['String']['output'];
  url?: Maybe<Scalars['String']['output']>;
};

export enum SubscriptionReleaseBehavior {
  Block = 'BLOCK',
  Download = 'DOWNLOAD',
  Review = 'REVIEW'
}

export enum SubscriptionReleaseClassification {
  CompilationLike = 'COMPILATION_LIKE',
  LargeGroup = 'LARGE_GROUP',
  SmallGroup = 'SMALL_GROUP',
  Solo = 'SOLO',
  Unknown = 'UNKNOWN'
}

export enum SubscriptionReleaseDateRange {
  All = 'ALL',
  FiveYears = 'FIVE_YEARS',
  OneYear = 'ONE_YEAR',
  ThreeYears = 'THREE_YEARS',
  TwoYears = 'TWO_YEARS'
}

export enum SubscriptionReleaseDecision {
  Blocked = 'BLOCKED',
  Downloaded = 'DOWNLOADED',
  Queued = 'QUEUED'
}

export type SubscriptionReleasePolicy = {
  __typename?: 'SubscriptionReleasePolicy';
  compilationBehavior: SubscriptionReleaseBehavior;
  groupBehavior: SubscriptionReleaseBehavior;
  maxGroupPerformerCount: Scalars['Int']['output'];
  releaseDateRange: SubscriptionReleaseDateRange;
  soloBehavior: SubscriptionReleaseBehavior;
};

export type SubscriptionReleasePolicyInput = {
  compilationBehavior: SubscriptionReleaseBehavior;
  groupBehavior: SubscriptionReleaseBehavior;
  maxGroupPerformerCount: Scalars['Int']['input'];
  releaseDateRange: SubscriptionReleaseDateRange;
  soloBehavior: SubscriptionReleaseBehavior;
};

export type SubscriptionStatus = {
  __typename?: 'SubscriptionStatus';
  /** Stash Box instances currently configured inside the Stash server. */
  stashBoxes: Array<StashBoxEndpoint>;
  /** Reason for the most recent Stash Box load failure. Null when stashBoxesLoaded is true. */
  stashBoxesLoadError?: Maybe<Scalars['String']['output']>;
  /** Whether the last attempt to load Stash Box endpoints from Stash succeeded. */
  stashBoxesLoaded: Scalars['Boolean']['output'];
};

export type SystemSettings = {
  __typename?: 'SystemSettings';
  taskDeletePolicy: TaskDeletePolicy;
};

export type Task = {
  __typename?: 'Task';
  candidate: DownloadCandidate;
  category: Scalars['String']['output'];
  code: Scalars['String']['output'];
  completedAt?: Maybe<Scalars['String']['output']>;
  contentPath: Scalars['String']['output'];
  createdAt: Scalars['String']['output'];
  error: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  progress: Scalars['Float']['output'];
  qbittorrentState: Scalars['String']['output'];
  query: Scalars['String']['output'];
  savePath: Scalars['String']['output'];
  source: TaskSource;
  stashJobId: Scalars['ID']['output'];
  stashMode: Scalars['String']['output'];
  stashScanError: Scalars['String']['output'];
  stashScanHint: Scalars['String']['output'];
  stashScanPath: Scalars['String']['output'];
  stashScanStartedAt?: Maybe<Scalars['String']['output']>;
  stashScanStatus: Scalars['String']['output'];
  stashSourcePath: Scalars['String']['output'];
  stashTransferAction: Scalars['String']['output'];
  stashTransferError: Scalars['String']['output'];
  stashTransferPath: Scalars['String']['output'];
  stashTransferStatus: Scalars['String']['output'];
  status: Scalars['String']['output'];
  tags: Scalars['String']['output'];
  torrentHash: Scalars['String']['output'];
  torrentName: Scalars['String']['output'];
  torrentUrl: Scalars['String']['output'];
  updatedAt: Scalars['String']['output'];
};

export enum TaskDeletePolicy {
  KeepOnly = 'KEEP_ONLY',
  RemoveTorrent = 'REMOVE_TORRENT',
  RemoveTorrentAndFiles = 'REMOVE_TORRENT_AND_FILES'
}

export enum TaskSource {
  Manual = 'MANUAL',
  Search = 'SEARCH',
  Subscription = 'SUBSCRIPTION'
}

export type TitleMatchClause = {
  __typename?: 'TitleMatchClause';
  effect: TitleMatchEffect;
  pattern: Scalars['String']['output'];
  patternMode: TitleMatchPatternMode;
};

export type TitleMatchClauseInput = {
  effect: TitleMatchEffect;
  pattern: Scalars['String']['input'];
  patternMode: TitleMatchPatternMode;
};

export enum TitleMatchEffect {
  Avoid = 'AVOID',
  Prefer = 'PREFER'
}

export enum TitleMatchPatternMode {
  Plain = 'PLAIN',
  Regex = 'REGEX'
}

export type TitleMatchRule = {
  __typename?: 'TitleMatchRule';
  clauses: Array<TitleMatchClause>;
};

export type TitleMatchRuleInput = {
  clauses: Array<TitleMatchClauseInput>;
};

export enum TorrentFileMatchEffect {
  Avoid = 'AVOID',
  Lock = 'LOCK',
  Prefer = 'PREFER'
}

export type TorrentFileNameMatchClause = {
  __typename?: 'TorrentFileNameMatchClause';
  effect: TorrentFileMatchEffect;
  pattern: Scalars['String']['output'];
  patternMode: TitleMatchPatternMode;
};

export type TorrentFileNameMatchClauseInput = {
  effect: TorrentFileMatchEffect;
  pattern: Scalars['String']['input'];
  patternMode: TitleMatchPatternMode;
};

export type TorrentFileNameMatchRule = {
  __typename?: 'TorrentFileNameMatchRule';
  clauses: Array<TorrentFileNameMatchClause>;
};

export type TorrentFileNameMatchRuleInput = {
  clauses: Array<TorrentFileNameMatchClauseInput>;
};

export enum TorrentSelectionDirection {
  Asc = 'ASC',
  Desc = 'DESC'
}

export type TorrentSelectionRule = {
  __typename?: 'TorrentSelectionRule';
  enabled: Scalars['Boolean']['output'];
  indexerPreference: IndexerPreferenceRule;
  publishDate: DirectionRule;
  seeders: DirectionRule;
  size: DirectionRule;
  titleMatch: TitleMatchRule;
  torrentFileNameMatch: TorrentFileNameMatchRule;
  type: TorrentSelectionRuleType;
};

export type TorrentSelectionRuleInput = {
  enabled: Scalars['Boolean']['input'];
  indexerPreference?: InputMaybe<IndexerPreferenceRuleInput>;
  publishDate?: InputMaybe<DirectionRuleInput>;
  seeders?: InputMaybe<DirectionRuleInput>;
  size?: InputMaybe<DirectionRuleInput>;
  titleMatch?: InputMaybe<TitleMatchRuleInput>;
  torrentFileNameMatch?: InputMaybe<TorrentFileNameMatchRuleInput>;
  type: TorrentSelectionRuleType;
};

export enum TorrentSelectionRuleType {
  IndexerPreference = 'INDEXER_PREFERENCE',
  PublishDate = 'PUBLISH_DATE',
  Seeders = 'SEEDERS',
  Size = 'SIZE',
  TitleMatch = 'TITLE_MATCH',
  TitleSimilarity = 'TITLE_SIMILARITY',
  TorrentFileNameMatch = 'TORRENT_FILE_NAME_MATCH',
  TorrentSingleVideo = 'TORRENT_SINGLE_VIDEO'
}

export type TorrentSelectionSettings = {
  __typename?: 'TorrentSelectionSettings';
  enabled: Scalars['Boolean']['output'];
  fastRules: Array<TorrentSelectionRule>;
  inspectionCandidateLimit: Scalars['Int']['output'];
  torrentRules: Array<TorrentSelectionRule>;
};

export type TorrentSelectionSettingsInput = {
  enabled: Scalars['Boolean']['input'];
  fastRules?: InputMaybe<Array<TorrentSelectionRuleInput>>;
  inspectionCandidateLimit: Scalars['Int']['input'];
  torrentRules?: InputMaybe<Array<TorrentSelectionRuleInput>>;
};

export type TransferIngestSettings = {
  __typename?: 'TransferIngestSettings';
  action: Scalars['String']['output'];
};

export type TransferIngestSettingsInput = {
  action: Scalars['String']['input'];
};

export type UpdateAutomationSettingsInput = {
  stashBoxEndpoints: Array<Scalars['String']['input']>;
  subscriptionPollIntervalHours: Scalars['Int']['input'];
  subscriptionReleasePolicy: SubscriptionReleasePolicyInput;
  taskProgressSyncIntervalSeconds: Scalars['Int']['input'];
  torrentSelection: TorrentSelectionSettingsInput;
};

export type UpdateIngestSettingsInput = {
  deliveryMode: Scalars['String']['input'];
  downloads: DownloadsIngestSettingsInput;
  library: LibraryIngestSettingsInput;
  transfer: TransferIngestSettingsInput;
};

export type UpdateJackettSettingsInput = {
  apiKey?: InputMaybe<Scalars['String']['input']>;
  /** Jackett dashboard password. Always sent by the UI and overwrites the stored value; pass an empty string to clear it. */
  password: Scalars['String']['input'];
  url: Scalars['String']['input'];
};

export type UpdateQBittorrentSettingsInput = {
  category: Scalars['String']['input'];
  defaultSavePath: Scalars['String']['input'];
  password?: InputMaybe<Scalars['String']['input']>;
  tags: Scalars['String']['input'];
  url: Scalars['String']['input'];
  username: Scalars['String']['input'];
};

export type UpdateStashSettingsInput = {
  apiKey?: InputMaybe<Scalars['String']['input']>;
  url: Scalars['String']['input'];
};

export type UpdateSystemSettingsInput = {
  taskDeletePolicy: TaskDeletePolicy;
};

export type DashboardDocumentQueryVariables = Exact<{ [key: string]: never; }>;


export type DashboardDocumentQuery = { __typename?: 'Query', version: string, health: { __typename?: 'Health', ok: boolean, message: string }, dashboardStats: { __typename?: 'DashboardStats', total: number, active: number, completed: number, downloading: number, pendingScans: number, failed: number }, settings: { __typename?: 'Settings', stash: { __typename?: 'StashSettings', configured: boolean, url: string, apiKeyConfigured: boolean, apiKey: string }, ingest: { __typename?: 'IngestSettings', deliveryMode: string, downloads: { __typename?: 'DownloadsIngestSettings', qbRoot: string, mojiRoot: string }, library: { __typename?: 'LibraryIngestSettings', mojiRoot: string, stashRoot: string }, transfer: { __typename?: 'TransferIngestSettings', action: string } }, jackett: { __typename?: 'JackettSettings', configured: boolean, url: string, apiKeyConfigured: boolean, apiKey: string, passwordConfigured: boolean, password: string }, qbittorrent: { __typename?: 'QBittorrentSettings', configured: boolean, url: string, username: string, usernameConfigured: boolean, passwordConfigured: boolean, password: string, defaultSavePath: string, category: string, tags: string }, automation: { __typename?: 'AutomationSettings', taskProgressSyncIntervalSeconds: number, subscriptionPollIntervalHours: number, stashBoxEndpoints: Array<string>, subscriptionReleasePolicy: { __typename?: 'SubscriptionReleasePolicy', soloBehavior: SubscriptionReleaseBehavior, groupBehavior: SubscriptionReleaseBehavior, compilationBehavior: SubscriptionReleaseBehavior, maxGroupPerformerCount: number, releaseDateRange: SubscriptionReleaseDateRange }, torrentSelection: { __typename?: 'TorrentSelectionSettings', enabled: boolean, inspectionCandidateLimit: number, fastRules: Array<{ __typename?: 'TorrentSelectionRule', type: TorrentSelectionRuleType, enabled: boolean, publishDate: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, seeders: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, size: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, indexerPreference: { __typename?: 'IndexerPreferenceRule', trackerIds: Array<string> }, titleMatch: { __typename?: 'TitleMatchRule', clauses: Array<{ __typename?: 'TitleMatchClause', pattern: string, patternMode: TitleMatchPatternMode, effect: TitleMatchEffect }> }, torrentFileNameMatch: { __typename?: 'TorrentFileNameMatchRule', clauses: Array<{ __typename?: 'TorrentFileNameMatchClause', pattern: string, patternMode: TitleMatchPatternMode, effect: TorrentFileMatchEffect }> } }>, torrentRules: Array<{ __typename?: 'TorrentSelectionRule', type: TorrentSelectionRuleType, enabled: boolean, publishDate: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, seeders: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, size: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, indexerPreference: { __typename?: 'IndexerPreferenceRule', trackerIds: Array<string> }, titleMatch: { __typename?: 'TitleMatchRule', clauses: Array<{ __typename?: 'TitleMatchClause', pattern: string, patternMode: TitleMatchPatternMode, effect: TitleMatchEffect }> }, torrentFileNameMatch: { __typename?: 'TorrentFileNameMatchRule', clauses: Array<{ __typename?: 'TorrentFileNameMatchClause', pattern: string, patternMode: TitleMatchPatternMode, effect: TorrentFileMatchEffect }> } }> } }, system: { __typename?: 'SystemSettings', taskDeletePolicy: TaskDeletePolicy } }, settingsStatus: { __typename?: 'SettingsStatus', stashLibrariesLoadError?: string | null, stash: { __typename?: 'ServiceStatus', configured: boolean, ready: boolean }, jackett: { __typename?: 'ServiceStatus', configured: boolean, ready: boolean }, qbittorrent: { __typename?: 'ServiceStatus', configured: boolean, ready: boolean }, automation: { __typename?: 'AutomationStatus', taskProgressSyncIntervalSeconds: number, taskProgressSyncEnabled: boolean, subscriptionPollIntervalHours: number, subscriptionPollEnabled: boolean }, subscription: { __typename?: 'SubscriptionStatus', stashBoxesLoaded: boolean, stashBoxesLoadError?: string | null, stashBoxes: Array<{ __typename?: 'StashBoxEndpoint', name: string, endpoint: string, apiKeyConfigured: boolean }> }, stashLibraries: Array<{ __typename?: 'StashLibrary', path: string }>, ingest: { __typename?: 'IngestStatus', configured: boolean }, stashStats: { __typename?: 'StashStats', version?: string | null, sceneCount?: number | null, pendingMojiScanCount: number, lastError?: string | null, okAt?: string | null }, jackettStats: { __typename?: 'JackettStats', indexerCount: number, configuredIndexerCount: number, lastIndexerLatencyMs: number, lastIndexerError?: string | null, lastIndexerSearchAt?: string | null, lastError?: string | null, okAt?: string | null }, qbittorrentStats: { __typename?: 'QBittorrentStats', downloadSpeed: number, uploadSpeed: number, activeTorrentCount: number, connectionStatus: string, altSpeedLimitEnabled: boolean, lastError?: string | null, okAt?: string | null } }, tasks: Array<{ __typename?: 'Task', id: string, source: TaskSource, query: string, code: string, status: string, torrentName: string, progress: number, qbittorrentState: string, contentPath: string, torrentHash: string, savePath: string, category: string, tags: string, error: string, completedAt?: string | null, stashMode: string, stashSourcePath: string, stashTransferAction: string, stashTransferPath: string, stashTransferStatus: string, stashTransferError: string, stashJobId: string, stashScanPath: string, stashScanStatus: string, stashScanError: string, stashScanHint: string, createdAt: string, updatedAt: string }> };

export type DiscoverScenesDocumentQueryVariables = Exact<{
  input: DiscoverScenesInput;
}>;


export type DiscoverScenesDocumentQuery = { __typename?: 'Query', discoverScenes: { __typename?: 'DiscoverSceneConnection', fallbackCount: number, searchedQuery: string, items: Array<{ __typename?: 'DiscoveredScene', key: string, sceneId: string, stashBoxEndpoint: string, stashBoxName: string, title: string, durationSeconds?: number | null, code?: string | null, date?: string | null, studioName?: string | null, imageUrl?: string | null, url?: string | null, performerNames: Array<string>, derivedQuery: string }>, usedStashBox?: { __typename?: 'MatchedStashBox', name: string, endpoint: string, performerId: string, performerName: string } | null } };

export type SearchDocumentQueryVariables = Exact<{
  input: JackettSearchInput;
}>;


export type SearchDocumentQuery = { __typename?: 'Query', jackettSearch: Array<{ __typename?: 'JackettSearchResult', title: string, size: any, seeders: number, peers: number, tracker: string, trackerId: string, categoryDesc: string, publishDate: string, details: string, link: string, magnetUri: string, infoHash: string }> };

export type PreviewJackettSelectionDocumentQueryVariables = Exact<{
  input: PreviewJackettSelectionInput;
}>;


export type PreviewJackettSelectionDocumentQuery = { __typename?: 'Query', previewJackettSelection: { __typename?: 'PreviewJackettSelectionResult', results: Array<{ __typename?: 'JackettSearchResult', title: string, size: any, seeders: number, peers: number, tracker: string, trackerId: string, categoryDesc: string, publishDate: string, details: string, link: string, magnetUri: string, infoHash: string }>, previewMeta: { __typename?: 'PreviewJackettSelectionMeta', appliedFastRules: boolean, appliedFileRules: boolean, inspectedCount: number, inspectableCount: number } } };

export type JackettIndexersDocumentQueryVariables = Exact<{ [key: string]: never; }>;


export type JackettIndexersDocumentQuery = { __typename?: 'Query', jackettIndexers: Array<{ __typename?: 'JackettIndexer', id: string, name: string, enabled: boolean }> };

export type QueueDiscoveredSceneDocumentMutationVariables = Exact<{
  input: QueueDiscoveredSceneInput;
}>;


export type QueueDiscoveredSceneDocumentMutation = { __typename?: 'Mutation', queueDiscoveredScene: { __typename?: 'Task', id: string, source: TaskSource, status: string, query: string, torrentName: string, progress: number, stashMode: string, stashScanStatus: string, createdAt: string } };

export type UpdateStashSettingsDocumentMutationVariables = Exact<{
  input: UpdateStashSettingsInput;
}>;


export type UpdateStashSettingsDocumentMutation = { __typename?: 'Mutation', updateStashSettings: { __typename?: 'Settings', stash: { __typename?: 'StashSettings', configured: boolean, url: string, apiKeyConfigured: boolean } } };

export type UpdateIngestSettingsDocumentMutationVariables = Exact<{
  input: UpdateIngestSettingsInput;
}>;


export type UpdateIngestSettingsDocumentMutation = { __typename?: 'Mutation', updateIngestSettings: { __typename?: 'Settings', ingest: { __typename?: 'IngestSettings', deliveryMode: string, downloads: { __typename?: 'DownloadsIngestSettings', qbRoot: string, mojiRoot: string }, library: { __typename?: 'LibraryIngestSettings', mojiRoot: string, stashRoot: string }, transfer: { __typename?: 'TransferIngestSettings', action: string } } } };

export type UpdateJackettSettingsDocumentMutationVariables = Exact<{
  input: UpdateJackettSettingsInput;
}>;


export type UpdateJackettSettingsDocumentMutation = { __typename?: 'Mutation', updateJackettSettings: { __typename?: 'Settings', jackett: { __typename?: 'JackettSettings', configured: boolean, url: string, apiKeyConfigured: boolean } } };

export type UpdateQBittorrentSettingsDocumentMutationVariables = Exact<{
  input: UpdateQBittorrentSettingsInput;
}>;


export type UpdateQBittorrentSettingsDocumentMutation = { __typename?: 'Mutation', updateQBittorrentSettings: { __typename?: 'Settings', qbittorrent: { __typename?: 'QBittorrentSettings', configured: boolean, url: string, username: string, usernameConfigured: boolean, passwordConfigured: boolean, defaultSavePath: string, category: string, tags: string } } };

export type UpdateAutomationSettingsDocumentMutationVariables = Exact<{
  input: UpdateAutomationSettingsInput;
}>;


export type UpdateAutomationSettingsDocumentMutation = { __typename?: 'Mutation', updateAutomationSettings: { __typename?: 'Settings', automation: { __typename?: 'AutomationSettings', taskProgressSyncIntervalSeconds: number, subscriptionPollIntervalHours: number, stashBoxEndpoints: Array<string>, subscriptionReleasePolicy: { __typename?: 'SubscriptionReleasePolicy', soloBehavior: SubscriptionReleaseBehavior, groupBehavior: SubscriptionReleaseBehavior, compilationBehavior: SubscriptionReleaseBehavior, maxGroupPerformerCount: number, releaseDateRange: SubscriptionReleaseDateRange }, torrentSelection: { __typename?: 'TorrentSelectionSettings', enabled: boolean, inspectionCandidateLimit: number, fastRules: Array<{ __typename?: 'TorrentSelectionRule', type: TorrentSelectionRuleType, enabled: boolean, publishDate: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, seeders: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, size: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, indexerPreference: { __typename?: 'IndexerPreferenceRule', trackerIds: Array<string> }, titleMatch: { __typename?: 'TitleMatchRule', clauses: Array<{ __typename?: 'TitleMatchClause', pattern: string, patternMode: TitleMatchPatternMode, effect: TitleMatchEffect }> }, torrentFileNameMatch: { __typename?: 'TorrentFileNameMatchRule', clauses: Array<{ __typename?: 'TorrentFileNameMatchClause', pattern: string, patternMode: TitleMatchPatternMode, effect: TorrentFileMatchEffect }> } }>, torrentRules: Array<{ __typename?: 'TorrentSelectionRule', type: TorrentSelectionRuleType, enabled: boolean, publishDate: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, seeders: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, size: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, indexerPreference: { __typename?: 'IndexerPreferenceRule', trackerIds: Array<string> }, titleMatch: { __typename?: 'TitleMatchRule', clauses: Array<{ __typename?: 'TitleMatchClause', pattern: string, patternMode: TitleMatchPatternMode, effect: TitleMatchEffect }> }, torrentFileNameMatch: { __typename?: 'TorrentFileNameMatchRule', clauses: Array<{ __typename?: 'TorrentFileNameMatchClause', pattern: string, patternMode: TitleMatchPatternMode, effect: TorrentFileMatchEffect }> } }> } } } };

export type UpdateSystemSettingsDocumentMutationVariables = Exact<{
  input: UpdateSystemSettingsInput;
}>;


export type UpdateSystemSettingsDocumentMutation = { __typename?: 'Mutation', updateSystemSettings: { __typename?: 'Settings', system: { __typename?: 'SystemSettings', taskDeletePolicy: TaskDeletePolicy } } };

export type RefreshSubscriptionStashBoxesDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type RefreshSubscriptionStashBoxesDocumentMutation = { __typename?: 'Mutation', refreshSubscriptionStashBoxes: { __typename?: 'Settings', automation: { __typename?: 'AutomationSettings', stashBoxEndpoints: Array<string> } } };

export type LogsDocumentQueryVariables = Exact<{
  limit?: InputMaybe<Scalars['Int']['input']>;
  minLevel?: InputMaybe<LogLevel>;
}>;


export type LogsDocumentQuery = { __typename?: 'Query', logs: Array<{ __typename?: 'LogEntry', time: string, level: LogLevel, message: string }> };

export type StashPerformersQueryVariables = Exact<{
  search?: InputMaybe<Scalars['String']['input']>;
  page?: InputMaybe<Scalars['Int']['input']>;
  pageSize?: InputMaybe<Scalars['Int']['input']>;
}>;


export type StashPerformersQuery = { __typename?: 'Query', stashPerformers: { __typename?: 'StashPerformerConnection', page: number, pageSize: number, totalCount: number, totalPages: number, hasPrevPage: boolean, hasNextPage: boolean, items: Array<{ __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }> } };

export type StashPerformerDetailQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type StashPerformerDetailQuery = { __typename?: 'Query', stashPerformerDetail: { __typename?: 'StashPerformerDetail', disambiguation?: string | null, birthdate?: string | null, ethnicity?: string | null, country?: string | null, eyeColor?: string | null, heightCm?: number | null, rating100?: number | null, urls: Array<string>, totalSceneCount: number, stashSceneCount: number, stashBoxSceneCount: number, dedupedSceneCount: number, performer: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }, matchedStashBox?: { __typename?: 'MatchedStashBox', name: string, endpoint: string, performerId: string, performerName: string } | null } };

export type StashPerformerScenesQueryVariables = Exact<{
  id: Scalars['ID']['input'];
  input: StashPerformerScenesInput;
}>;


export type StashPerformerScenesQuery = { __typename?: 'Query', stashPerformerScenes: { __typename?: 'StashPerformerSceneConnection', page: number, pageSize: number, totalCount: number, totalPages: number, hasPrevPage: boolean, hasNextPage: boolean, stashSceneCount: number, stashBoxCount: number, dedupedCount: number, items: Array<{ __typename?: 'StashPerformerScene', key: string, primarySource: SceneSource, sourceSceneId: string, title?: string | null, code?: string | null, date?: string | null, studioName?: string | null, imageUrl?: string | null, url?: string | null, inLibrary: boolean, matchedStashSceneId?: string | null, hasStashSource: boolean, hasStashBoxSource: boolean, stashBoxSceneId?: string | null, stashBoxEndpoint?: string | null, sourceLabels: Array<string>, stashIds: Array<{ __typename?: 'StashSceneID', endpoint: string, stashId: string }> }> } };

export type SubscribedPerformersQueryVariables = Exact<{ [key: string]: never; }>;


export type SubscribedPerformersQuery = { __typename?: 'Query', subscribedPerformers: Array<{ __typename?: 'SubscribedPerformer', lastCheckedAt?: string | null, lastError?: string | null, pendingReleaseCount: number, processedReleaseCount: number, performer: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }, recentReleases: Array<{ __typename?: 'SubscriptionRelease', key: string, source: string, title: string, code?: string | null, date?: string | null, url?: string | null, query: string, taskID?: string | null, performerCount: number, performerNames: Array<string>, classification: SubscriptionReleaseClassification, decision: SubscriptionReleaseDecision, decisionReason: string, seenAt: string }> }> };

export type SubscribePerformerMutationVariables = Exact<{
  stashPerformerID: Scalars['ID']['input'];
}>;


export type SubscribePerformerMutation = { __typename?: 'Mutation', subscribePerformer: { __typename?: 'SubscribedPerformer', performer: { __typename?: 'StashPerformer', id: string, subscribed: boolean } } };

export type UnsubscribePerformerMutationVariables = Exact<{
  stashPerformerID: Scalars['ID']['input'];
}>;


export type UnsubscribePerformerMutation = { __typename?: 'Mutation', unsubscribePerformer: boolean };

export type RefreshSubscribedPerformerMutationVariables = Exact<{
  stashPerformerID: Scalars['ID']['input'];
}>;


export type RefreshSubscribedPerformerMutation = { __typename?: 'Mutation', refreshSubscribedPerformer: { __typename?: 'SubscribedPerformer', lastCheckedAt?: string | null, lastError?: string | null, pendingReleaseCount: number, processedReleaseCount: number, performer: { __typename?: 'StashPerformer', id: string, subscribed: boolean }, recentReleases: Array<{ __typename?: 'SubscriptionRelease', key: string, title: string, code?: string | null, date?: string | null, query: string, taskID?: string | null, performerCount: number, performerNames: Array<string>, classification: SubscriptionReleaseClassification, decision: SubscriptionReleaseDecision, decisionReason: string, seenAt: string }> } };

export type RefreshSubscriptionNowMutationVariables = Exact<{ [key: string]: never; }>;


export type RefreshSubscriptionNowMutation = { __typename?: 'Mutation', refreshSubscriptionsNow: Array<{ __typename?: 'SubscribedPerformer', performer: { __typename?: 'StashPerformer', id: string } }> };

export type AddTorrentDocumentMutationVariables = Exact<{
  input: QBittorrentAddInput;
}>;


export type AddTorrentDocumentMutation = { __typename?: 'Mutation', addTorrent: { __typename?: 'Task', id: string, source: TaskSource, status: string, query: string, code: string, torrentName: string, progress: number, stashMode: string, stashScanStatus: string, createdAt: string } };

export type SyncTaskProgressDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type SyncTaskProgressDocumentMutation = { __typename?: 'Mutation', syncTaskProgress: Array<{ __typename?: 'Task', id: string, source: TaskSource, status: string, progress: number, qbittorrentState: string, updatedAt: string }> };

export type TriggerStashScansDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type TriggerStashScansDocumentMutation = { __typename?: 'Mutation', triggerStashScans: Array<{ __typename?: 'Task', id: string, source: TaskSource, stashMode: string, stashTransferStatus: string, stashTransferError: string, stashJobId: string, stashScanPath: string, stashScanStatus: string, stashScanError: string, stashScanHint: string, updatedAt: string }> };

export type TriggerTaskStashScanDocumentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type TriggerTaskStashScanDocumentMutation = { __typename?: 'Mutation', triggerTaskStashScan: { __typename?: 'Task', id: string, source: TaskSource, stashMode: string, stashTransferStatus: string, stashTransferError: string, stashJobId: string, stashScanPath: string, stashScanStatus: string, stashScanError: string, stashScanHint: string, updatedAt: string } };

export type DeleteTaskDocumentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteTaskDocumentMutation = { __typename?: 'Mutation', deleteTask: { __typename?: 'Task', id: string, source: TaskSource, status: string, query: string, code: string, updatedAt: string } };


export const DashboardDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DashboardDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"dashboardStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"active"}},{"kind":"Field","name":{"kind":"Name","value":"completed"}},{"kind":"Field","name":{"kind":"Name","value":"downloading"}},{"kind":"Field","name":{"kind":"Name","value":"pendingScans"}},{"kind":"Field","name":{"kind":"Name","value":"failed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"apiKey"}}]}},{"kind":"Field","name":{"kind":"Name","value":"ingest"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deliveryMode"}},{"kind":"Field","name":{"kind":"Name","value":"downloads"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"qbRoot"}},{"kind":"Field","name":{"kind":"Name","value":"mojiRoot"}}]}},{"kind":"Field","name":{"kind":"Name","value":"library"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"mojiRoot"}},{"kind":"Field","name":{"kind":"Name","value":"stashRoot"}}]}},{"kind":"Field","name":{"kind":"Name","value":"transfer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"action"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"apiKey"}},{"kind":"Field","name":{"kind":"Name","value":"passwordConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"password"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"username"}},{"kind":"Field","name":{"kind":"Name","value":"usernameConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"passwordConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"password"}},{"kind":"Field","name":{"kind":"Name","value":"defaultSavePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}}]}},{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionPollIntervalHours"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxEndpoints"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionReleasePolicy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"soloBehavior"}},{"kind":"Field","name":{"kind":"Name","value":"groupBehavior"}},{"kind":"Field","name":{"kind":"Name","value":"compilationBehavior"}},{"kind":"Field","name":{"kind":"Name","value":"maxGroupPerformerCount"}},{"kind":"Field","name":{"kind":"Name","value":"releaseDateRange"}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentSelection"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"inspectionCandidateLimit"}},{"kind":"Field","name":{"kind":"Name","value":"fastRules"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"publishDate"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"seeders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"size"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"indexerPreference"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"trackerIds"}}]}},{"kind":"Field","name":{"kind":"Name","value":"titleMatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clauses"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pattern"}},{"kind":"Field","name":{"kind":"Name","value":"patternMode"}},{"kind":"Field","name":{"kind":"Name","value":"effect"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentFileNameMatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clauses"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pattern"}},{"kind":"Field","name":{"kind":"Name","value":"patternMode"}},{"kind":"Field","name":{"kind":"Name","value":"effect"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentRules"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"publishDate"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"seeders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"size"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"indexerPreference"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"trackerIds"}}]}},{"kind":"Field","name":{"kind":"Name","value":"titleMatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clauses"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pattern"}},{"kind":"Field","name":{"kind":"Name","value":"patternMode"}},{"kind":"Field","name":{"kind":"Name","value":"effect"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentFileNameMatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clauses"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pattern"}},{"kind":"Field","name":{"kind":"Name","value":"patternMode"}},{"kind":"Field","name":{"kind":"Name","value":"effect"}}]}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"system"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskDeletePolicy"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"settingsStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"ready"}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"ready"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"ready"}}]}},{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionPollIntervalHours"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionPollEnabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"subscription"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashBoxes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxesLoaded"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxesLoadError"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stashLibraries"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"path"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stashLibrariesLoadError"}},{"kind":"Field","name":{"kind":"Name","value":"ingest"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stashStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingMojiScanCount"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"okAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackettStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerCount"}},{"kind":"Field","name":{"kind":"Name","value":"configuredIndexerCount"}},{"kind":"Field","name":{"kind":"Name","value":"lastIndexerLatencyMs"}},{"kind":"Field","name":{"kind":"Name","value":"lastIndexerError"}},{"kind":"Field","name":{"kind":"Name","value":"lastIndexerSearchAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"okAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"downloadSpeed"}},{"kind":"Field","name":{"kind":"Name","value":"uploadSpeed"}},{"kind":"Field","name":{"kind":"Name","value":"activeTorrentCount"}},{"kind":"Field","name":{"kind":"Name","value":"connectionStatus"}},{"kind":"Field","name":{"kind":"Name","value":"altSpeedLimitEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"okAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"tasks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"contentPath"}},{"kind":"Field","name":{"kind":"Name","value":"torrentHash"}},{"kind":"Field","name":{"kind":"Name","value":"savePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"completedAt"}},{"kind":"Field","name":{"kind":"Name","value":"stashMode"}},{"kind":"Field","name":{"kind":"Name","value":"stashSourcePath"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferAction"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferError"}},{"kind":"Field","name":{"kind":"Name","value":"stashJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanHint"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<DashboardDocumentQuery, DashboardDocumentQueryVariables>;
export const DiscoverScenesDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DiscoverScenesDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"DiscoverScenesInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"discoverScenes"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"sceneId"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxEndpoint"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxName"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"durationSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"studioName"}},{"kind":"Field","name":{"kind":"Name","value":"imageUrl"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"performerNames"}},{"kind":"Field","name":{"kind":"Name","value":"derivedQuery"}}]}},{"kind":"Field","name":{"kind":"Name","value":"usedStashBox"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"performerName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fallbackCount"}},{"kind":"Field","name":{"kind":"Name","value":"searchedQuery"}}]}}]}}]} as unknown as DocumentNode<DiscoverScenesDocumentQuery, DiscoverScenesDocumentQueryVariables>;
export const SearchDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SearchDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"JackettSearchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jackettSearch"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}},{"kind":"Field","name":{"kind":"Name","value":"tracker"}},{"kind":"Field","name":{"kind":"Name","value":"trackerId"}},{"kind":"Field","name":{"kind":"Name","value":"categoryDesc"}},{"kind":"Field","name":{"kind":"Name","value":"publishDate"}},{"kind":"Field","name":{"kind":"Name","value":"details"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"magnetUri"}},{"kind":"Field","name":{"kind":"Name","value":"infoHash"}}]}}]}}]} as unknown as DocumentNode<SearchDocumentQuery, SearchDocumentQueryVariables>;
export const PreviewJackettSelectionDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"PreviewJackettSelectionDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"PreviewJackettSelectionInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"previewJackettSelection"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}},{"kind":"Field","name":{"kind":"Name","value":"tracker"}},{"kind":"Field","name":{"kind":"Name","value":"trackerId"}},{"kind":"Field","name":{"kind":"Name","value":"categoryDesc"}},{"kind":"Field","name":{"kind":"Name","value":"publishDate"}},{"kind":"Field","name":{"kind":"Name","value":"details"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"magnetUri"}},{"kind":"Field","name":{"kind":"Name","value":"infoHash"}}]}},{"kind":"Field","name":{"kind":"Name","value":"previewMeta"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"appliedFastRules"}},{"kind":"Field","name":{"kind":"Name","value":"appliedFileRules"}},{"kind":"Field","name":{"kind":"Name","value":"inspectedCount"}},{"kind":"Field","name":{"kind":"Name","value":"inspectableCount"}}]}}]}}]}}]} as unknown as DocumentNode<PreviewJackettSelectionDocumentQuery, PreviewJackettSelectionDocumentQueryVariables>;
export const JackettIndexersDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"JackettIndexersDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jackettIndexers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}}]}}]} as unknown as DocumentNode<JackettIndexersDocumentQuery, JackettIndexersDocumentQueryVariables>;
export const QueueDiscoveredSceneDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"QueueDiscoveredSceneDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"QueueDiscoveredSceneInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"queueDiscoveredScene"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"stashMode"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<QueueDiscoveredSceneDocumentMutation, QueueDiscoveredSceneDocumentMutationVariables>;
export const UpdateStashSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateStashSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateStashSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateStashSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateStashSettingsDocumentMutation, UpdateStashSettingsDocumentMutationVariables>;
export const UpdateIngestSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateIngestSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateIngestSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateIngestSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ingest"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deliveryMode"}},{"kind":"Field","name":{"kind":"Name","value":"downloads"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"qbRoot"}},{"kind":"Field","name":{"kind":"Name","value":"mojiRoot"}}]}},{"kind":"Field","name":{"kind":"Name","value":"library"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"mojiRoot"}},{"kind":"Field","name":{"kind":"Name","value":"stashRoot"}}]}},{"kind":"Field","name":{"kind":"Name","value":"transfer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"action"}}]}}]}}]}}]}}]} as unknown as DocumentNode<UpdateIngestSettingsDocumentMutation, UpdateIngestSettingsDocumentMutationVariables>;
export const UpdateJackettSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateJackettSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateJackettSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateJackettSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateJackettSettingsDocumentMutation, UpdateJackettSettingsDocumentMutationVariables>;
export const UpdateQBittorrentSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateQBittorrentSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateQBittorrentSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateQBittorrentSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"username"}},{"kind":"Field","name":{"kind":"Name","value":"usernameConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"passwordConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"defaultSavePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateQBittorrentSettingsDocumentMutation, UpdateQBittorrentSettingsDocumentMutationVariables>;
export const UpdateAutomationSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateAutomationSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateAutomationSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateAutomationSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionPollIntervalHours"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxEndpoints"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionReleasePolicy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"soloBehavior"}},{"kind":"Field","name":{"kind":"Name","value":"groupBehavior"}},{"kind":"Field","name":{"kind":"Name","value":"compilationBehavior"}},{"kind":"Field","name":{"kind":"Name","value":"maxGroupPerformerCount"}},{"kind":"Field","name":{"kind":"Name","value":"releaseDateRange"}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentSelection"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"inspectionCandidateLimit"}},{"kind":"Field","name":{"kind":"Name","value":"fastRules"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"publishDate"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"seeders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"size"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"indexerPreference"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"trackerIds"}}]}},{"kind":"Field","name":{"kind":"Name","value":"titleMatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clauses"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pattern"}},{"kind":"Field","name":{"kind":"Name","value":"patternMode"}},{"kind":"Field","name":{"kind":"Name","value":"effect"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentFileNameMatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clauses"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pattern"}},{"kind":"Field","name":{"kind":"Name","value":"patternMode"}},{"kind":"Field","name":{"kind":"Name","value":"effect"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentRules"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"publishDate"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"seeders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"size"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"indexerPreference"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"trackerIds"}}]}},{"kind":"Field","name":{"kind":"Name","value":"titleMatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clauses"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pattern"}},{"kind":"Field","name":{"kind":"Name","value":"patternMode"}},{"kind":"Field","name":{"kind":"Name","value":"effect"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentFileNameMatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clauses"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pattern"}},{"kind":"Field","name":{"kind":"Name","value":"patternMode"}},{"kind":"Field","name":{"kind":"Name","value":"effect"}}]}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<UpdateAutomationSettingsDocumentMutation, UpdateAutomationSettingsDocumentMutationVariables>;
export const UpdateSystemSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateSystemSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateSystemSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateSystemSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"system"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskDeletePolicy"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateSystemSettingsDocumentMutation, UpdateSystemSettingsDocumentMutationVariables>;
export const RefreshSubscriptionStashBoxesDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshSubscriptionStashBoxesDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshSubscriptionStashBoxes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashBoxEndpoints"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshSubscriptionStashBoxesDocumentMutation, RefreshSubscriptionStashBoxesDocumentMutationVariables>;
export const LogsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"LogsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"minLevel"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"LogLevel"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"logs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}},{"kind":"Argument","name":{"kind":"Name","value":"minLevel"},"value":{"kind":"Variable","name":{"kind":"Name","value":"minLevel"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"level"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<LogsDocumentQuery, LogsDocumentQueryVariables>;
export const StashPerformersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"StashPerformers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"search"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashPerformers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"search"},"value":{"kind":"Variable","name":{"kind":"Name","value":"search"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}},{"kind":"Argument","name":{"kind":"Name","value":"pageSize"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"pageSize"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"hasPrevPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}}]}}]}}]} as unknown as DocumentNode<StashPerformersQuery, StashPerformersQueryVariables>;
export const StashPerformerDetailDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"StashPerformerDetail"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashPerformerDetail"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"disambiguation"}},{"kind":"Field","name":{"kind":"Name","value":"birthdate"}},{"kind":"Field","name":{"kind":"Name","value":"ethnicity"}},{"kind":"Field","name":{"kind":"Name","value":"country"}},{"kind":"Field","name":{"kind":"Name","value":"eyeColor"}},{"kind":"Field","name":{"kind":"Name","value":"heightCm"}},{"kind":"Field","name":{"kind":"Name","value":"rating100"}},{"kind":"Field","name":{"kind":"Name","value":"urls"}},{"kind":"Field","name":{"kind":"Name","value":"matchedStashBox"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"performerName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalSceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxSceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"dedupedSceneCount"}}]}}]}}]} as unknown as DocumentNode<StashPerformerDetailQuery, StashPerformerDetailQueryVariables>;
export const StashPerformerScenesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"StashPerformerScenes"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"StashPerformerScenesInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashPerformerScenes"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"primarySource"}},{"kind":"Field","name":{"kind":"Name","value":"sourceSceneId"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"studioName"}},{"kind":"Field","name":{"kind":"Name","value":"imageUrl"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"inLibrary"}},{"kind":"Field","name":{"kind":"Name","value":"matchedStashSceneId"}},{"kind":"Field","name":{"kind":"Name","value":"hasStashSource"}},{"kind":"Field","name":{"kind":"Name","value":"hasStashBoxSource"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxSceneId"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxEndpoint"}},{"kind":"Field","name":{"kind":"Name","value":"sourceLabels"}},{"kind":"Field","name":{"kind":"Name","value":"stashIds"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"stashId"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"pageSize"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"hasPrevPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxCount"}},{"kind":"Field","name":{"kind":"Name","value":"dedupedCount"}}]}}]}}]} as unknown as DocumentNode<StashPerformerScenesQuery, StashPerformerScenesQueryVariables>;
export const SubscribedPerformersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SubscribedPerformers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscribedPerformers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"taskID"}},{"kind":"Field","name":{"kind":"Name","value":"performerCount"}},{"kind":"Field","name":{"kind":"Name","value":"performerNames"}},{"kind":"Field","name":{"kind":"Name","value":"classification"}},{"kind":"Field","name":{"kind":"Name","value":"decision"}},{"kind":"Field","name":{"kind":"Name","value":"decisionReason"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]} as unknown as DocumentNode<SubscribedPerformersQuery, SubscribedPerformersQueryVariables>;
export const SubscribePerformerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SubscribePerformer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscribePerformer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"stashPerformerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}}]}}]}}]} as unknown as DocumentNode<SubscribePerformerMutation, SubscribePerformerMutationVariables>;
export const UnsubscribePerformerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnsubscribePerformer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unsubscribePerformer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"stashPerformerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}}}]}]}}]} as unknown as DocumentNode<UnsubscribePerformerMutation, UnsubscribePerformerMutationVariables>;
export const RefreshSubscribedPerformerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshSubscribedPerformer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshSubscribedPerformer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"stashPerformerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"taskID"}},{"kind":"Field","name":{"kind":"Name","value":"performerCount"}},{"kind":"Field","name":{"kind":"Name","value":"performerNames"}},{"kind":"Field","name":{"kind":"Name","value":"classification"}},{"kind":"Field","name":{"kind":"Name","value":"decision"}},{"kind":"Field","name":{"kind":"Name","value":"decisionReason"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshSubscribedPerformerMutation, RefreshSubscribedPerformerMutationVariables>;
export const RefreshSubscriptionNowDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshSubscriptionNow"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshSubscriptionsNow"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshSubscriptionNowMutation, RefreshSubscriptionNowMutationVariables>;
export const AddTorrentDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddTorrentDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"QBittorrentAddInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"addTorrent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"stashMode"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<AddTorrentDocumentMutation, AddTorrentDocumentMutationVariables>;
export const SyncTaskProgressDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SyncTaskProgressDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"syncTaskProgress"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<SyncTaskProgressDocumentMutation, SyncTaskProgressDocumentMutationVariables>;
export const TriggerStashScansDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TriggerStashScansDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggerStashScans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"stashMode"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferError"}},{"kind":"Field","name":{"kind":"Name","value":"stashJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanHint"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<TriggerStashScansDocumentMutation, TriggerStashScansDocumentMutationVariables>;
export const TriggerTaskStashScanDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TriggerTaskStashScanDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggerTaskStashScan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"stashMode"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferError"}},{"kind":"Field","name":{"kind":"Name","value":"stashJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanHint"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<TriggerTaskStashScanDocumentMutation, TriggerTaskStashScanDocumentMutationVariables>;
export const DeleteTaskDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteTaskDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteTask"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<DeleteTaskDocumentMutation, DeleteTaskDocumentMutationVariables>;