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
  subscriptionPollIntervalSeconds: Scalars['Int']['output'];
  taskProgressSyncIntervalSeconds: Scalars['Int']['output'];
};

export type AutomationStatus = {
  __typename?: 'AutomationStatus';
  subscriptionPollEnabled: Scalars['Boolean']['output'];
  subscriptionPollIntervalSeconds: Scalars['Int']['output'];
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

/** Basic service health */
export type Health = {
  __typename?: 'Health';
  message: Scalars['String']['output'];
  ok: Scalars['Boolean']['output'];
};

export type JackettSearchInput = {
  categories?: InputMaybe<Array<Scalars['Int']['input']>>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  query: Scalars['String']['input'];
  trackers?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type JackettSearchResult = {
  __typename?: 'JackettSearchResult';
  categoryDesc: Scalars['String']['output'];
  details: Scalars['String']['output'];
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
  enabled: Scalars['Boolean']['output'];
  /** Currently configured Jackett dashboard password. Returned in plaintext for the settings UI; never logged. */
  password: Scalars['String']['output'];
  passwordConfigured: Scalars['Boolean']['output'];
  url: Scalars['String']['output'];
};

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
  /** ISO 8601 timestamp of the most recent successful refresh. */
  okAt: Scalars['String']['output'];
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

export type Mutation = {
  __typename?: 'Mutation';
  /** Add a torrent URL or magnet and create a persisted Moji task */
  addTorrent: Task;
  /** Search torrent candidates and create a Moji download task */
  downloadMedia: Task;
  /**
   * Add a torrent to qBittorrent via magnet or http(s) URL
   * @deprecated Use addTorrent for persisted Moji task tracking.
   */
  qbittorrentAdd: Scalars['Boolean']['output'];
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
  /** Update Jackett settings and persist them to backend config */
  updateJackettSettings: Settings;
  /** Update qBittorrent settings and persist them to backend config */
  updateQBittorrentSettings: Settings;
  /** Update Stash settings and persist them to backend config */
  updateStashSettings: Settings;
  /** Update Subscription settings and persist them to backend config */
  updateSubscriptionSettings: Settings;
};


export type MutationAddTorrentArgs = {
  input: QBittorrentAddInput;
};


export type MutationDownloadMediaArgs = {
  input: DownloadMediaInput;
};


export type MutationQbittorrentAddArgs = {
  input: QBittorrentAddInput;
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


export type MutationUpdateJackettSettingsArgs = {
  input: UpdateJackettSettingsInput;
};


export type MutationUpdateQBittorrentSettingsArgs = {
  input: UpdateQBittorrentSettingsInput;
};


export type MutationUpdateStashSettingsArgs = {
  input: UpdateStashSettingsInput;
};


export type MutationUpdateSubscriptionSettingsArgs = {
  input: UpdateSubscriptionSettingsInput;
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
  enabled: Scalars['Boolean']['output'];
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
  /** ISO 8601 timestamp of the most recent successful refresh. */
  okAt: Scalars['String']['output'];
  /** Global upload rate in bytes/sec. */
  uploadSpeed: Scalars['Int']['output'];
};

export type Query = {
  __typename?: 'Query';
  /** Get aggregate task stats for the dashboard and task center */
  dashboardStats: DashboardStats;
  health: Health;
  /** Search torrents via Jackett */
  jackettSearch: Array<JackettSearchResult>;
  /** Retrieve recent Moji logs for troubleshooting */
  logs: Array<LogEntry>;
  /** List torrents from qBittorrent */
  qbittorrentTorrents: Array<QbTorrent>;
  /** Get editable configuration for the Settings surface */
  settings: Settings;
  /** Get runtime state for the Settings surface */
  settingsStatus: SettingsStatus;
  /** Get a Stash background job by id */
  stashJob?: Maybe<StashJob>;
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


export type QueryJackettSearchArgs = {
  input: JackettSearchInput;
};


export type QueryLogsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  minLevel?: InputMaybe<LogLevel>;
};


export type QueryQbittorrentTorrentsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
};


export type QueryStashJobArgs = {
  id: Scalars['ID']['input'];
};


export type QueryStashPerformersArgs = {
  page?: InputMaybe<Scalars['Int']['input']>;
  pageSize?: InputMaybe<Scalars['Int']['input']>;
  search?: InputMaybe<Scalars['String']['input']>;
};


export type QueryTaskArgs = {
  id: Scalars['ID']['input'];
};

export type ServiceStatus = {
  __typename?: 'ServiceStatus';
  configured: Scalars['Boolean']['output'];
  enabled: Scalars['Boolean']['output'];
};

export type Settings = {
  __typename?: 'Settings';
  automation: AutomationSettings;
  jackett: JackettSettings;
  qbittorrent: QBittorrentSettings;
  stash: StashSettings;
  subscription: SubscriptionSettings;
};

export type SettingsStatus = {
  __typename?: 'SettingsStatus';
  automation: AutomationStatus;
  jackett: ServiceStatus;
  /** Runtime stats for the Jackett indexer aggregator. Refreshed by the stats collector. */
  jackettStats: JackettStats;
  qbittorrent: ServiceStatus;
  /** Runtime stats for the qBittorrent download client. Refreshed by the stats collector. */
  qbittorrentStats: QBittorrentStats;
  stash: ServiceStatus;
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

export type StashSettings = {
  __typename?: 'StashSettings';
  /** Currently configured Stash API key. Returned in plaintext for the settings UI; never logged. */
  apiKey: Scalars['String']['output'];
  apiKeyConfigured: Scalars['Boolean']['output'];
  configured: Scalars['Boolean']['output'];
  enabled: Scalars['Boolean']['output'];
  libraryPath: Scalars['String']['output'];
  mode: Scalars['String']['output'];
  qbittorrentPathPrefix: Scalars['String']['output'];
  stashPathPrefix: Scalars['String']['output'];
  transferAction: Scalars['String']['output'];
  transferTargetPath: Scalars['String']['output'];
  url: Scalars['String']['output'];
};

/** Per-service runtime stats. okAt is the timestamp of the most recent successful refresh; lastError is the message from the most recent failed refresh (if any). When lastError is non-null, other numeric fields still reflect the last known-good snapshot. */
export type StashStats = {
  __typename?: 'StashStats';
  /** Most recent error message from any Stash-side refresh. Null = OK. */
  lastError?: Maybe<Scalars['String']['output']>;
  /** ISO 8601 timestamp of the most recent successful refresh. */
  okAt: Scalars['String']['output'];
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
  code?: Maybe<Scalars['String']['output']>;
  date?: Maybe<Scalars['String']['output']>;
  key: Scalars['ID']['output'];
  query: Scalars['String']['output'];
  seenAt: Scalars['String']['output'];
  source: Scalars['String']['output'];
  taskID?: Maybe<Scalars['ID']['output']>;
  title: Scalars['String']['output'];
  url?: Maybe<Scalars['String']['output']>;
};

export type SubscriptionSettings = {
  __typename?: 'SubscriptionSettings';
  /** Endpoint URLs in the user-defined order used for subscription lookups. Endpoints not listed here are still queried, in their Stash order, appended after the listed ones. An empty list means use Stash's order as-is. */
  stashBoxEndpoints: Array<Scalars['String']['output']>;
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

export type Task = {
  __typename?: 'Task';
  candidate: DownloadCandidate;
  category: Scalars['String']['output'];
  completedAt?: Maybe<Scalars['String']['output']>;
  contentPath: Scalars['String']['output'];
  createdAt: Scalars['String']['output'];
  error: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  progress: Scalars['Float']['output'];
  qbittorrentState: Scalars['String']['output'];
  query: Scalars['String']['output'];
  savePath: Scalars['String']['output'];
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

export type UpdateAutomationSettingsInput = {
  subscriptionPollIntervalSeconds: Scalars['Int']['input'];
  taskProgressSyncIntervalSeconds: Scalars['Int']['input'];
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
  libraryPath: Scalars['String']['input'];
  mode: Scalars['String']['input'];
  qbittorrentPathPrefix: Scalars['String']['input'];
  stashPathPrefix: Scalars['String']['input'];
  transferAction: Scalars['String']['input'];
  transferTargetPath: Scalars['String']['input'];
  url: Scalars['String']['input'];
};

export type UpdateSubscriptionSettingsInput = {
  /** See SubscriptionSettings.stashBoxEndpoints. */
  stashBoxEndpoints: Array<Scalars['String']['input']>;
};

export type DashboardDocumentQueryVariables = Exact<{ [key: string]: never; }>;


export type DashboardDocumentQuery = { __typename?: 'Query', version: string, health: { __typename?: 'Health', ok: boolean, message: string }, dashboardStats: { __typename?: 'DashboardStats', total: number, active: number, completed: number, downloading: number, pendingScans: number, failed: number }, settings: { __typename?: 'Settings', stash: { __typename?: 'StashSettings', configured: boolean, enabled: boolean, url: string, apiKeyConfigured: boolean, apiKey: string, mode: string, libraryPath: string, qbittorrentPathPrefix: string, stashPathPrefix: string, transferAction: string, transferTargetPath: string }, jackett: { __typename?: 'JackettSettings', configured: boolean, enabled: boolean, url: string, apiKeyConfigured: boolean, apiKey: string, passwordConfigured: boolean, password: string }, qbittorrent: { __typename?: 'QBittorrentSettings', configured: boolean, enabled: boolean, url: string, username: string, usernameConfigured: boolean, passwordConfigured: boolean, password: string, defaultSavePath: string, category: string, tags: string }, automation: { __typename?: 'AutomationSettings', taskProgressSyncIntervalSeconds: number, subscriptionPollIntervalSeconds: number }, subscription: { __typename?: 'SubscriptionSettings', stashBoxEndpoints: Array<string> } }, settingsStatus: { __typename?: 'SettingsStatus', stash: { __typename?: 'ServiceStatus', configured: boolean, enabled: boolean }, jackett: { __typename?: 'ServiceStatus', configured: boolean, enabled: boolean }, qbittorrent: { __typename?: 'ServiceStatus', configured: boolean, enabled: boolean }, automation: { __typename?: 'AutomationStatus', taskProgressSyncIntervalSeconds: number, taskProgressSyncEnabled: boolean, subscriptionPollIntervalSeconds: number, subscriptionPollEnabled: boolean }, subscription: { __typename?: 'SubscriptionStatus', stashBoxesLoaded: boolean, stashBoxesLoadError?: string | null, stashBoxes: Array<{ __typename?: 'StashBoxEndpoint', name: string, endpoint: string, apiKeyConfigured: boolean }> }, stashStats: { __typename?: 'StashStats', version?: string | null, sceneCount?: number | null, pendingMojiScanCount: number, lastError?: string | null, okAt: string }, jackettStats: { __typename?: 'JackettStats', indexerCount: number, configuredIndexerCount: number, lastIndexerLatencyMs: number, lastIndexerError?: string | null, lastIndexerSearchAt?: string | null, lastError?: string | null, okAt: string }, qbittorrentStats: { __typename?: 'QBittorrentStats', downloadSpeed: number, uploadSpeed: number, activeTorrentCount: number, connectionStatus: string, altSpeedLimitEnabled: boolean, lastError?: string | null, okAt: string } }, tasks: Array<{ __typename?: 'Task', id: string, query: string, status: string, torrentName: string, progress: number, qbittorrentState: string, contentPath: string, torrentHash: string, savePath: string, category: string, tags: string, error: string, completedAt?: string | null, stashMode: string, stashSourcePath: string, stashTransferAction: string, stashTransferPath: string, stashTransferStatus: string, stashTransferError: string, stashJobId: string, stashScanPath: string, stashScanStatus: string, stashScanError: string, stashScanHint: string, createdAt: string, updatedAt: string }> };

export type SearchDocumentQueryVariables = Exact<{
  input: JackettSearchInput;
}>;


export type SearchDocumentQuery = { __typename?: 'Query', jackettSearch: Array<{ __typename?: 'JackettSearchResult', title: string, size: any, seeders: number, peers: number, tracker: string, categoryDesc: string, publishDate: string, link: string, magnetUri: string }> };

export type UpdateStashSettingsDocumentMutationVariables = Exact<{
  input: UpdateStashSettingsInput;
}>;


export type UpdateStashSettingsDocumentMutation = { __typename?: 'Mutation', updateStashSettings: { __typename?: 'Settings', stash: { __typename?: 'StashSettings', configured: boolean, enabled: boolean, url: string, apiKeyConfigured: boolean, mode: string, libraryPath: string, qbittorrentPathPrefix: string, stashPathPrefix: string, transferAction: string, transferTargetPath: string } } };

export type UpdateJackettSettingsDocumentMutationVariables = Exact<{
  input: UpdateJackettSettingsInput;
}>;


export type UpdateJackettSettingsDocumentMutation = { __typename?: 'Mutation', updateJackettSettings: { __typename?: 'Settings', jackett: { __typename?: 'JackettSettings', configured: boolean, enabled: boolean, url: string, apiKeyConfigured: boolean } } };

export type UpdateQBittorrentSettingsDocumentMutationVariables = Exact<{
  input: UpdateQBittorrentSettingsInput;
}>;


export type UpdateQBittorrentSettingsDocumentMutation = { __typename?: 'Mutation', updateQBittorrentSettings: { __typename?: 'Settings', qbittorrent: { __typename?: 'QBittorrentSettings', configured: boolean, enabled: boolean, url: string, username: string, usernameConfigured: boolean, passwordConfigured: boolean, defaultSavePath: string, category: string, tags: string } } };

export type UpdateAutomationSettingsDocumentMutationVariables = Exact<{
  input: UpdateAutomationSettingsInput;
}>;


export type UpdateAutomationSettingsDocumentMutation = { __typename?: 'Mutation', updateAutomationSettings: { __typename?: 'Settings', automation: { __typename?: 'AutomationSettings', taskProgressSyncIntervalSeconds: number, subscriptionPollIntervalSeconds: number } } };

export type UpdateSubscriptionSettingsDocumentMutationVariables = Exact<{
  input: UpdateSubscriptionSettingsInput;
}>;


export type UpdateSubscriptionSettingsDocumentMutation = { __typename?: 'Mutation', updateSubscriptionSettings: { __typename?: 'Settings', subscription: { __typename?: 'SubscriptionSettings', stashBoxEndpoints: Array<string> } } };

export type RefreshSubscriptionStashBoxesDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type RefreshSubscriptionStashBoxesDocumentMutation = { __typename?: 'Mutation', refreshSubscriptionStashBoxes: { __typename?: 'Settings', subscription: { __typename?: 'SubscriptionSettings', stashBoxEndpoints: Array<string> } } };

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

export type SubscribedPerformersQueryVariables = Exact<{ [key: string]: never; }>;


export type SubscribedPerformersQuery = { __typename?: 'Query', subscribedPerformers: Array<{ __typename?: 'SubscribedPerformer', lastCheckedAt?: string | null, lastError?: string | null, pendingReleaseCount: number, processedReleaseCount: number, performer: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }, recentReleases: Array<{ __typename?: 'SubscriptionRelease', key: string, source: string, title: string, code?: string | null, date?: string | null, url?: string | null, query: string, taskID?: string | null, seenAt: string }> }> };

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


export type RefreshSubscribedPerformerMutation = { __typename?: 'Mutation', refreshSubscribedPerformer: { __typename?: 'SubscribedPerformer', lastCheckedAt?: string | null, lastError?: string | null, pendingReleaseCount: number, processedReleaseCount: number, performer: { __typename?: 'StashPerformer', id: string, subscribed: boolean }, recentReleases: Array<{ __typename?: 'SubscriptionRelease', key: string, title: string, code?: string | null, date?: string | null, query: string, taskID?: string | null, seenAt: string }> } };

export type RefreshSubscriptionNowMutationVariables = Exact<{ [key: string]: never; }>;


export type RefreshSubscriptionNowMutation = { __typename?: 'Mutation', refreshSubscriptionsNow: Array<{ __typename?: 'SubscribedPerformer', performer: { __typename?: 'StashPerformer', id: string } }> };

export type AddTorrentDocumentMutationVariables = Exact<{
  input: QBittorrentAddInput;
}>;


export type AddTorrentDocumentMutation = { __typename?: 'Mutation', addTorrent: { __typename?: 'Task', id: string, status: string, query: string, torrentName: string, progress: number, stashMode: string, stashScanStatus: string, createdAt: string } };

export type SyncTaskProgressDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type SyncTaskProgressDocumentMutation = { __typename?: 'Mutation', syncTaskProgress: Array<{ __typename?: 'Task', id: string, status: string, progress: number, qbittorrentState: string, updatedAt: string }> };

export type TriggerStashScansDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type TriggerStashScansDocumentMutation = { __typename?: 'Mutation', triggerStashScans: Array<{ __typename?: 'Task', id: string, stashMode: string, stashTransferStatus: string, stashTransferError: string, stashJobId: string, stashScanPath: string, stashScanStatus: string, stashScanError: string, stashScanHint: string, updatedAt: string }> };

export type TriggerTaskStashScanDocumentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type TriggerTaskStashScanDocumentMutation = { __typename?: 'Mutation', triggerTaskStashScan: { __typename?: 'Task', id: string, stashMode: string, stashTransferStatus: string, stashTransferError: string, stashJobId: string, stashScanPath: string, stashScanStatus: string, stashScanError: string, stashScanHint: string, updatedAt: string } };


export const DashboardDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DashboardDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"dashboardStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"active"}},{"kind":"Field","name":{"kind":"Name","value":"completed"}},{"kind":"Field","name":{"kind":"Name","value":"downloading"}},{"kind":"Field","name":{"kind":"Name","value":"pendingScans"}},{"kind":"Field","name":{"kind":"Name","value":"failed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"apiKey"}},{"kind":"Field","name":{"kind":"Name","value":"mode"}},{"kind":"Field","name":{"kind":"Name","value":"libraryPath"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentPathPrefix"}},{"kind":"Field","name":{"kind":"Name","value":"stashPathPrefix"}},{"kind":"Field","name":{"kind":"Name","value":"transferAction"}},{"kind":"Field","name":{"kind":"Name","value":"transferTargetPath"}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"apiKey"}},{"kind":"Field","name":{"kind":"Name","value":"passwordConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"password"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"username"}},{"kind":"Field","name":{"kind":"Name","value":"usernameConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"passwordConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"password"}},{"kind":"Field","name":{"kind":"Name","value":"defaultSavePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}}]}},{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionPollIntervalSeconds"}}]}},{"kind":"Field","name":{"kind":"Name","value":"subscription"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashBoxEndpoints"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"settingsStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionPollIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionPollEnabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"subscription"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashBoxes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxesLoaded"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxesLoadError"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stashStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingMojiScanCount"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"okAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackettStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerCount"}},{"kind":"Field","name":{"kind":"Name","value":"configuredIndexerCount"}},{"kind":"Field","name":{"kind":"Name","value":"lastIndexerLatencyMs"}},{"kind":"Field","name":{"kind":"Name","value":"lastIndexerError"}},{"kind":"Field","name":{"kind":"Name","value":"lastIndexerSearchAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"okAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"downloadSpeed"}},{"kind":"Field","name":{"kind":"Name","value":"uploadSpeed"}},{"kind":"Field","name":{"kind":"Name","value":"activeTorrentCount"}},{"kind":"Field","name":{"kind":"Name","value":"connectionStatus"}},{"kind":"Field","name":{"kind":"Name","value":"altSpeedLimitEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"okAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"tasks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"contentPath"}},{"kind":"Field","name":{"kind":"Name","value":"torrentHash"}},{"kind":"Field","name":{"kind":"Name","value":"savePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"completedAt"}},{"kind":"Field","name":{"kind":"Name","value":"stashMode"}},{"kind":"Field","name":{"kind":"Name","value":"stashSourcePath"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferAction"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferError"}},{"kind":"Field","name":{"kind":"Name","value":"stashJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanHint"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<DashboardDocumentQuery, DashboardDocumentQueryVariables>;
export const SearchDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SearchDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"JackettSearchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jackettSearch"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}},{"kind":"Field","name":{"kind":"Name","value":"tracker"}},{"kind":"Field","name":{"kind":"Name","value":"categoryDesc"}},{"kind":"Field","name":{"kind":"Name","value":"publishDate"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"magnetUri"}}]}}]}}]} as unknown as DocumentNode<SearchDocumentQuery, SearchDocumentQueryVariables>;
export const UpdateStashSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateStashSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateStashSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateStashSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"mode"}},{"kind":"Field","name":{"kind":"Name","value":"libraryPath"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentPathPrefix"}},{"kind":"Field","name":{"kind":"Name","value":"stashPathPrefix"}},{"kind":"Field","name":{"kind":"Name","value":"transferAction"}},{"kind":"Field","name":{"kind":"Name","value":"transferTargetPath"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateStashSettingsDocumentMutation, UpdateStashSettingsDocumentMutationVariables>;
export const UpdateJackettSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateJackettSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateJackettSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateJackettSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateJackettSettingsDocumentMutation, UpdateJackettSettingsDocumentMutationVariables>;
export const UpdateQBittorrentSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateQBittorrentSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateQBittorrentSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateQBittorrentSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"username"}},{"kind":"Field","name":{"kind":"Name","value":"usernameConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"passwordConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"defaultSavePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateQBittorrentSettingsDocumentMutation, UpdateQBittorrentSettingsDocumentMutationVariables>;
export const UpdateAutomationSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateAutomationSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateAutomationSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateAutomationSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionPollIntervalSeconds"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateAutomationSettingsDocumentMutation, UpdateAutomationSettingsDocumentMutationVariables>;
export const UpdateSubscriptionSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateSubscriptionSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateSubscriptionSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateSubscriptionSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscription"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashBoxEndpoints"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateSubscriptionSettingsDocumentMutation, UpdateSubscriptionSettingsDocumentMutationVariables>;
export const RefreshSubscriptionStashBoxesDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshSubscriptionStashBoxesDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshSubscriptionStashBoxes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscription"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashBoxEndpoints"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshSubscriptionStashBoxesDocumentMutation, RefreshSubscriptionStashBoxesDocumentMutationVariables>;
export const LogsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"LogsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"minLevel"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"LogLevel"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"logs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}},{"kind":"Argument","name":{"kind":"Name","value":"minLevel"},"value":{"kind":"Variable","name":{"kind":"Name","value":"minLevel"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"level"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<LogsDocumentQuery, LogsDocumentQueryVariables>;
export const StashPerformersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"StashPerformers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"search"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashPerformers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"search"},"value":{"kind":"Variable","name":{"kind":"Name","value":"search"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}},{"kind":"Argument","name":{"kind":"Name","value":"pageSize"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"pageSize"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"hasPrevPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}}]}}]}}]} as unknown as DocumentNode<StashPerformersQuery, StashPerformersQueryVariables>;
export const SubscribedPerformersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SubscribedPerformers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscribedPerformers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"taskID"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]} as unknown as DocumentNode<SubscribedPerformersQuery, SubscribedPerformersQueryVariables>;
export const SubscribePerformerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SubscribePerformer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscribePerformer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"stashPerformerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}}]}}]}}]} as unknown as DocumentNode<SubscribePerformerMutation, SubscribePerformerMutationVariables>;
export const UnsubscribePerformerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnsubscribePerformer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unsubscribePerformer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"stashPerformerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}}}]}]}}]} as unknown as DocumentNode<UnsubscribePerformerMutation, UnsubscribePerformerMutationVariables>;
export const RefreshSubscribedPerformerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshSubscribedPerformer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshSubscribedPerformer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"stashPerformerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"taskID"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshSubscribedPerformerMutation, RefreshSubscribedPerformerMutationVariables>;
export const RefreshSubscriptionNowDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshSubscriptionNow"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshSubscriptionsNow"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshSubscriptionNowMutation, RefreshSubscriptionNowMutationVariables>;
export const AddTorrentDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddTorrentDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"QBittorrentAddInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"addTorrent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"stashMode"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<AddTorrentDocumentMutation, AddTorrentDocumentMutationVariables>;
export const SyncTaskProgressDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SyncTaskProgressDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"syncTaskProgress"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<SyncTaskProgressDocumentMutation, SyncTaskProgressDocumentMutationVariables>;
export const TriggerStashScansDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TriggerStashScansDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggerStashScans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"stashMode"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferError"}},{"kind":"Field","name":{"kind":"Name","value":"stashJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanHint"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<TriggerStashScansDocumentMutation, TriggerStashScansDocumentMutationVariables>;
export const TriggerTaskStashScanDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TriggerTaskStashScanDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggerTaskStashScan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"stashMode"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashTransferError"}},{"kind":"Field","name":{"kind":"Name","value":"stashJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanHint"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<TriggerTaskStashScanDocumentMutation, TriggerTaskStashScanDocumentMutationVariables>;