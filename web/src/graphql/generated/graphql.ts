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
  apiKeyConfigured: Scalars['Boolean']['output'];
  configured: Scalars['Boolean']['output'];
  enabled: Scalars['Boolean']['output'];
  url: Scalars['String']['output'];
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

export type LoggingSettings = {
  __typename?: 'LoggingSettings';
  filePath: Scalars['String']['output'];
  level: Scalars['String']['output'];
  maxEntries: Scalars['Int']['output'];
  maxFileBackups: Scalars['Int']['output'];
  maxFileSizeBytes: Scalars['Int']['output'];
};

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
  /** Update Jackett settings and persist them to backend config */
  updateJackettSettings: Settings;
  /** Update logging settings and persist them to backend config */
  updateLoggingSettings: Settings;
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


export type MutationUpdateJackettSettingsArgs = {
  input: UpdateJackettSettingsInput;
};


export type MutationUpdateLoggingSettingsArgs = {
  input: UpdateLoggingSettingsInput;
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
  passwordConfigured: Scalars['Boolean']['output'];
  tags: Scalars['String']['output'];
  url: Scalars['String']['output'];
  username: Scalars['String']['output'];
  usernameConfigured: Scalars['Boolean']['output'];
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
  /** Get runtime capability and configuration state for the Settings surface */
  settings: Settings;
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

export type Settings = {
  __typename?: 'Settings';
  jackett: JackettSettings;
  logging: LoggingSettings;
  qbittorrent: QBittorrentSettings;
  stash: StashSettings;
  subscription: SubscriptionSettings;
  system: SystemSettings;
  tasks: TaskSettings;
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
  apiKeyConfigured: Scalars['Boolean']['output'];
  configured: Scalars['Boolean']['output'];
  enabled: Scalars['Boolean']['output'];
  libraryPath: Scalars['String']['output'];
  url: Scalars['String']['output'];
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
  dbPath: Scalars['String']['output'];
  pollEnabled: Scalars['Boolean']['output'];
  pollIntervalSeconds: Scalars['Int']['output'];
  /** Endpoints the user has selected for subscription lookups. Empty = use every configured Stash Box. */
  selectedStashBoxEndpoints: Array<Scalars['String']['output']>;
  /** Stash Box instances currently configured inside the Stash server. */
  stashBoxes: Array<StashBoxEndpoint>;
  /** Reason for the most recent Stash Box load failure. Null when stashBoxesLoaded is true. */
  stashBoxesLoadError?: Maybe<Scalars['String']['output']>;
  /** Whether the last attempt to load Stash Box endpoints from Stash succeeded. */
  stashBoxesLoaded: Scalars['Boolean']['output'];
  store: Scalars['String']['output'];
};

export type SystemSettings = {
  __typename?: 'SystemSettings';
  appVersion: Scalars['String']['output'];
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
  stashScanError: Scalars['String']['output'];
  stashScanStartedAt?: Maybe<Scalars['String']['output']>;
  stashScanStatus: Scalars['String']['output'];
  status: Scalars['String']['output'];
  tags: Scalars['String']['output'];
  torrentHash: Scalars['String']['output'];
  torrentName: Scalars['String']['output'];
  torrentUrl: Scalars['String']['output'];
  updatedAt: Scalars['String']['output'];
};

export type TaskSettings = {
  __typename?: 'TaskSettings';
  dbPath: Scalars['String']['output'];
  progressSyncEnabled: Scalars['Boolean']['output'];
  progressSyncIntervalSeconds: Scalars['Int']['output'];
  store: Scalars['String']['output'];
};

export type UpdateJackettSettingsInput = {
  apiKey?: InputMaybe<Scalars['String']['input']>;
  url: Scalars['String']['input'];
};

export type UpdateLoggingSettingsInput = {
  filePath: Scalars['String']['input'];
  level: Scalars['String']['input'];
  maxEntries: Scalars['Int']['input'];
  maxFileBackups: Scalars['Int']['input'];
  maxFileSizeBytes: Scalars['Int']['input'];
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
  url: Scalars['String']['input'];
};

export type UpdateSubscriptionSettingsInput = {
  dbPath: Scalars['String']['input'];
  pollIntervalSeconds: Scalars['Int']['input'];
  /** Endpoints selected for subscription lookups. Empty = use every configured Stash Box. */
  selectedStashBoxEndpoints: Array<Scalars['String']['input']>;
  store: Scalars['String']['input'];
};

export type DashboardDocumentQueryVariables = Exact<{ [key: string]: never; }>;


export type DashboardDocumentQuery = { __typename?: 'Query', version: string, health: { __typename?: 'Health', ok: boolean, message: string }, dashboardStats: { __typename?: 'DashboardStats', total: number, active: number, completed: number, downloading: number, pendingScans: number, failed: number }, settings: { __typename?: 'Settings', stash: { __typename?: 'StashSettings', configured: boolean, enabled: boolean, url: string, apiKeyConfigured: boolean, libraryPath: string }, jackett: { __typename?: 'JackettSettings', configured: boolean, enabled: boolean, url: string, apiKeyConfigured: boolean }, qbittorrent: { __typename?: 'QBittorrentSettings', configured: boolean, enabled: boolean, url: string, username: string, usernameConfigured: boolean, passwordConfigured: boolean, defaultSavePath: string, category: string, tags: string }, tasks: { __typename?: 'TaskSettings', store: string, dbPath: string, progressSyncIntervalSeconds: number, progressSyncEnabled: boolean }, subscription: { __typename?: 'SubscriptionSettings', store: string, dbPath: string, pollIntervalSeconds: number, pollEnabled: boolean, selectedStashBoxEndpoints: Array<string>, stashBoxesLoaded: boolean, stashBoxesLoadError?: string | null, stashBoxes: Array<{ __typename?: 'StashBoxEndpoint', name: string, endpoint: string, apiKeyConfigured: boolean }> }, logging: { __typename?: 'LoggingSettings', level: string, filePath: string, maxEntries: number, maxFileSizeBytes: number, maxFileBackups: number }, system: { __typename?: 'SystemSettings', appVersion: string } }, tasks: Array<{ __typename?: 'Task', id: string, query: string, status: string, torrentName: string, progress: number, qbittorrentState: string, contentPath: string, torrentHash: string, savePath: string, category: string, tags: string, error: string, completedAt?: string | null, stashJobId: string, stashScanStatus: string, stashScanError: string, createdAt: string, updatedAt: string }> };

export type SearchDocumentQueryVariables = Exact<{
  input: JackettSearchInput;
}>;


export type SearchDocumentQuery = { __typename?: 'Query', jackettSearch: Array<{ __typename?: 'JackettSearchResult', title: string, size: any, seeders: number, peers: number, tracker: string, categoryDesc: string, publishDate: string, link: string, magnetUri: string }> };

export type UpdateStashSettingsDocumentMutationVariables = Exact<{
  input: UpdateStashSettingsInput;
}>;


export type UpdateStashSettingsDocumentMutation = { __typename?: 'Mutation', updateStashSettings: { __typename?: 'Settings', stash: { __typename?: 'StashSettings', configured: boolean, enabled: boolean, url: string, apiKeyConfigured: boolean, libraryPath: string } } };

export type UpdateJackettSettingsDocumentMutationVariables = Exact<{
  input: UpdateJackettSettingsInput;
}>;


export type UpdateJackettSettingsDocumentMutation = { __typename?: 'Mutation', updateJackettSettings: { __typename?: 'Settings', jackett: { __typename?: 'JackettSettings', configured: boolean, enabled: boolean, url: string, apiKeyConfigured: boolean } } };

export type UpdateQBittorrentSettingsDocumentMutationVariables = Exact<{
  input: UpdateQBittorrentSettingsInput;
}>;


export type UpdateQBittorrentSettingsDocumentMutation = { __typename?: 'Mutation', updateQBittorrentSettings: { __typename?: 'Settings', qbittorrent: { __typename?: 'QBittorrentSettings', configured: boolean, enabled: boolean, url: string, username: string, usernameConfigured: boolean, passwordConfigured: boolean, defaultSavePath: string, category: string, tags: string } } };

export type UpdateSubscriptionSettingsDocumentMutationVariables = Exact<{
  input: UpdateSubscriptionSettingsInput;
}>;


export type UpdateSubscriptionSettingsDocumentMutation = { __typename?: 'Mutation', updateSubscriptionSettings: { __typename?: 'Settings', subscription: { __typename?: 'SubscriptionSettings', store: string, dbPath: string, pollIntervalSeconds: number, pollEnabled: boolean, selectedStashBoxEndpoints: Array<string>, stashBoxesLoaded: boolean, stashBoxesLoadError?: string | null, stashBoxes: Array<{ __typename?: 'StashBoxEndpoint', name: string, endpoint: string, apiKeyConfigured: boolean }> } } };

export type UpdateLoggingSettingsDocumentMutationVariables = Exact<{
  input: UpdateLoggingSettingsInput;
}>;


export type UpdateLoggingSettingsDocumentMutation = { __typename?: 'Mutation', updateLoggingSettings: { __typename?: 'Settings', logging: { __typename?: 'LoggingSettings', level: string, filePath: string, maxEntries: number, maxFileSizeBytes: number, maxFileBackups: number } } };

export type RefreshSubscriptionStashBoxesDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type RefreshSubscriptionStashBoxesDocumentMutation = { __typename?: 'Mutation', refreshSubscriptionStashBoxes: { __typename?: 'Settings', subscription: { __typename?: 'SubscriptionSettings', store: string, dbPath: string, pollIntervalSeconds: number, pollEnabled: boolean, selectedStashBoxEndpoints: Array<string>, stashBoxesLoaded: boolean, stashBoxesLoadError?: string | null, stashBoxes: Array<{ __typename?: 'StashBoxEndpoint', name: string, endpoint: string, apiKeyConfigured: boolean }> } } };

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


export type AddTorrentDocumentMutation = { __typename?: 'Mutation', addTorrent: { __typename?: 'Task', id: string, status: string, query: string, torrentName: string, progress: number, stashScanStatus: string, createdAt: string } };

export type SyncTaskProgressDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type SyncTaskProgressDocumentMutation = { __typename?: 'Mutation', syncTaskProgress: Array<{ __typename?: 'Task', id: string, status: string, progress: number, qbittorrentState: string, updatedAt: string }> };

export type TriggerStashScansDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type TriggerStashScansDocumentMutation = { __typename?: 'Mutation', triggerStashScans: Array<{ __typename?: 'Task', id: string, stashJobId: string, stashScanStatus: string, stashScanError: string, updatedAt: string }> };

export type TriggerTaskStashScanDocumentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type TriggerTaskStashScanDocumentMutation = { __typename?: 'Mutation', triggerTaskStashScan: { __typename?: 'Task', id: string, stashJobId: string, stashScanStatus: string, stashScanError: string, updatedAt: string } };


export const DashboardDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DashboardDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"dashboardStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"active"}},{"kind":"Field","name":{"kind":"Name","value":"completed"}},{"kind":"Field","name":{"kind":"Name","value":"downloading"}},{"kind":"Field","name":{"kind":"Name","value":"pendingScans"}},{"kind":"Field","name":{"kind":"Name","value":"failed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"libraryPath"}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"username"}},{"kind":"Field","name":{"kind":"Name","value":"usernameConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"passwordConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"defaultSavePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}}]}},{"kind":"Field","name":{"kind":"Name","value":"tasks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"store"}},{"kind":"Field","name":{"kind":"Name","value":"dbPath"}},{"kind":"Field","name":{"kind":"Name","value":"progressSyncIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"progressSyncEnabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"subscription"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"store"}},{"kind":"Field","name":{"kind":"Name","value":"dbPath"}},{"kind":"Field","name":{"kind":"Name","value":"pollIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"pollEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}},{"kind":"Field","name":{"kind":"Name","value":"selectedStashBoxEndpoints"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxesLoaded"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxesLoadError"}}]}},{"kind":"Field","name":{"kind":"Name","value":"logging"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"level"}},{"kind":"Field","name":{"kind":"Name","value":"filePath"}},{"kind":"Field","name":{"kind":"Name","value":"maxEntries"}},{"kind":"Field","name":{"kind":"Name","value":"maxFileSizeBytes"}},{"kind":"Field","name":{"kind":"Name","value":"maxFileBackups"}}]}},{"kind":"Field","name":{"kind":"Name","value":"system"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"appVersion"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"tasks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"contentPath"}},{"kind":"Field","name":{"kind":"Name","value":"torrentHash"}},{"kind":"Field","name":{"kind":"Name","value":"savePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"completedAt"}},{"kind":"Field","name":{"kind":"Name","value":"stashJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<DashboardDocumentQuery, DashboardDocumentQueryVariables>;
export const SearchDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SearchDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"JackettSearchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jackettSearch"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}},{"kind":"Field","name":{"kind":"Name","value":"tracker"}},{"kind":"Field","name":{"kind":"Name","value":"categoryDesc"}},{"kind":"Field","name":{"kind":"Name","value":"publishDate"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"magnetUri"}}]}}]}}]} as unknown as DocumentNode<SearchDocumentQuery, SearchDocumentQueryVariables>;
export const UpdateStashSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateStashSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateStashSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateStashSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"libraryPath"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateStashSettingsDocumentMutation, UpdateStashSettingsDocumentMutationVariables>;
export const UpdateJackettSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateJackettSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateJackettSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateJackettSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateJackettSettingsDocumentMutation, UpdateJackettSettingsDocumentMutationVariables>;
export const UpdateQBittorrentSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateQBittorrentSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateQBittorrentSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateQBittorrentSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"username"}},{"kind":"Field","name":{"kind":"Name","value":"usernameConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"passwordConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"defaultSavePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateQBittorrentSettingsDocumentMutation, UpdateQBittorrentSettingsDocumentMutationVariables>;
export const UpdateSubscriptionSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateSubscriptionSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateSubscriptionSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateSubscriptionSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscription"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"store"}},{"kind":"Field","name":{"kind":"Name","value":"dbPath"}},{"kind":"Field","name":{"kind":"Name","value":"pollIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"pollEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}},{"kind":"Field","name":{"kind":"Name","value":"selectedStashBoxEndpoints"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxesLoaded"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxesLoadError"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateSubscriptionSettingsDocumentMutation, UpdateSubscriptionSettingsDocumentMutationVariables>;
export const UpdateLoggingSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateLoggingSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateLoggingSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateLoggingSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"logging"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"level"}},{"kind":"Field","name":{"kind":"Name","value":"filePath"}},{"kind":"Field","name":{"kind":"Name","value":"maxEntries"}},{"kind":"Field","name":{"kind":"Name","value":"maxFileSizeBytes"}},{"kind":"Field","name":{"kind":"Name","value":"maxFileBackups"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateLoggingSettingsDocumentMutation, UpdateLoggingSettingsDocumentMutationVariables>;
export const RefreshSubscriptionStashBoxesDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshSubscriptionStashBoxesDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshSubscriptionStashBoxes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscription"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"store"}},{"kind":"Field","name":{"kind":"Name","value":"dbPath"}},{"kind":"Field","name":{"kind":"Name","value":"pollIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"pollEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}},{"kind":"Field","name":{"kind":"Name","value":"selectedStashBoxEndpoints"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxesLoaded"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxesLoadError"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshSubscriptionStashBoxesDocumentMutation, RefreshSubscriptionStashBoxesDocumentMutationVariables>;
export const LogsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"LogsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"minLevel"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"LogLevel"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"logs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}},{"kind":"Argument","name":{"kind":"Name","value":"minLevel"},"value":{"kind":"Variable","name":{"kind":"Name","value":"minLevel"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"level"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<LogsDocumentQuery, LogsDocumentQueryVariables>;
export const StashPerformersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"StashPerformers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"search"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashPerformers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"search"},"value":{"kind":"Variable","name":{"kind":"Name","value":"search"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}},{"kind":"Argument","name":{"kind":"Name","value":"pageSize"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"pageSize"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"hasPrevPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}}]}}]}}]} as unknown as DocumentNode<StashPerformersQuery, StashPerformersQueryVariables>;
export const SubscribedPerformersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SubscribedPerformers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscribedPerformers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"taskID"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]} as unknown as DocumentNode<SubscribedPerformersQuery, SubscribedPerformersQueryVariables>;
export const SubscribePerformerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SubscribePerformer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscribePerformer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"stashPerformerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}}]}}]}}]} as unknown as DocumentNode<SubscribePerformerMutation, SubscribePerformerMutationVariables>;
export const UnsubscribePerformerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnsubscribePerformer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unsubscribePerformer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"stashPerformerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}}}]}]}}]} as unknown as DocumentNode<UnsubscribePerformerMutation, UnsubscribePerformerMutationVariables>;
export const RefreshSubscribedPerformerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshSubscribedPerformer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshSubscribedPerformer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"stashPerformerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"taskID"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshSubscribedPerformerMutation, RefreshSubscribedPerformerMutationVariables>;
export const RefreshSubscriptionNowDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshSubscriptionNow"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshSubscriptionsNow"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshSubscriptionNowMutation, RefreshSubscriptionNowMutationVariables>;
export const AddTorrentDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddTorrentDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"QBittorrentAddInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"addTorrent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<AddTorrentDocumentMutation, AddTorrentDocumentMutationVariables>;
export const SyncTaskProgressDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SyncTaskProgressDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"syncTaskProgress"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<SyncTaskProgressDocumentMutation, SyncTaskProgressDocumentMutationVariables>;
export const TriggerStashScansDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TriggerStashScansDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggerStashScans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"stashJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<TriggerStashScansDocumentMutation, TriggerStashScansDocumentMutationVariables>;
export const TriggerTaskStashScanDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TriggerTaskStashScanDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggerTaskStashScan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"stashJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<TriggerTaskStashScanDocumentMutation, TriggerTaskStashScanDocumentMutationVariables>;