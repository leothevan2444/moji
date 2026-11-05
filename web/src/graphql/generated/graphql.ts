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
  Long: { input: number; output: number; }
  Upload: { input: File; output: File; }
};

export type AutomationSettings = {
  __typename?: 'AutomationSettings';
  sourcingRetryIntervalHours: Scalars['Int']['output'];
  /** Endpoint URLs in the user-defined order used for subscription lookups. Endpoints not listed here are still queried, in their Stash order, appended after the listed ones. An empty list means use Stash's order as-is. */
  stashBoxEndpoints: Array<Scalars['String']['output']>;
  subscriptionPollIntervalHours: Scalars['Int']['output'];
  subscriptionReleasePolicy: SubscriptionReleasePolicy;
  /** Whether automatic download and ingest processing is enabled */
  taskProcessingEnabled: Scalars['Boolean']['output'];
  taskProgressSyncIntervalSeconds: Scalars['Int']['output'];
  torrentSelection: TorrentSelectionSettings;
};

/** Legacy aggregate mutation input; new section mutations use SettingsSection. */
export enum AutomationSettingsSection {
  All = 'ALL',
  Indexers = 'INDEXERS',
  Metadata = 'METADATA',
  Resources = 'RESOURCES',
  Tasks = 'TASKS',
  TaskDeletion = 'TASK_DELETION'
}

export type AutomationStatus = {
  __typename?: 'AutomationStatus';
  subscriptionPollEnabled: Scalars['Boolean']['output'];
  subscriptionPollIntervalHours: Scalars['Int']['output'];
  taskProgressSyncEnabled: Scalars['Boolean']['output'];
  taskProgressSyncIntervalSeconds: Scalars['Int']['output'];
};

export type BuiltinIndexerInput = {
  baseUrl?: InputMaybe<Scalars['String']['input']>;
  definitionId: Scalars['ID']['input'];
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
  name?: InputMaybe<Scalars['String']['input']>;
  networkProfile?: InputMaybe<Scalars['String']['input']>;
  settings?: InputMaybe<Array<IndexerSettingChangeInput>>;
  trustedOrigins?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type BuiltinIndexerPatchInput = {
  baseUrl?: InputMaybe<Scalars['String']['input']>;
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
  name?: InputMaybe<Scalars['String']['input']>;
  networkProfile?: InputMaybe<Scalars['String']['input']>;
  settings?: InputMaybe<Array<IndexerSettingChangeInput>>;
  trustedOrigins?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type BuiltinIndexerResult = {
  __typename?: 'BuiltinIndexerResult';
  instance: IndexerInstance;
  revision: Scalars['String']['output'];
};

export type DashboardStats = {
  __typename?: 'DashboardStats';
  active: Scalars['Int']['output'];
  attention: Scalars['Int']['output'];
  completed: Scalars['Int']['output'];
  downloading: Scalars['Int']['output'];
  failed: Scalars['Int']['output'];
  pendingScans: Scalars['Int']['output'];
  total: Scalars['Int']['output'];
  waiting: Scalars['Int']['output'];
};

export type DatabasePerformance = {
  __typename?: 'DatabasePerformance';
  averageExecutionMilliseconds: Scalars['Float']['output'];
  averageWaitMilliseconds: Scalars['Float']['output'];
  busyRetries: Scalars['Long']['output'];
  databaseBytes: Scalars['Long']['output'];
  freeBytes: Scalars['Long']['output'];
  queueCanceled: Scalars['Long']['output'];
  queueCapacity: Scalars['Int']['output'];
  queueRejected: Scalars['Long']['output'];
  queued: Scalars['Int']['output'];
  reader: DatabasePoolMetrics;
  role: Scalars['String']['output'];
  totalExecutionMilliseconds: Scalars['Float']['output'];
  totalWaitMilliseconds: Scalars['Float']['output'];
  walBytes: Scalars['Long']['output'];
  writeAttempts: Scalars['Long']['output'];
  writeErrors: Scalars['Long']['output'];
  writer: DatabasePoolMetrics;
  writesPerSecond: Scalars['Float']['output'];
};

export type DatabasePerformanceSnapshot = {
  __typename?: 'DatabasePerformanceSnapshot';
  databases: Array<DatabasePerformance>;
  intervalSeconds: Scalars['Float']['output'];
  lastError: Scalars['String']['output'];
  sampledAt: Scalars['String']['output'];
  startedAt: Scalars['String']['output'];
};

export type DatabasePoolMetrics = {
  __typename?: 'DatabasePoolMetrics';
  idle: Scalars['Int']['output'];
  inUse: Scalars['Int']['output'];
  limit: Scalars['Int']['output'];
  open: Scalars['Int']['output'];
  waitCount: Scalars['Long']['output'];
  waitMilliseconds: Scalars['Float']['output'];
};

export type DeliveryPatchInput = {
  downloads?: InputMaybe<DownloadRootsPatchInput>;
  generate_transcodes?: InputMaybe<Scalars['Boolean']['input']>;
  keep_subtitles?: InputMaybe<Scalars['Boolean']['input']>;
  library?: InputMaybe<LibraryRootsPatchInput>;
  transfer?: InputMaybe<TransferPatchInput>;
};

export type DirectionRule = {
  __typename?: 'DirectionRule';
  direction: TorrentSelectionDirection;
};

export type DirectionRuleInput = {
  direction: TorrentSelectionDirection;
};

export type DirectionRulePatchInput = {
  direction?: InputMaybe<TorrentSelectionDirection>;
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
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
  code: Scalars['String']['input'];
  limit?: InputMaybe<Scalars['Int']['input']>;
  paused?: InputMaybe<Scalars['Boolean']['input']>;
  savePath?: InputMaybe<Scalars['String']['input']>;
  tags?: InputMaybe<Scalars['String']['input']>;
  trackers?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type DownloadRootsPatchInput = {
  moji_root?: InputMaybe<Scalars['String']['input']>;
  qb_root?: InputMaybe<Scalars['String']['input']>;
};

export type DownloaderConnectionPatchInput = {
  qbittorrent?: InputMaybe<QbConnectionPatchInput>;
};

export type DownloaderOverview = {
  __typename?: 'DownloaderOverview';
  attentionCount: Scalars['Int']['output'];
  checkedAt: Scalars['String']['output'];
  completedBytes: Scalars['Long']['output'];
  connected: Scalars['Boolean']['output'];
  connectionStatus: Scalars['String']['output'];
  dhtNodes: Scalars['Long']['output'];
  downloadLimit: Scalars['Long']['output'];
  downloadSpeed: Scalars['Long']['output'];
  freeSpace?: Maybe<Scalars['Long']['output']>;
  policy: DownloaderPolicy;
  remainingBytes: Scalars['Long']['output'];
  stale: Scalars['Boolean']['output'];
  storageAvailable: Scalars['Boolean']['output'];
  torrents: Array<ManagedTorrent>;
  total: Scalars['Int']['output'];
  uploadLimit: Scalars['Long']['output'];
  uploadSpeed: Scalars['Long']['output'];
};

export type DownloaderPolicy = {
  __typename?: 'DownloaderPolicy';
  afterIngest: Scalars['Boolean']['output'];
  deleteFiles: Scalars['Boolean']['output'];
  enabled: Scalars['Boolean']['output'];
  matchAll: Scalars['Boolean']['output'];
  ratio: Scalars['Float']['output'];
  seedHours: Scalars['Float']['output'];
};

export type DownloaderPolicyInput = {
  afterIngest?: InputMaybe<Scalars['Boolean']['input']>;
  deleteFiles?: InputMaybe<Scalars['Boolean']['input']>;
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
  matchAll?: InputMaybe<Scalars['Boolean']['input']>;
  ratio?: InputMaybe<Scalars['Float']['input']>;
  seedHours?: InputMaybe<Scalars['Float']['input']>;
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

export enum ExternalService {
  Jackett = 'JACKETT',
  Qbittorrent = 'QBITTORRENT',
  Stash = 'STASH'
}

export type FastRulesPatchInput = {
  publish_date?: InputMaybe<DirectionRulePatchInput>;
  seeders?: InputMaybe<DirectionRulePatchInput>;
  size?: InputMaybe<DirectionRulePatchInput>;
  title_match?: InputMaybe<TitleRulePatchInput>;
  title_similarity?: InputMaybe<ToggleRulePatchInput>;
};

export type FileClausePatchInput = {
  effect: TorrentFileMatchEffect;
  pattern: Scalars['String']['input'];
  pattern_mode: TitleMatchPatternMode;
};

export type FileNameRulePatchInput = {
  clauses?: InputMaybe<Array<FileClausePatchInput>>;
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
};

export type FileRulesPatchInput = {
  torrent_file_name_match?: InputMaybe<FileNameRulePatchInput>;
  torrent_single_video?: InputMaybe<ToggleRulePatchInput>;
};

/** Basic service health */
export type Health = {
  __typename?: 'Health';
  message: Scalars['String']['output'];
  ok: Scalars['Boolean']['output'];
};

export type HomeCheckPerformer = {
  __typename?: 'HomeCheckPerformer';
  name: Scalars['String']['output'];
  performerId: Scalars['ID']['output'];
};

export type HomeIndexerIssue = {
  __typename?: 'HomeIndexerIssue';
  errorCode: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
  observation?: Maybe<IndexerObservation>;
  operation: Scalars['String']['output'];
};

/** Local indexer overview; reading this field never contacts a remote indexer. */
export type HomeIndexerSummary = {
  __typename?: 'HomeIndexerSummary';
  attention: Scalars['Int']['output'];
  builtinEnabled: Scalars['Int']['output'];
  builtinSearchSucceeded: Scalars['Int']['output'];
  /** Configured sites without an observed verification result; does not imply connectivity. */
  builtinUnverified: Scalars['Int']['output'];
  definitionErrorCode: Scalars['String']['output'];
  /** Latest failed or interrupted persisted definition job; null once superseded. */
  definitionJob?: Maybe<IndexerDefinitionJob>;
  healthErrorCode: Scalars['String']['output'];
  /** At most five enabled sites with configuration, rule or observed operation errors. */
  issues: Array<HomeIndexerIssue>;
  /** Present only when the external gateway is configured and enabled. */
  jackett?: Maybe<HomeJackettSummary>;
  revision: Scalars['String']['output'];
  /** Enabled site count; unknown until the enabled Jackett directory is confirmed. */
  total?: Maybe<Scalars['Int']['output']>;
};

export type HomeJackettSummary = {
  __typename?: 'HomeJackettSummary';
  errorCode: Scalars['String']['output'];
  /** Enabled remote sites; null for an unloaded directory, zero for confirmed empty. */
  sites?: Maybe<Scalars['Int']['output']>;
  stale: Scalars['Boolean']['output'];
  updatedAt?: Maybe<Scalars['String']['output']>;
};

export type HomeOverview = {
  __typename?: 'HomeOverview';
  active: Scalars['Int']['output'];
  activeTasks: Array<Task>;
  attention: Scalars['Int']['output'];
  attentionTasks: Array<Task>;
  completedLast24Hours: Scalars['Int']['output'];
  recentTasks: Array<Task>;
  total: Scalars['Int']['output'];
  waiting: Scalars['Int']['output'];
};

export type HomeSubscriptionCheckRun = {
  __typename?: 'HomeSubscriptionCheckRun';
  completedCount: Scalars['Int']['output'];
  currentPerformers: Array<HomeCheckPerformer>;
  failedCount: Scalars['Int']['output'];
  finishedAt?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  scope: SubscriptionCheckScope;
  skippedCount: Scalars['Int']['output'];
  startedAt: Scalars['String']['output'];
  status: SubscriptionCheckStatus;
  succeededCount: Scalars['Int']['output'];
  totalCount: Scalars['Int']['output'];
  trigger: SubscriptionCheckTrigger;
};

export type HomeSubscriptionChecks = {
  __typename?: 'HomeSubscriptionChecks';
  activeCount: Scalars['Int']['output'];
  activeRuns: Array<HomeSubscriptionCheckRun>;
  enabled: Scalars['Boolean']['output'];
  latestScheduled?: Maybe<HomeSubscriptionCheckRun>;
  nextRunAt?: Maybe<Scalars['String']['output']>;
  recentRuns: Array<HomeSubscriptionCheckRun>;
};

export type HomeSubscriptionSummary = {
  __typename?: 'HomeSubscriptionSummary';
  attention: Scalars['Int']['output'];
  attentionPerformerIds: Array<Scalars['ID']['output']>;
  lastCheckedAt?: Maybe<Scalars['String']['output']>;
  total: Scalars['Int']['output'];
  unchecked: Scalars['Int']['output'];
};

export type HomeTaskSnapshot = {
  __typename?: 'HomeTaskSnapshot';
  active: Scalars['Int']['output'];
  activeTasks: Array<TaskSummary>;
  attention: Scalars['Int']['output'];
  attentionTasks: Array<TaskSummary>;
  completedLast24Hours: Scalars['Int']['output'];
  recentTasks: Array<TaskSummary>;
  sequence: Scalars['Int']['output'];
  streamId: Scalars['String']['output'];
  total: Scalars['Int']['output'];
  waiting: Scalars['Int']['output'];
};

export type ImageCachePatchInput = {
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
  max_size_mb?: InputMaybe<Scalars['Int']['input']>;
  retention_days?: InputMaybe<Scalars['Int']['input']>;
};

export type ImageCacheSettings = {
  __typename?: 'ImageCacheSettings';
  enabled: Scalars['Boolean']['output'];
  maxSizeMb: Scalars['Int']['output'];
  retentionDays: Scalars['Int']['output'];
};

export type ImageCacheSettingsInput = {
  enabled: Scalars['Boolean']['input'];
  maxSizeMb: Scalars['Int']['input'];
  retentionDays: Scalars['Int']['input'];
};

export type ImageCacheStatus = {
  __typename?: 'ImageCacheStatus';
  cacheDirectory: Scalars['String']['output'];
  entryCount: Scalars['Int']['output'];
  lastCleanupAt?: Maybe<Scalars['String']['output']>;
  lastError?: Maybe<Scalars['String']['output']>;
  usedBytes: Scalars['Long']['output'];
};

/** Aggregate counters only; no query keys, source settings, candidates or locators. */
export type IndexerCacheStatus = {
  __typename?: 'IndexerCacheStatus';
  activeRequests: Scalars['Int']['output'];
  activeWaiters: Scalars['Int']['output'];
  emptyTtlSeconds: Scalars['Int']['output'];
  enabled: Scalars['Boolean']['output'];
  entries: Scalars['Int']['output'];
  maxSizeMb: Scalars['Int']['output'];
  revision: Scalars['String']['output'];
  ttlSeconds: Scalars['Int']['output'];
  usedBytes: Scalars['Long']['output'];
};

export type IndexerCheckActivity = {
  __typename?: 'IndexerCheckActivity';
  instanceId: Scalars['ID']['output'];
  requestId: Scalars['ID']['output'];
  state: IndexerCheckActivityState;
};

export enum IndexerCheckActivityState {
  Queued = 'QUEUED',
  Running = 'RUNNING'
}

export type IndexerCheckReport = {
  __typename?: 'IndexerCheckReport';
  configRevision: Scalars['String']['output'];
  finishedAt: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  instanceId: Scalars['ID']['output'];
  kind?: Maybe<IndexerSourceKind>;
  partial: Scalars['Boolean']['output'];
  results: Scalars['Int']['output'];
  ruleRevision: Scalars['String']['output'];
  stages: Array<IndexerCheckStage>;
  startedAt: Scalars['String']['output'];
  truncated: Scalars['Boolean']['output'];
};

export type IndexerCheckStage = {
  __typename?: 'IndexerCheckStage';
  elapsedMs: Scalars['Long']['output'];
  errorCode: Scalars['String']['output'];
  name: IndexerCheckStageName;
  state: IndexerCheckStageState;
};

export enum IndexerCheckStageName {
  Authentication = 'AUTHENTICATION',
  Configuration = 'CONFIGURATION',
  Connection = 'CONNECTION',
  Download = 'DOWNLOAD',
  Search = 'SEARCH'
}

export enum IndexerCheckStageState {
  Empty = 'EMPTY',
  Failed = 'FAILED',
  NotObserved = 'NOT_OBSERVED',
  NotRequested = 'NOT_REQUESTED',
  NotRequired = 'NOT_REQUIRED',
  NoCandidate = 'NO_CANDIDATE',
  Partial = 'PARTIAL',
  Passed = 'PASSED',
  Skipped = 'SKIPPED'
}

export type IndexerDefinition = {
  __typename?: 'IndexerDefinition';
  categories: Array<Scalars['String']['output']>;
  description: Scalars['String']['output'];
  digest: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  language: Scalars['String']['output'];
  links: Array<Scalars['String']['output']>;
  /** Whether the rule declares a login flow; independent of site privacy. */
  loginSupported: Scalars['Boolean']['output'];
  modes: Array<Scalars['String']['output']>;
  name: Scalars['String']['output'];
  privacy: Scalars['String']['output'];
  settings: Array<IndexerSetting>;
  warnings: Array<Scalars['String']['output']>;
};

export type IndexerDefinitionChangeInput = {
  expectedActiveRevision: Scalars['String']['input'];
  targetRevision?: InputMaybe<Scalars['String']['input']>;
};

export type IndexerDefinitionFilterInput = {
  capability?: InputMaybe<Scalars['String']['input']>;
  language?: InputMaybe<Scalars['String']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  privacy?: InputMaybe<Scalars['String']['input']>;
  query?: InputMaybe<Scalars['String']['input']>;
};

export type IndexerDefinitionJob = {
  __typename?: 'IndexerDefinitionJob';
  correlationId: Scalars['String']['output'];
  downloadedBytes: Scalars['Long']['output'];
  errorCode: Scalars['String']['output'];
  expectedActiveRevision: Scalars['String']['output'];
  finishedAt?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  kind: IndexerDefinitionJobKind;
  revision: Scalars['Long']['output'];
  startedAt: Scalars['String']['output'];
  state: IndexerDefinitionJobState;
  targetRevision: Scalars['String']['output'];
};

export enum IndexerDefinitionJobKind {
  Install = 'INSTALL',
  Repair = 'REPAIR',
  Rollback = 'ROLLBACK',
  Update = 'UPDATE'
}

export enum IndexerDefinitionJobState {
  Activating = 'ACTIVATING',
  Cancelled = 'CANCELLED',
  CheckingInstances = 'CHECKING_INSTANCES',
  Completed = 'COMPLETED',
  Downloading = 'DOWNLOADING',
  Failed = 'FAILED',
  Interrupted = 'INTERRUPTED',
  Queued = 'QUEUED',
  Validating = 'VALIDATING'
}

export type IndexerDefinitionManifest = {
  __typename?: 'IndexerDefinitionManifest';
  definitionCount: Scalars['Int']['output'];
  engineVersion: Scalars['String']['output'];
  installedAt: Scalars['String']['output'];
  revision: Scalars['String']['output'];
  schemaVersion: Scalars['Int']['output'];
};

export type IndexerDefinitionPage = {
  __typename?: 'IndexerDefinitionPage';
  items: Array<IndexerDefinition>;
  limit: Scalars['Int']['output'];
  offset: Scalars['Int']['output'];
  revision: Scalars['String']['output'];
  totalCount: Scalars['Int']['output'];
};

export type IndexerDefinitionPolicy = {
  __typename?: 'IndexerDefinitionPolicy';
  autoUpdate: Scalars['Boolean']['output'];
  checkIntervalHours: Scalars['Int']['output'];
  revision: Scalars['String']['output'];
  schemaVersion: Scalars['Int']['output'];
  source: Scalars['String']['output'];
};

export type IndexerDefinitionPolicyPatchInput = {
  autoUpdate?: InputMaybe<Scalars['Boolean']['input']>;
  checkIntervalHours?: InputMaybe<Scalars['Int']['input']>;
};

export type IndexerDefinitionStatus = {
  __typename?: 'IndexerDefinitionStatus';
  activeRevision: Scalars['String']['output'];
  available: Scalars['Boolean']['output'];
  availableRevision: Scalars['String']['output'];
  checkErrorCode: Scalars['String']['output'];
  errorCode: Scalars['String']['output'];
  installed: Array<IndexerDefinitionManifest>;
  job?: Maybe<IndexerDefinitionJob>;
  lastCheckedAt?: Maybe<Scalars['String']['output']>;
  previousRevision: Scalars['String']['output'];
  revision: Scalars['Long']['output'];
  streamId: Scalars['String']['output'];
};

/** Invalidation only: reread the relevant bounded snapshot after READY or a change. */
export type IndexerEvent = {
  __typename?: 'IndexerEvent';
  kind: IndexerEventKind;
  sequence: Scalars['Long']['output'];
  streamId: Scalars['String']['output'];
};

export enum IndexerEventKind {
  Cache = 'CACHE',
  Catalog = 'CATALOG',
  Configuration = 'CONFIGURATION',
  Ready = 'READY',
  Site = 'SITE'
}

/** Last observed outcome of each operation; absence means no matching observation. */
export type IndexerHealth = {
  __typename?: 'IndexerHealth';
  fetch?: Maybe<IndexerObservation>;
  search?: Maybe<IndexerObservation>;
};

export type IndexerInstance = {
  __typename?: 'IndexerInstance';
  baseUrl: Scalars['String']['output'];
  connectionRevision: Scalars['String']['output'];
  definitionId: Scalars['String']['output'];
  description: Scalars['String']['output'];
  enabled: Scalars['Boolean']['output'];
  errorCode: Scalars['String']['output'];
  health: IndexerHealth;
  id: Scalars['ID']['output'];
  kind: IndexerSourceKind;
  language: Scalars['String']['output'];
  /** Whether the current built-in rule declares a login flow. */
  loginSupported: Scalars['Boolean']['output'];
  modes: Array<Scalars['String']['output']>;
  name: Scalars['String']['output'];
  networkProfile: Scalars['String']['output'];
  privacy: Scalars['String']['output'];
  ruleDigest: Scalars['String']['output'];
  settings: Array<IndexerSetting>;
  siteUrl: Scalars['String']['output'];
  trustedOrigins: Array<Scalars['String']['output']>;
};

export type IndexerInstances = {
  __typename?: 'IndexerInstances';
  catalogErrorCode: Scalars['String']['output'];
  healthErrorCode: Scalars['String']['output'];
  items: Array<IndexerInstance>;
  jackettEnabled: Scalars['Boolean']['output'];
  networkProfiles: Array<Scalars['String']['output']>;
  order: Array<Scalars['ID']['output']>;
  revision: Scalars['String']['output'];
  ruleErrorCode: Scalars['String']['output'];
  ruleRevision: Scalars['String']['output'];
};

export type IndexerLoginChallenge = {
  __typename?: 'IndexerLoginChallenge';
  id: Scalars['ID']['output'];
  imageData?: Maybe<Scalars['String']['output']>;
  prompt: Scalars['String']['output'];
  type: Scalars['String']['output'];
};

export type IndexerLoginInput = {
  expectedRevision: Scalars['String']['input'];
  instanceId: Scalars['ID']['input'];
};

export type IndexerLoginReport = {
  __typename?: 'IndexerLoginReport';
  challenge?: Maybe<IndexerLoginChallenge>;
  configRevision: Scalars['String']['output'];
  expiresAt: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  instanceId: Scalars['ID']['output'];
  ruleRevision: Scalars['String']['output'];
  state: IndexerLoginState;
};

export enum IndexerLoginState {
  NotRequired = 'NOT_REQUIRED',
  Ready = 'READY',
  Waiting = 'WAITING'
}

export type IndexerNetworkProfile = {
  __typename?: 'IndexerNetworkProfile';
  flareSolverrConfigured: Scalars['Boolean']['output'];
  id: Scalars['ID']['output'];
  proxyConfigured: Scalars['Boolean']['output'];
  usedBy: Scalars['Int']['output'];
};

export type IndexerNetworkProfilePatchInput = {
  clearFlareSolverr?: InputMaybe<Scalars['Boolean']['input']>;
  clearProxy?: InputMaybe<Scalars['Boolean']['input']>;
  flareSolverrUrl?: InputMaybe<Scalars['String']['input']>;
  proxyUrl?: InputMaybe<Scalars['String']['input']>;
};

export type IndexerNetworkSettings = {
  __typename?: 'IndexerNetworkSettings';
  challengeTimeoutSeconds: Scalars['Int']['output'];
  profiles: Array<IndexerNetworkProfile>;
  requestTimeoutSeconds: Scalars['Int']['output'];
  revision: Scalars['String']['output'];
  siteTimeoutSeconds: Scalars['Int']['output'];
};

export type IndexerNetworkTimeoutPatchInput = {
  challengeTimeoutSeconds?: InputMaybe<Scalars['Int']['input']>;
  requestTimeoutSeconds?: InputMaybe<Scalars['Int']['input']>;
  siteTimeoutSeconds?: InputMaybe<Scalars['Int']['input']>;
};

export type IndexerObservation = {
  __typename?: 'IndexerObservation';
  correlationId: Scalars['String']['output'];
  errorCode: Scalars['String']['output'];
  finishedAt: Scalars['String']['output'];
  retryAt?: Maybe<Scalars['String']['output']>;
  startedAt: Scalars['String']['output'];
};

export type IndexerPreferenceRule = {
  __typename?: 'IndexerPreferenceRule';
  trackerIds: Array<Scalars['String']['output']>;
};

export type IndexerPreferenceRuleInput = {
  trackerIds: Array<Scalars['String']['input']>;
};

export type IndexerSearchCachePatchInput = {
  emptyTtlSeconds?: InputMaybe<Scalars['Int']['input']>;
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
  maxSizeMb?: InputMaybe<Scalars['Int']['input']>;
  ttlSeconds?: InputMaybe<Scalars['Int']['input']>;
};

export type IndexerSetting = {
  __typename?: 'IndexerSetting';
  configured: Scalars['Boolean']['output'];
  defaultValue?: Maybe<IndexerSettingValue>;
  label: Scalars['String']['output'];
  name: Scalars['String']['output'];
  options: Array<IndexerSettingOption>;
  secret: Scalars['Boolean']['output'];
  type: Scalars['String']['output'];
  value?: Maybe<IndexerSettingValue>;
};

export type IndexerSettingChangeInput = {
  clear?: InputMaybe<Scalars['Boolean']['input']>;
  name: Scalars['String']['input'];
  value?: InputMaybe<IndexerSettingValueInput>;
};

export type IndexerSettingOption = {
  __typename?: 'IndexerSettingOption';
  label: Scalars['String']['output'];
  value: Scalars['String']['output'];
};

export type IndexerSettingValue = {
  __typename?: 'IndexerSettingValue';
  boolean?: Maybe<Scalars['Boolean']['output']>;
  number?: Maybe<Scalars['Float']['output']>;
  string?: Maybe<Scalars['String']['output']>;
};

export type IndexerSettingValueInput = {
  boolean?: InputMaybe<Scalars['Boolean']['input']>;
  number?: InputMaybe<Scalars['Float']['input']>;
  string?: InputMaybe<Scalars['String']['input']>;
};

export enum IndexerSourceKind {
  Builtin = 'BUILTIN',
  Jackett = 'JACKETT'
}

export type IndexersPatchInput = {
  jackett?: InputMaybe<JackettPatchInput>;
  query_order?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type IngestSettings = {
  __typename?: 'IngestSettings';
  downloads: DownloadsIngestSettings;
  generateTranscodes: Scalars['Boolean']['output'];
  keepSubtitles: Scalars['Boolean']['output'];
  library: LibraryIngestSettings;
  transfer: TransferIngestSettings;
};

export type IngestStatus = {
  __typename?: 'IngestStatus';
  /** Whether the ingest pipeline is fully wired for the selected mode. Becomes true only when the mode-specific fields are all filled in; reaches false as soon as any required field is cleared. */
  configured: Scalars['Boolean']['output'];
};

export type JackettPatchInput = {
  api_key?: InputMaybe<Scalars['String']['input']>;
  password?: InputMaybe<Scalars['String']['input']>;
  url?: InputMaybe<Scalars['String']['input']>;
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

export type LibraryRootsPatchInput = {
  moji_root?: InputMaybe<Scalars['String']['input']>;
  stash_root?: InputMaybe<Scalars['String']['input']>;
};

export type LogEntry = {
  __typename?: 'LogEntry';
  level: LogLevel;
  message: Scalars['String']['output'];
  sequence: Scalars['Int']['output'];
  time: Scalars['String']['output'];
};

export type LogEvent = {
  __typename?: 'LogEvent';
  entry: LogEntry;
  sequence: Scalars['Int']['output'];
};

export enum LogLevel {
  Debug = 'Debug',
  Error = 'Error',
  Info = 'Info',
  Warning = 'Warning'
}

export type ManagedTorrent = {
  __typename?: 'ManagedTorrent';
  allCompleted: Scalars['Boolean']['output'];
  availability: Scalars['Float']['output'];
  canDeleteFiles: Scalars['Boolean']['output'];
  category: Scalars['String']['output'];
  cleanupError: Scalars['Boolean']['output'];
  completedBytes: Scalars['Long']['output'];
  downloadSpeed: Scalars['Long']['output'];
  downloaded: Scalars['Long']['output'];
  eta: Scalars['Long']['output'];
  hash: Scalars['String']['output'];
  lastSeenAt: Scalars['String']['output'];
  name: Scalars['String']['output'];
  needsAttention: Scalars['Boolean']['output'];
  peers: Scalars['Long']['output'];
  present: Scalars['Boolean']['output'];
  progress: Scalars['Float']['output'];
  ratio: Scalars['Float']['output'];
  removalReason: Scalars['String']['output'];
  removedAt: Scalars['String']['output'];
  savePath: Scalars['String']['output'];
  seedingSeconds: Scalars['Long']['output'];
  seeds: Scalars['Long']['output'];
  size: Scalars['Long']['output'];
  state: Scalars['String']['output'];
  tags: Scalars['String']['output'];
  taskCodes: Array<Scalars['String']['output']>;
  taskIds: Array<Scalars['ID']['output']>;
  uploadSpeed: Scalars['Long']['output'];
  uploaded: Scalars['Long']['output'];
};

export type MatchedStashBox = {
  __typename?: 'MatchedStashBox';
  endpoint: Scalars['String']['output'];
  name: Scalars['String']['output'];
  performerId: Scalars['ID']['output'];
  performerName: Scalars['String']['output'];
};

export type Mutation = {
  __typename?: 'Mutation';
  addBuiltinIndexer: BuiltinIndexerResult;
  addIndexerNetworkProfile: IndexerNetworkSettings;
  /** Add a torrent URL or magnet and create a persisted Moji task */
  addTorrent: Task;
  answerIndexerChallenge: IndexerLoginReport;
  beginIndexerLogin: IndexerLoginReport;
  cancelIndexerDefinitionUpdate: IndexerDefinitionJob;
  cancelIndexerLogin: Scalars['Boolean']['output'];
  cancelTorrentSearch: TorrentSearch;
  checkIndexerDefinitionUpdate: IndexerDefinitionStatus;
  /** Delete cached image files while preserving registered proxy keys */
  clearImageCache: ImageCacheStatus;
  clearIndexerSearchCache: IndexerCacheStatus;
  /** Delete all cached StashBox metadata snapshots and entities */
  clearStashBoxDataCache: StashBoxDataCacheStatus;
  controlTorrent: Scalars['Boolean']['output'];
  /** Delete a persisted Moji task record */
  deleteTask: Task;
  /** Delete selected tasks using the configured deletion policy */
  deleteTasks: TaskBatchPayload;
  /** Persist a download task immediately; search and submit torrent candidates in the background */
  downloadMedia: Task;
  installIndexerDefinitions: IndexerDefinitionJob;
  /** Merge only supplied configuration fields and persist atomically. Objects merge; lists replace. */
  patchSettings: Settings;
  prepareTorrentDownload: PreparedTorrentDownload;
  previewTorrentSelection: TorrentSelectionPreview;
  processReleaseReviews: ReleaseReviewPayload;
  /** Advance selected tasks through transfer and Stash ingest */
  processTaskIngest: TaskBatchPayload;
  /**
   * Add a torrent to qBittorrent via magnet or http(s) URL
   * @deprecated Use addTorrent for persisted Moji task tracking.
   */
  qbittorrentAdd: Scalars['Boolean']['output'];
  /** Queue a discovered StashBox scene into the standard download workflow */
  queueDiscoveredScene: Task;
  /** Queue selected performer scenes into the standard Moji download workflow */
  queuePerformerScenes: QueuePerformerScenesPayload;
  queueTorrentCandidate: Task;
  refreshDownloader: Scalars['Boolean']['output'];
  refreshJackettIndexers: IndexerInstances;
  /** Re-fetch the Stash-Box list from the configured Stash server. The updated list and load status are reflected in the returned Settings snapshot. */
  refreshStashBoxes: Settings;
  /** Force-refresh one performer StashBox cache and return the requested page */
  refreshStashPerformerScenes: StashPerformerSceneConnection;
  /**
   * Legacy synchronous performer check; prefer startSubscriptionCheck for background execution
   * @deprecated Use startSubscriptionCheck(ids: [...]) for tracked background checks.
   */
  refreshSubscribedPerformer: SubscribedPerformer;
  /** Refresh selected subscribed performers against the configured release source */
  refreshSubscribedPerformers: PerformerBatchPayload;
  /** Refresh all subscribed performers against the configured release source */
  refreshSubscriptionsNow: Array<SubscribedPerformer>;
  refreshUpcomingRelease: UpcomingRelease;
  removeBuiltinIndexer: IndexerInstances;
  removeIndexerNetworkProfile: IndexerNetworkSettings;
  reorderIndexers: IndexerInstances;
  repairIndexerDefinitions: IndexerDefinitionJob;
  /** Resolve a blocked sourcing task with a manually selected torrent */
  resolveBlockedSourcingTask: Task;
  /** Choose main video files in playback order for a blocked download */
  resolveTaskFiles: Task;
  resolveTaskSourcingCandidate: Task;
  /** Delete downloaded and delivered files and linked Stash scenes, clear execution state, and return to manual torrent selection */
  restartTaskSourcing: Task;
  /** Retry a blocked task or immediately check a task waiting for resources */
  retryTask: Task;
  /** Retry blocked tasks or check waiting resources with per-task results */
  retryTasks: TaskBatchPayload;
  rollbackIndexerDefinitions: IndexerDefinitionJob;
  setDownloaderSpeedLimit: Scalars['Boolean']['output'];
  setIndexerEnabled: IndexerInstances;
  /** Start a background check. Omit ids to check all subscriptions. Duplicate active checks return the existing run. */
  startSubscriptionCheck: SubscriptionCheckRun;
  startTorrentSearch: TorrentSearch;
  /** Start a Stash metadata scan */
  stashMetadataScan: Scalars['ID']['output'];
  /** Mark a Stash performer as subscribed by Moji */
  subscribePerformer: SubscribedPerformer;
  /** Subscribe selected Stash performers */
  subscribePerformers: PerformerBatchPayload;
  /** Synchronize downloads and advance file preparation and Stash ingest, matching one background processing cycle */
  syncTaskProgress: Array<Task>;
  testIndexer: IndexerCheckReport;
  /** Test the supplied credentials without saving configuration or updating runtime state. */
  testServiceConnection: ServiceConnectionTestResult;
  /** Trigger Stash metadata scans for completed Moji tasks */
  triggerStashScans: Array<Task>;
  /** Trigger a Stash metadata scan for a single completed Moji task */
  triggerTaskStashScan: Task;
  /** Remove Moji subscription mark from a Stash performer */
  unsubscribePerformer: Scalars['Boolean']['output'];
  /** Unsubscribe selected Stash performers */
  unsubscribePerformers: PerformerBatchPayload;
  /**
   * Update automation settings and persist them to backend config
   * @deprecated Use updateSettingsSection to save independent business sections.
   */
  updateAutomationSettings: Settings;
  updateBuiltinIndexer: BuiltinIndexerResult;
  updateDownloaderPolicy: DownloaderPolicy;
  updateIndexerDefinitionPolicy: IndexerDefinitionPolicy;
  updateIndexerDefinitions: IndexerDefinitionJob;
  updateIndexerNetworkProfile: IndexerNetworkSettings;
  updateIndexerNetworkTimeouts: IndexerNetworkSettings;
  updateIndexerSearchCache: IndexerCacheStatus;
  /** Update ingest settings and persist them to backend config */
  updateIngestSettings: Settings;
  /** Update Jackett settings and persist them to backend config */
  updateJackettSettings: Settings;
  /** Update qBittorrent settings and persist them to backend config */
  updateQBittorrentSettings: Settings;
  updateSettingsSection: Settings;
  updateStashBoxCacheSettings: Settings;
  /** Update Stash settings and persist them to backend config */
  updateStashSettings: Settings;
  /** Update system settings and persist them to backend config */
  updateSystemSettings: Settings;
};


export type MutationAddBuiltinIndexerArgs = {
  expectedRevision: Scalars['String']['input'];
  input: BuiltinIndexerInput;
};


export type MutationAddIndexerNetworkProfileArgs = {
  expectedRevision: Scalars['String']['input'];
  id: Scalars['ID']['input'];
  input: IndexerNetworkProfilePatchInput;
};


export type MutationAddTorrentArgs = {
  input: QBittorrentAddInput;
};


export type MutationAnswerIndexerChallengeArgs = {
  answer: Scalars['String']['input'];
  id: Scalars['ID']['input'];
  input: IndexerLoginInput;
};


export type MutationBeginIndexerLoginArgs = {
  input: IndexerLoginInput;
};


export type MutationCancelIndexerDefinitionUpdateArgs = {
  id: Scalars['ID']['input'];
};


export type MutationCancelIndexerLoginArgs = {
  id: Scalars['ID']['input'];
  input: IndexerLoginInput;
};


export type MutationCancelTorrentSearchArgs = {
  id: Scalars['ID']['input'];
};


export type MutationClearIndexerSearchCacheArgs = {
  instanceId?: InputMaybe<Scalars['ID']['input']>;
};


export type MutationControlTorrentArgs = {
  action: TorrentAction;
  hash: Scalars['String']['input'];
};


export type MutationDeleteTaskArgs = {
  id: Scalars['ID']['input'];
};


export type MutationDeleteTasksArgs = {
  ids: Array<Scalars['ID']['input']>;
};


export type MutationDownloadMediaArgs = {
  input: DownloadMediaInput;
};


export type MutationInstallIndexerDefinitionsArgs = {
  input: IndexerDefinitionChangeInput;
};


export type MutationPatchSettingsArgs = {
  input: SettingsPatchInput;
};


export type MutationPrepareTorrentDownloadArgs = {
  input: PrepareTorrentDownloadInput;
};


export type MutationPreviewTorrentSelectionArgs = {
  input: PreviewTorrentSelectionInput;
};


export type MutationProcessReleaseReviewsArgs = {
  input: ProcessReleaseReviewsInput;
};


export type MutationProcessTaskIngestArgs = {
  ids: Array<Scalars['ID']['input']>;
};


export type MutationQbittorrentAddArgs = {
  input: QBittorrentAddInput;
};


export type MutationQueueDiscoveredSceneArgs = {
  input: QueueDiscoveredSceneInput;
};


export type MutationQueuePerformerScenesArgs = {
  input: QueuePerformerScenesInput;
};


export type MutationQueueTorrentCandidateArgs = {
  input: QueueTorrentCandidateInput;
};


export type MutationRefreshStashPerformerScenesArgs = {
  id: Scalars['ID']['input'];
  input: StashPerformerScenesInput;
};


export type MutationRefreshSubscribedPerformerArgs = {
  stashPerformerID: Scalars['ID']['input'];
};


export type MutationRefreshSubscribedPerformersArgs = {
  ids: Array<Scalars['ID']['input']>;
};


export type MutationRefreshUpcomingReleaseArgs = {
  key: Scalars['ID']['input'];
};


export type MutationRemoveBuiltinIndexerArgs = {
  expectedRevision: Scalars['String']['input'];
  id: Scalars['ID']['input'];
};


export type MutationRemoveIndexerNetworkProfileArgs = {
  expectedRevision: Scalars['String']['input'];
  id: Scalars['ID']['input'];
};


export type MutationReorderIndexersArgs = {
  expectedRevision: Scalars['String']['input'];
  ids: Array<Scalars['ID']['input']>;
};


export type MutationRepairIndexerDefinitionsArgs = {
  input: IndexerDefinitionChangeInput;
};


export type MutationResolveBlockedSourcingTaskArgs = {
  id: Scalars['ID']['input'];
  input: ResolveBlockedSourcingTaskInput;
};


export type MutationResolveTaskFilesArgs = {
  id: Scalars['ID']['input'];
  videoIndices: Array<Scalars['Int']['input']>;
};


export type MutationResolveTaskSourcingCandidateArgs = {
  input: ResolveTaskSourcingCandidateInput;
};


export type MutationRestartTaskSourcingArgs = {
  id: Scalars['ID']['input'];
};


export type MutationRetryTaskArgs = {
  id: Scalars['ID']['input'];
};


export type MutationRetryTasksArgs = {
  ids: Array<Scalars['ID']['input']>;
};


export type MutationRollbackIndexerDefinitionsArgs = {
  input: IndexerDefinitionChangeInput;
};


export type MutationSetDownloaderSpeedLimitArgs = {
  kibPerSecond: Scalars['Int']['input'];
  upload: Scalars['Boolean']['input'];
};


export type MutationSetIndexerEnabledArgs = {
  enabled: Scalars['Boolean']['input'];
  expectedRevision: Scalars['String']['input'];
  id: Scalars['ID']['input'];
};


export type MutationStartSubscriptionCheckArgs = {
  ids?: InputMaybe<Array<Scalars['ID']['input']>>;
};


export type MutationStartTorrentSearchArgs = {
  input: StartTorrentSearchInput;
};


export type MutationStashMetadataScanArgs = {
  input: StashMetadataScanInput;
};


export type MutationSubscribePerformerArgs = {
  stashPerformerID: Scalars['ID']['input'];
};


export type MutationSubscribePerformersArgs = {
  ids: Array<Scalars['ID']['input']>;
};


export type MutationTestIndexerArgs = {
  input: TestIndexerInput;
};


export type MutationTestServiceConnectionArgs = {
  input: TestServiceConnectionInput;
};


export type MutationTriggerTaskStashScanArgs = {
  id: Scalars['ID']['input'];
};


export type MutationUnsubscribePerformerArgs = {
  stashPerformerID: Scalars['ID']['input'];
};


export type MutationUnsubscribePerformersArgs = {
  ids: Array<Scalars['ID']['input']>;
};


export type MutationUpdateAutomationSettingsArgs = {
  input: UpdateAutomationSettingsInput;
};


export type MutationUpdateBuiltinIndexerArgs = {
  expectedRevision: Scalars['String']['input'];
  id: Scalars['ID']['input'];
  input: BuiltinIndexerPatchInput;
};


export type MutationUpdateDownloaderPolicyArgs = {
  input: DownloaderPolicyInput;
};


export type MutationUpdateIndexerDefinitionPolicyArgs = {
  expectedRevision: Scalars['String']['input'];
  input: IndexerDefinitionPolicyPatchInput;
};


export type MutationUpdateIndexerDefinitionsArgs = {
  input: IndexerDefinitionChangeInput;
};


export type MutationUpdateIndexerNetworkProfileArgs = {
  expectedRevision: Scalars['String']['input'];
  id: Scalars['ID']['input'];
  input: IndexerNetworkProfilePatchInput;
};


export type MutationUpdateIndexerNetworkTimeoutsArgs = {
  expectedRevision: Scalars['String']['input'];
  input: IndexerNetworkTimeoutPatchInput;
};


export type MutationUpdateIndexerSearchCacheArgs = {
  expectedRevision: Scalars['String']['input'];
  input: IndexerSearchCachePatchInput;
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


export type MutationUpdateSettingsSectionArgs = {
  input: UpdateSettingsSectionInput;
};


export type MutationUpdateStashBoxCacheSettingsArgs = {
  ttlHours: Scalars['Int']['input'];
};


export type MutationUpdateStashSettingsArgs = {
  input: UpdateStashSettingsInput;
};


export type MutationUpdateSystemSettingsArgs = {
  input: UpdateSystemSettingsInput;
};

export type PerformerBatchPayload = {
  __typename?: 'PerformerBatchPayload';
  batchId: Scalars['ID']['output'];
  results: Array<PerformerBatchResult>;
  summary: PerformerBatchSummary;
};

export type PerformerBatchResult = {
  __typename?: 'PerformerBatchResult';
  performer?: Maybe<StashPerformer>;
  performerId: Scalars['ID']['output'];
  reasonCode: Scalars['String']['output'];
  state?: Maybe<SubscribedPerformer>;
  status: PerformerBatchStatus;
};

export enum PerformerBatchStatus {
  Failed = 'FAILED',
  Skipped = 'SKIPPED',
  Succeeded = 'SUCCEEDED'
}

export type PerformerBatchSummary = {
  __typename?: 'PerformerBatchSummary';
  failedCount: Scalars['Int']['output'];
  requestedCount: Scalars['Int']['output'];
  skippedCount: Scalars['Int']['output'];
  succeededCount: Scalars['Int']['output'];
};

export type PerformerCheckActivity = {
  __typename?: 'PerformerCheckActivity';
  performerId: Scalars['ID']['output'];
  runId: Scalars['ID']['output'];
  status: SubscriptionCheckStatus;
};

export enum PerformerListFilter {
  All = 'ALL',
  Error = 'ERROR',
  Favorite = 'FAVORITE',
  Review = 'REVIEW',
  Subscribed = 'SUBSCRIBED'
}

export type PerformerListOptions = {
  filter?: InputMaybe<PerformerListFilter>;
  sort?: InputMaybe<PerformerListSort>;
};

export enum PerformerListSort {
  Default = 'DEFAULT',
  LastChecked = 'LAST_CHECKED',
  Name = 'NAME',
  SceneCount = 'SCENE_COUNT'
}

export type PerformerScenePerson = {
  __typename?: 'PerformerScenePerson';
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
};

export type PerformerSceneTag = {
  __typename?: 'PerformerSceneTag';
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
};

export type PerformerSceneTask = {
  __typename?: 'PerformerSceneTask';
  id: Scalars['ID']['output'];
  progress: Scalars['Float']['output'];
  stage: TaskStage;
  stageLabel: Scalars['String']['output'];
  stageStatus: TaskStageStatus;
  stageStatusLabel: Scalars['String']['output'];
};

export type PerformerSubscriptionEvent = {
  __typename?: 'PerformerSubscriptionEvent';
  performerId: Scalars['ID']['output'];
  sequence: Scalars['Int']['output'];
  state?: Maybe<SubscribedPerformer>;
  type: PerformerSubscriptionEventType;
};

export enum PerformerSubscriptionEventType {
  Created = 'CREATED',
  Deleted = 'DELETED',
  Updated = 'UPDATED'
}

export type PerformerWorkspaceSnapshot = {
  __typename?: 'PerformerWorkspaceSnapshot';
  performers: StashPerformerConnection;
  subscribedPerformers: Array<SubscribedPerformer>;
};

export type PrepareTorrentDownloadInput = {
  candidateId: Scalars['ID']['input'];
  searchId: Scalars['ID']['input'];
};

/** An explicitly acquired artifact; the URL only serves stored data and expires. */
export type PreparedTorrentDownload = {
  __typename?: 'PreparedTorrentDownload';
  downloadUrl: Scalars['String']['output'];
  expiresAt: Scalars['String']['output'];
  kind: PreparedTorrentDownloadKind;
};

export enum PreparedTorrentDownloadKind {
  Magnet = 'MAGNET',
  Torrent = 'TORRENT'
}

export type PreviewTorrentSelectionInput = {
  applyFastRules?: Scalars['Boolean']['input'];
  applyFileRules?: Scalars['Boolean']['input'];
  candidateIds?: InputMaybe<Array<Scalars['ID']['input']>>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  searchId: Scalars['ID']['input'];
  sortBy?: InputMaybe<TorrentSearchSort>;
};

export type ProcessReleaseReviewsInput = {
  action: ReleaseReviewAction;
  keys: Array<Scalars['ID']['input']>;
};

export type QbConnectionPatchInput = {
  category?: InputMaybe<Scalars['String']['input']>;
  default_save_path?: InputMaybe<Scalars['String']['input']>;
  password?: InputMaybe<Scalars['String']['input']>;
  tags?: InputMaybe<Scalars['String']['input']>;
  url?: InputMaybe<Scalars['String']['input']>;
  username?: InputMaybe<Scalars['String']['input']>;
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
  downloader: DownloaderOverview;
  health: Health;
  homeIndexerSummary: HomeIndexerSummary;
  /** Bounded task lists and mutually exclusive operational counts for Home. */
  homeOverview: HomeOverview;
  /** Bounded local check progress, schedule, and latest outcomes for Home. */
  homeSubscriptionChecks: HomeSubscriptionChecks;
  /** Persisted subscription health without an upstream performer catalogue read. */
  homeSubscriptionSummary: HomeSubscriptionSummary;
  homeTaskSnapshot: HomeTaskSnapshot;
  indexerCacheStatus: IndexerCacheStatus;
  indexerDefinitionPolicy: IndexerDefinitionPolicy;
  indexerDefinitionStatus: IndexerDefinitionStatus;
  indexerDefinitionUpdate?: Maybe<IndexerDefinitionJob>;
  indexerDefinitions: IndexerDefinitionPage;
  indexerInstances: IndexerInstances;
  indexerLogin?: Maybe<IndexerLoginReport>;
  indexerNetworkSettings: IndexerNetworkSettings;
  /** Retrieve recent Moji logs for troubleshooting */
  logs: Array<LogEntry>;
  /** Load the actor list and subscription state from one Stash snapshot */
  performerWorkspace: PerformerWorkspaceSnapshot;
  /** List torrents from qBittorrent */
  qbittorrentTorrents: Array<QbTorrent>;
  releaseReviews: ReleaseReviewConnection;
  /** Get editable configuration for the Settings surface */
  settings: Settings;
  /** Get runtime state for the Settings surface */
  settingsStatus: SettingsStatus;
  /** Get a Stash background job by id */
  stashJob?: Maybe<StashJob>;
  /** Fetch Stash performer detail with Moji / StashBox context */
  stashPerformerDetail: StashPerformerDetail;
  /** Fetch basic performer information from Stash without external metadata requests */
  stashPerformerProfile: StashPerformerProfile;
  /** List deduplicated performer scenes from Stash and the preferred StashBox */
  stashPerformerScenes: StashPerformerSceneConnection;
  /** List Stash performers with current Moji subscription state */
  stashPerformers: StashPerformerConnection;
  /** List performers currently subscribed by Moji */
  subscribedPerformers: Array<SubscribedPerformer>;
  subscriptionCheck?: Maybe<SubscriptionCheckRun>;
  /** Check reservations for at most 256 visible or selected actors; no history or upstream reads. */
  subscriptionCheckActivity: SubscriptionCheckActivity;
  /** Get a Moji download task by id */
  task?: Maybe<Task>;
  /** A stable oldest-first review session, independent of list filters/pages. */
  taskAttentionQueue: TaskAttentionQueue;
  taskDetailSnapshot: TaskDetailSnapshot;
  taskPage: TaskPage;
  /** Read current Stash information for the scenes linked to a task. */
  taskStashScenes: Array<TaskStashScene>;
  taskSummaries: Array<TaskSummary>;
  /** List Moji download tasks, newest first */
  tasks: Array<Task>;
  torrentSearch: TorrentSearch;
  upcomingReleases: UpcomingReleaseConnection;
  version: Scalars['String']['output'];
};


export type QueryDiscoverScenesArgs = {
  input: DiscoverScenesInput;
};


export type QueryDownloaderArgs = {
  attentionOnly?: InputMaybe<Scalars['Boolean']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  search?: InputMaybe<Scalars['String']['input']>;
  taskId?: InputMaybe<Scalars['ID']['input']>;
};


export type QueryIndexerDefinitionUpdateArgs = {
  id: Scalars['ID']['input'];
};


export type QueryIndexerDefinitionsArgs = {
  input?: InputMaybe<IndexerDefinitionFilterInput>;
};


export type QueryIndexerLoginArgs = {
  input: IndexerLoginInput;
};


export type QueryLogsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  minLevel?: InputMaybe<LogLevel>;
};


export type QueryPerformerWorkspaceArgs = {
  options?: InputMaybe<PerformerListOptions>;
  page?: InputMaybe<Scalars['Int']['input']>;
  pageSize?: InputMaybe<Scalars['Int']['input']>;
  search?: InputMaybe<Scalars['String']['input']>;
};


export type QueryQbittorrentTorrentsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
};


export type QueryReleaseReviewsArgs = {
  input: ReleaseReviewQueryInput;
};


export type QueryStashJobArgs = {
  id: Scalars['ID']['input'];
};


export type QueryStashPerformerDetailArgs = {
  id: Scalars['ID']['input'];
};


export type QueryStashPerformerProfileArgs = {
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


export type QuerySubscriptionCheckArgs = {
  id: Scalars['ID']['input'];
};


export type QuerySubscriptionCheckActivityArgs = {
  ids: Array<Scalars['ID']['input']>;
};


export type QueryTaskArgs = {
  id: Scalars['ID']['input'];
};


export type QueryTaskDetailSnapshotArgs = {
  id: Scalars['ID']['input'];
};


export type QueryTaskPageArgs = {
  input: TaskPageInput;
};


export type QueryTaskStashScenesArgs = {
  id: Scalars['ID']['input'];
};


export type QueryTaskSummariesArgs = {
  ids: Array<Scalars['ID']['input']>;
};


export type QueryTorrentSearchArgs = {
  id: Scalars['ID']['input'];
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  sortBy?: InputMaybe<TorrentSearchSort>;
};


export type QueryUpcomingReleasesArgs = {
  input: UpcomingReleaseQueryInput;
};

export type QueueDiscoveredSceneInput = {
  sceneId: Scalars['ID']['input'];
  stashBoxEndpoint: Scalars['String']['input'];
};

export type QueuePerformerSceneResult = {
  __typename?: 'QueuePerformerSceneResult';
  key: Scalars['ID']['output'];
  reasonCode: Scalars['String']['output'];
  resolvedCode?: Maybe<Scalars['String']['output']>;
  status: QueuePerformerSceneStatus;
  task?: Maybe<Task>;
};

export enum QueuePerformerSceneStatus {
  Failed = 'FAILED',
  Queued = 'QUEUED',
  Skipped = 'SKIPPED'
}

export type QueuePerformerScenesInput = {
  performerId: Scalars['ID']['input'];
  sceneKeys: Array<Scalars['ID']['input']>;
};

export type QueuePerformerScenesPayload = {
  __typename?: 'QueuePerformerScenesPayload';
  queuedTasks: Array<Task>;
  results: Array<QueuePerformerSceneResult>;
  summary: QueuePerformerScenesSummary;
};

export type QueuePerformerScenesSummary = {
  __typename?: 'QueuePerformerScenesSummary';
  failedCount: Scalars['Int']['output'];
  queuedCount: Scalars['Int']['output'];
  requestedCount: Scalars['Int']['output'];
  skippedCount: Scalars['Int']['output'];
};

export type QueueTorrentCandidateInput = {
  candidateId: Scalars['ID']['input'];
  category?: InputMaybe<Scalars['String']['input']>;
  code: Scalars['String']['input'];
  paused?: InputMaybe<Scalars['Boolean']['input']>;
  savePath?: InputMaybe<Scalars['String']['input']>;
  searchId: Scalars['ID']['input'];
  tags?: InputMaybe<Scalars['String']['input']>;
};

export type ReleasePolicyPatchInput = {
  compilation_behavior?: InputMaybe<SubscriptionReleaseBehavior>;
  group_behavior?: InputMaybe<SubscriptionReleaseBehavior>;
  max_group_performer_count?: InputMaybe<Scalars['Int']['input']>;
  release_date_range?: InputMaybe<SubscriptionReleaseDateRange>;
  solo_behavior?: InputMaybe<SubscriptionReleaseBehavior>;
  vr_behavior?: InputMaybe<SubscriptionReleaseBehavior>;
};

export type ReleaseReview = {
  __typename?: 'ReleaseReview';
  code: Scalars['String']['output'];
  date?: Maybe<Scalars['String']['output']>;
  error?: Maybe<Scalars['String']['output']>;
  imageUrl?: Maybe<Scalars['String']['output']>;
  key: Scalars['ID']['output'];
  performerNames: Array<Scalars['String']['output']>;
  performers: Array<ReleaseReviewPerformer>;
  reasonCode: Scalars['String']['output'];
  reviewedAt?: Maybe<Scalars['String']['output']>;
  seenAt: Scalars['String']['output'];
  source: Scalars['String']['output'];
  status: ReleaseReviewStatus;
  taskId?: Maybe<Scalars['ID']['output']>;
  title: Scalars['String']['output'];
  url?: Maybe<Scalars['String']['output']>;
  vr: Scalars['Boolean']['output'];
};

export enum ReleaseReviewAction {
  Download = 'DOWNLOAD',
  Reconsider = 'RECONSIDER',
  Reject = 'REJECT'
}

export type ReleaseReviewConnection = {
  __typename?: 'ReleaseReviewConnection';
  failedCount: Scalars['Int']['output'];
  items: Array<ReleaseReview>;
  page: Scalars['Int']['output'];
  pageSize: Scalars['Int']['output'];
  pendingCount: Scalars['Int']['output'];
  performers: Array<ReleaseReviewPerformer>;
  totalCount: Scalars['Int']['output'];
  totalPages: Scalars['Int']['output'];
};

export type ReleaseReviewPayload = {
  __typename?: 'ReleaseReviewPayload';
  failedCount: Scalars['Int']['output'];
  results: Array<ReleaseReviewResult>;
  skippedCount: Scalars['Int']['output'];
  succeededCount: Scalars['Int']['output'];
};

export type ReleaseReviewPerformer = {
  __typename?: 'ReleaseReviewPerformer';
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
  pendingCount: Scalars['Int']['output'];
};

export type ReleaseReviewQueryInput = {
  page?: InputMaybe<Scalars['Int']['input']>;
  pageSize?: InputMaybe<Scalars['Int']['input']>;
  performerId?: InputMaybe<Scalars['ID']['input']>;
  search?: InputMaybe<Scalars['String']['input']>;
  status?: InputMaybe<ReleaseReviewStatus>;
  vr?: InputMaybe<Scalars['Boolean']['input']>;
};

export type ReleaseReviewResult = {
  __typename?: 'ReleaseReviewResult';
  key: Scalars['ID']['output'];
  outcome: Scalars['String']['output'];
  reasonCode: Scalars['String']['output'];
  release?: Maybe<ReleaseReview>;
};

export enum ReleaseReviewStatus {
  All = 'ALL',
  Downloaded = 'DOWNLOADED',
  Failed = 'FAILED',
  InLibrary = 'IN_LIBRARY',
  Pending = 'PENDING',
  Rejected = 'REJECTED',
  Skipped = 'SKIPPED'
}

export type ResolveBlockedSourcingTaskInput = {
  infoHash?: InputMaybe<Scalars['String']['input']>;
  paused?: InputMaybe<Scalars['Boolean']['input']>;
  peers?: InputMaybe<Scalars['Int']['input']>;
  seeders?: InputMaybe<Scalars['Int']['input']>;
  size?: InputMaybe<Scalars['Long']['input']>;
  title?: InputMaybe<Scalars['String']['input']>;
  torrentFile?: InputMaybe<Scalars['Upload']['input']>;
  /** Provide exactly one URL or uploaded torrent file. */
  torrentUrl?: InputMaybe<Scalars['String']['input']>;
  tracker?: InputMaybe<Scalars['String']['input']>;
};

export type ResolveTaskSourcingCandidateInput = {
  candidateId: Scalars['ID']['input'];
  expectedRevision: Scalars['Long']['input'];
  paused?: InputMaybe<Scalars['Boolean']['input']>;
  searchId: Scalars['ID']['input'];
  taskId: Scalars['ID']['input'];
};

export type ResourceRulesPatchInput = {
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
  fast_rule_order?: InputMaybe<Array<TorrentSelectionRuleType>>;
  fast_rules?: InputMaybe<FastRulesPatchInput>;
  inspection_candidate_limit?: InputMaybe<Scalars['Int']['input']>;
  torrent_rules?: InputMaybe<FileRulesPatchInput>;
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

export enum ServiceConnectionTestCode {
  Failed = 'FAILED',
  InvalidUrl = 'INVALID_URL',
  Ok = 'OK',
  Timeout = 'TIMEOUT'
}

export type ServiceConnectionTestResult = {
  __typename?: 'ServiceConnectionTestResult';
  code: ServiceConnectionTestCode;
};

export type ServiceStatus = {
  __typename?: 'ServiceStatus';
  /** True iff the minimum connection fields are present, so the backend can attempt to talk to the upstream service. */
  configured: Scalars['Boolean']['output'];
  /** True iff the upstream service is configured AND a recent probe succeeded. A probe result older than ~4 minutes, or a failed probe, returns false. The proximate cause of a non-ready state lives on the corresponding *Stats.lastError. */
  ready: Scalars['Boolean']['output'];
};

export type ServiceStatusEvent = {
  __typename?: 'ServiceStatusEvent';
  observedAt: Scalars['String']['output'];
  sequence: Scalars['Int']['output'];
  services: Array<ExternalService>;
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

export enum SettingsConnectionService {
  Jackett = 'JACKETT',
  Qbittorrent = 'QBITTORRENT',
  Stash = 'STASH'
}

export type SettingsPatchInput = {
  downloader?: InputMaybe<DownloaderConnectionPatchInput>;
  indexers?: InputMaybe<IndexersPatchInput>;
  stash?: InputMaybe<StashPatchInput>;
  system?: InputMaybe<SystemPatchInput>;
  tasks?: InputMaybe<TasksPatchInput>;
};

export enum SettingsSection {
  All = 'ALL',
  Indexers = 'INDEXERS',
  Metadata = 'METADATA',
  Resources = 'RESOURCES',
  Tasks = 'TASKS',
  TaskDeletion = 'TASK_DELETION'
}

export type SettingsStatus = {
  __typename?: 'SettingsStatus';
  automation: AutomationStatus;
  imageCache: ImageCacheStatus;
  ingest: IngestStatus;
  jackett: ServiceStatus;
  /** Runtime stats for the Jackett indexer aggregator. Refreshed by the stats collector. */
  jackettStats: JackettStats;
  qbittorrent: ServiceStatus;
  /** Runtime stats for the qBittorrent download client. Refreshed by the stats collector. */
  qbittorrentStats: QBittorrentStats;
  stash: ServiceStatus;
  stashBox: StashBoxStatus;
  stashBoxDataCache: StashBoxDataCacheStatus;
  stashLibraries: Array<StashLibrary>;
  stashLibrariesLoadError?: Maybe<Scalars['String']['output']>;
  /** Runtime stats for the Stash server. Refreshed by the stats collector. */
  stashStats: StashStats;
};

export type StartTorrentSearchInput = {
  categories?: InputMaybe<Array<Scalars['Int']['input']>>;
  instanceIds?: InputMaybe<Array<Scalars['ID']['input']>>;
  keywords: Scalars['String']['input'];
  limit?: InputMaybe<Scalars['Int']['input']>;
  mode?: InputMaybe<Scalars['String']['input']>;
  parameters?: InputMaybe<Array<TorrentSearchParameterInput>>;
  refresh?: InputMaybe<Scalars['Boolean']['input']>;
  taskId?: InputMaybe<Scalars['ID']['input']>;
};

export type StashBoxDataCacheSettings = {
  __typename?: 'StashBoxDataCacheSettings';
  ttlHours: Scalars['Int']['output'];
};

export type StashBoxDataCacheSettingsInput = {
  ttlHours: Scalars['Int']['input'];
};

export type StashBoxDataCacheStatus = {
  __typename?: 'StashBoxDataCacheStatus';
  databasePath: Scalars['String']['output'];
  lastCleanupAt?: Maybe<Scalars['String']['output']>;
  lastError?: Maybe<Scalars['String']['output']>;
  performerCount: Scalars['Int']['output'];
  sceneCount: Scalars['Int']['output'];
  snapshotCount: Scalars['Int']['output'];
  usedBytes: Scalars['Long']['output'];
};

export type StashBoxEndpoint = {
  __typename?: 'StashBoxEndpoint';
  apiKeyConfigured: Scalars['Boolean']['output'];
  endpoint: Scalars['String']['output'];
  name: Scalars['String']['output'];
};

export type StashBoxStatus = {
  __typename?: 'StashBoxStatus';
  /** Stash Box instances currently configured inside the Stash server. */
  stashBoxes: Array<StashBoxEndpoint>;
  /** Reason for the most recent Stash Box load failure. Null when stashBoxesLoaded is true. */
  stashBoxesLoadError?: Maybe<Scalars['String']['output']>;
  /** Whether the last attempt to load Stash Box endpoints from Stash succeeded. */
  stashBoxesLoaded: Scalars['Boolean']['output'];
};

export type StashCachePatchInput = {
  ttl_hours?: InputMaybe<Scalars['Int']['input']>;
};

export type StashConnectionPatchInput = {
  api_key?: InputMaybe<Scalars['String']['input']>;
  url?: InputMaybe<Scalars['String']['input']>;
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

export type StashPatchInput = {
  connection?: InputMaybe<StashConnectionPatchInput>;
  data_cache?: InputMaybe<StashCachePatchInput>;
  ingest?: InputMaybe<DeliveryPatchInput>;
  metadata_priority?: InputMaybe<Array<Scalars['String']['input']>>;
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
  /** Unfinished tasks associated with loaded performer scenes, counted once per task */
  activeTaskCount: Scalars['Int']['output'];
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
  /** Whether the source catalog used for the task count is fully loaded */
  sceneStatsComplete: Scalars['Boolean']['output'];
  /** Whether the displayed source snapshot is awaiting background refresh */
  sceneStatsStale: Scalars['Boolean']['output'];
  stashBoxSceneCount: Scalars['Int']['output'];
  stashSceneCount: Scalars['Int']['output'];
  totalSceneCount: Scalars['Int']['output'];
  urls: Array<Scalars['String']['output']>;
};

export type StashPerformerProfile = {
  __typename?: 'StashPerformerProfile';
  birthdate?: Maybe<Scalars['String']['output']>;
  country?: Maybe<Scalars['String']['output']>;
  disambiguation?: Maybe<Scalars['String']['output']>;
  ethnicity?: Maybe<Scalars['String']['output']>;
  eyeColor?: Maybe<Scalars['String']['output']>;
  heightCm?: Maybe<Scalars['Int']['output']>;
  performer: StashPerformer;
  rating100?: Maybe<Scalars['Int']['output']>;
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
  mojiTask?: Maybe<PerformerSceneTask>;
  performerCount: Scalars['Int']['output'];
  performers: Array<PerformerScenePerson>;
  primarySource: SceneSource;
  sourceLabels: Array<Scalars['String']['output']>;
  sourceSceneId: Scalars['ID']['output'];
  stashBoxEndpoint?: Maybe<Scalars['String']['output']>;
  stashBoxSceneId?: Maybe<Scalars['ID']['output']>;
  stashIds: Array<StashSceneId>;
  studioName?: Maybe<Scalars['String']['output']>;
  tagCount: Scalars['Int']['output'];
  tags: Array<PerformerSceneTag>;
  title?: Maybe<Scalars['String']['output']>;
  upcomingState?: Maybe<Scalars['String']['output']>;
  url?: Maybe<Scalars['String']['output']>;
};

export type StashPerformerSceneConnection = {
  __typename?: 'StashPerformerSceneConnection';
  cacheComplete: Scalars['Boolean']['output'];
  cacheStale: Scalars['Boolean']['output'];
  cacheUpdatedAt?: Maybe<Scalars['String']['output']>;
  dedupedCount: Scalars['Int']['output'];
  hasNextPage: Scalars['Boolean']['output'];
  hasPrevPage: Scalars['Boolean']['output'];
  items: Array<StashPerformerScene>;
  page: Scalars['Int']['output'];
  pageSize: Scalars['Int']['output'];
  stashBoxCount: Scalars['Int']['output'];
  stashBoxLoadedCount: Scalars['Int']['output'];
  stashBoxRemoteCount: Scalars['Int']['output'];
  stashSceneCount: Scalars['Int']['output'];
  totalCount: Scalars['Int']['output'];
  totalCountExact: Scalars['Boolean']['output'];
  totalPages?: Maybe<Scalars['Int']['output']>;
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
  reviewReleaseCount: Scalars['Int']['output'];
};

export type Subscription = {
  __typename?: 'Subscription';
  /** Latest database metrics pushed by the live owner. Available only while developer mode is enabled. */
  databasePerformance: DatabasePerformanceSnapshot;
  indexerCheckActivity: Array<IndexerCheckActivity>;
  indexerDefinitionEvents: IndexerDefinitionStatus;
  indexerEvents: IndexerEvent;
  /** Stream new in-memory Moji log entries */
  logEvents: LogEvent;
  performerSubscriptionEvents: PerformerSubscriptionEvent;
  serviceStatusEvents: ServiceStatusEvent;
  taskEvents: TaskEvent;
  torrentSearchEvents: TorrentSearchEvent;
};


export type SubscriptionIndexerCheckActivityArgs = {
  clientId: Scalars['ID']['input'];
};


export type SubscriptionTorrentSearchEventsArgs = {
  id: Scalars['ID']['input'];
};

export type SubscriptionCheckActivity = {
  __typename?: 'SubscriptionCheckActivity';
  activeCount: Scalars['Int']['output'];
  allRunId?: Maybe<Scalars['ID']['output']>;
  performers: Array<PerformerCheckActivity>;
};

export type SubscriptionCheckItem = {
  __typename?: 'SubscriptionCheckItem';
  error?: Maybe<Scalars['String']['output']>;
  finishedAt?: Maybe<Scalars['String']['output']>;
  name: Scalars['String']['output'];
  performerId: Scalars['ID']['output'];
  startedAt?: Maybe<Scalars['String']['output']>;
  status: SubscriptionCheckStatus;
};

export type SubscriptionCheckRun = {
  __typename?: 'SubscriptionCheckRun';
  completedCount: Scalars['Int']['output'];
  error?: Maybe<Scalars['String']['output']>;
  failedCount: Scalars['Int']['output'];
  finishedAt?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  items: Array<SubscriptionCheckItem>;
  scope: SubscriptionCheckScope;
  skippedCount: Scalars['Int']['output'];
  startedAt: Scalars['String']['output'];
  status: SubscriptionCheckStatus;
  succeededCount: Scalars['Int']['output'];
  totalCount: Scalars['Int']['output'];
  trigger: SubscriptionCheckTrigger;
};

export enum SubscriptionCheckScope {
  All = 'ALL',
  Selected = 'SELECTED',
  Single = 'SINGLE'
}

export enum SubscriptionCheckStatus {
  Failed = 'FAILED',
  Interrupted = 'INTERRUPTED',
  Partial = 'PARTIAL',
  Pending = 'PENDING',
  Running = 'RUNNING',
  Skipped = 'SKIPPED',
  Succeeded = 'SUCCEEDED'
}

export enum SubscriptionCheckTrigger {
  Manual = 'MANUAL',
  Scheduled = 'SCHEDULED'
}

export type SubscriptionRelease = {
  __typename?: 'SubscriptionRelease';
  code?: Maybe<Scalars['String']['output']>;
  date?: Maybe<Scalars['String']['output']>;
  key: Scalars['ID']['output'];
  performerCount: Scalars['Int']['output'];
  performerNames: Array<Scalars['String']['output']>;
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

export enum SubscriptionReleaseDateRange {
  All = 'ALL',
  FiveYears = 'FIVE_YEARS',
  OneYear = 'ONE_YEAR',
  ThreeYears = 'THREE_YEARS',
  TwoYears = 'TWO_YEARS'
}

export type SubscriptionReleasePolicy = {
  __typename?: 'SubscriptionReleasePolicy';
  compilationBehavior: SubscriptionReleaseBehavior;
  groupBehavior: SubscriptionReleaseBehavior;
  maxGroupPerformerCount: Scalars['Int']['output'];
  releaseDateRange: SubscriptionReleaseDateRange;
  soloBehavior: SubscriptionReleaseBehavior;
  vrBehavior: SubscriptionReleaseBehavior;
};

export type SubscriptionReleasePolicyInput = {
  compilationBehavior: SubscriptionReleaseBehavior;
  groupBehavior: SubscriptionReleaseBehavior;
  maxGroupPerformerCount: Scalars['Int']['input'];
  releaseDateRange: SubscriptionReleaseDateRange;
  soloBehavior: SubscriptionReleaseBehavior;
  /** Omit to preserve the current VR policy. VR is identified only by VR in the release code. */
  vrBehavior?: InputMaybe<SubscriptionReleaseBehavior>;
};

export type SystemPatchInput = {
  developer_mode?: InputMaybe<Scalars['Boolean']['input']>;
  image_cache?: InputMaybe<ImageCachePatchInput>;
  language?: InputMaybe<Scalars['String']['input']>;
  release_timezone?: InputMaybe<Scalars['String']['input']>;
};

export type SystemSettings = {
  __typename?: 'SystemSettings';
  developerMode: Scalars['Boolean']['output'];
  effectiveReleaseTimezone: Scalars['String']['output'];
  imageCache: ImageCacheSettings;
  language: Scalars['String']['output'];
  releaseTimezone: Scalars['String']['output'];
  stashBoxDataCache: StashBoxDataCacheSettings;
  taskDeletePolicy: TaskDeletePolicy;
};

export type Task = {
  __typename?: 'Task';
  candidate: DownloadCandidate;
  category?: Maybe<Scalars['String']['output']>;
  code: Scalars['String']['output'];
  contentPath?: Maybe<Scalars['String']['output']>;
  createdAt: Scalars['String']['output'];
  downloadCompletedAt?: Maybe<Scalars['String']['output']>;
  files: Array<TaskFile>;
  id: Scalars['ID']['output'];
  /** Time when all ingest steps passed verification; null when not recorded. */
  ingestCompletedAt?: Maybe<Scalars['String']['output']>;
  /** Last observed execution change, excluding unchanged polling; null for untracked history. */
  lastProgressAt?: Maybe<Scalars['String']['output']>;
  mojiSourcePath?: Maybe<Scalars['String']['output']>;
  mojiTransferPath?: Maybe<Scalars['String']['output']>;
  progress: Scalars['Float']['output'];
  qbittorrentState?: Maybe<Scalars['String']['output']>;
  revision: Scalars['Long']['output'];
  savePath?: Maybe<Scalars['String']['output']>;
  source: TaskSource;
  /** Time when a selected torrent was successfully submitted to the downloader; null when not recorded. */
  sourcingCompletedAt?: Maybe<Scalars['String']['output']>;
  /** Last automatic resource search; persists across restarts. */
  sourcingLastCheckedAt?: Maybe<Scalars['String']['output']>;
  /** Next scheduled resource search; null when no search is scheduled. */
  sourcingNextCheckAt?: Maybe<Scalars['String']['output']>;
  stage: TaskStage;
  stageErrorCode?: Maybe<Scalars['String']['output']>;
  stageErrorMessage?: Maybe<Scalars['String']['output']>;
  stageLabel: Scalars['String']['output'];
  stageStatus: TaskStageStatus;
  stageStatusLabel: Scalars['String']['output'];
  stashGenerateJobId?: Maybe<Scalars['ID']['output']>;
  stashIdentifyJobId?: Maybe<Scalars['ID']['output']>;
  /** Current Stash ingest step: SCAN, GENERATE, IDENTIFY, or VERIFY */
  stashIngestPhase?: Maybe<Scalars['String']['output']>;
  stashScanError?: Maybe<Scalars['String']['output']>;
  stashScanHint?: Maybe<Scalars['String']['output']>;
  stashScanJobId?: Maybe<Scalars['ID']['output']>;
  stashScanPath?: Maybe<Scalars['String']['output']>;
  stashScanStartedAt?: Maybe<Scalars['String']['output']>;
  stashSceneIds: Array<Scalars['ID']['output']>;
  tags?: Maybe<Scalars['String']['output']>;
  torrentHash?: Maybe<Scalars['String']['output']>;
  torrentName?: Maybe<Scalars['String']['output']>;
  torrentUrl: Scalars['String']['output'];
  transferAction?: Maybe<Scalars['String']['output']>;
  transferError?: Maybe<Scalars['String']['output']>;
  updatedAt: Scalars['String']['output'];
  version: Scalars['String']['output'];
};

export type TaskAttentionQueue = {
  __typename?: 'TaskAttentionQueue';
  items: Array<TaskSummary>;
  sequence: Scalars['Int']['output'];
  streamId: Scalars['String']['output'];
};

export type TaskBatchPayload = {
  __typename?: 'TaskBatchPayload';
  batchId: Scalars['ID']['output'];
  results: Array<TaskBatchResult>;
  summary: TaskBatchSummary;
};

export type TaskBatchResult = {
  __typename?: 'TaskBatchResult';
  reasonCode: Scalars['String']['output'];
  status: TaskBatchStatus;
  task?: Maybe<Task>;
  taskId: Scalars['ID']['output'];
};

export enum TaskBatchStatus {
  Failed = 'FAILED',
  Skipped = 'SKIPPED',
  Succeeded = 'SUCCEEDED'
}

export type TaskBatchSummary = {
  __typename?: 'TaskBatchSummary';
  failedCount: Scalars['Int']['output'];
  requestedCount: Scalars['Int']['output'];
  skippedCount: Scalars['Int']['output'];
  succeededCount: Scalars['Int']['output'];
};

export enum TaskDeletePolicy {
  KeepOnly = 'KEEP_ONLY',
  RemoveTorrent = 'REMOVE_TORRENT',
  RemoveTorrentAndFiles = 'REMOVE_TORRENT_AND_FILES'
}

export type TaskDetailSnapshot = {
  __typename?: 'TaskDetailSnapshot';
  sequence: Scalars['Int']['output'];
  streamId: Scalars['String']['output'];
  task?: Maybe<Task>;
};

export type TaskEvent = {
  __typename?: 'TaskEvent';
  dashboardStats?: Maybe<DashboardStats>;
  listChanged: Scalars['Boolean']['output'];
  sequence: Scalars['Int']['output'];
  streamId: Scalars['String']['output'];
  summary?: Maybe<TaskSummary>;
  task?: Maybe<Task>;
  taskId: Scalars['ID']['output'];
  type: TaskEventType;
};

export enum TaskEventType {
  Created = 'CREATED',
  Deleted = 'DELETED',
  Ready = 'READY',
  Updated = 'UPDATED'
}

/** A classified qBittorrent file; order applies to selected main video parts */
export type TaskFile = {
  __typename?: 'TaskFile';
  index: Scalars['Int']['output'];
  kind: Scalars['String']['output'];
  part: Scalars['Int']['output'];
  path: Scalars['String']['output'];
  reason: Scalars['String']['output'];
  selected: Scalars['Boolean']['output'];
  size: Scalars['Long']['output'];
};

export enum TaskListSort {
  Created = 'CREATED',
  Progress = 'PROGRESS',
  Updated = 'UPDATED'
}

export enum TaskListStatus {
  Active = 'ACTIVE',
  All = 'ALL',
  Attention = 'ATTENTION',
  Completed = 'COMPLETED',
  Waiting = 'WAITING'
}

export type TaskPage = {
  __typename?: 'TaskPage';
  items: Array<TaskSummary>;
  page: Scalars['Int']['output'];
  pageSize: Scalars['Int']['output'];
  sequence: Scalars['Int']['output'];
  stats: DashboardStats;
  streamId: Scalars['String']['output'];
  totalCount: Scalars['Int']['output'];
  totalPages: Scalars['Int']['output'];
};

export type TaskPageInput = {
  completedWithin24Hours?: InputMaybe<Scalars['Boolean']['input']>;
  ids?: InputMaybe<Array<Scalars['ID']['input']>>;
  page?: InputMaybe<Scalars['Int']['input']>;
  pageSize?: InputMaybe<Scalars['Int']['input']>;
  search?: InputMaybe<Scalars['String']['input']>;
  sort?: InputMaybe<TaskListSort>;
  status?: InputMaybe<TaskListStatus>;
};

export type TaskSettingsInput = {
  sourcingRetryIntervalHours: Scalars['Int']['input'];
  subscriptionPollIntervalHours: Scalars['Int']['input'];
  subscriptionReleasePolicy: SubscriptionReleasePolicyInput;
  taskProcessingEnabled: Scalars['Boolean']['input'];
  taskProgressSyncIntervalSeconds: Scalars['Int']['input'];
};

export enum TaskSource {
  Manual = 'MANUAL',
  Search = 'SEARCH',
  Subscription = 'SUBSCRIPTION'
}

export enum TaskStage {
  Completed = 'COMPLETED',
  Downloading = 'DOWNLOADING',
  PendingIngest = 'PENDING_INGEST',
  Scanning = 'SCANNING',
  Sourcing = 'SOURCING',
  Transferring = 'TRANSFERRING'
}

export enum TaskStageStatus {
  Blocked = 'BLOCKED',
  Done = 'DONE',
  Pending = 'PENDING',
  Running = 'RUNNING'
}

/** A Stash scene linked to a task's ingest result. */
export type TaskStashScene = {
  __typename?: 'TaskStashScene';
  code: Scalars['String']['output'];
  date: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  imageUrl?: Maybe<Scalars['String']['output']>;
  performers: Array<Scalars['String']['output']>;
  studio: Scalars['String']['output'];
  title: Scalars['String']['output'];
  url?: Maybe<Scalars['String']['output']>;
};

/** Small list/event projection. File plans, candidates and paths belong to task details. */
export type TaskSummary = {
  __typename?: 'TaskSummary';
  code: Scalars['String']['output'];
  createdAt: Scalars['String']['output'];
  downloadCompletedAt?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  ingestCompletedAt?: Maybe<Scalars['String']['output']>;
  lastProgressAt?: Maybe<Scalars['String']['output']>;
  progress: Scalars['Float']['output'];
  qbittorrentState?: Maybe<Scalars['String']['output']>;
  revision: Scalars['Long']['output'];
  source: TaskSource;
  sourcingCompletedAt?: Maybe<Scalars['String']['output']>;
  sourcingLastCheckedAt?: Maybe<Scalars['String']['output']>;
  sourcingNextCheckAt?: Maybe<Scalars['String']['output']>;
  stage: TaskStage;
  stageErrorCode?: Maybe<Scalars['String']['output']>;
  stageErrorMessage?: Maybe<Scalars['String']['output']>;
  stageLabel: Scalars['String']['output'];
  stageStatus: TaskStageStatus;
  stageStatusLabel: Scalars['String']['output'];
  stashIngestPhase?: Maybe<Scalars['String']['output']>;
  stashSceneId?: Maybe<Scalars['ID']['output']>;
  torrentName?: Maybe<Scalars['String']['output']>;
  updatedAt: Scalars['String']['output'];
  version: Scalars['String']['output'];
};

export type TasksPatchInput = {
  sourcing_retry_interval_hours?: InputMaybe<Scalars['Int']['input']>;
  subscription_poll_interval_hours?: InputMaybe<Scalars['Int']['input']>;
  subscription_release_policy?: InputMaybe<ReleasePolicyPatchInput>;
  task_delete_policy?: InputMaybe<TaskDeletePolicy>;
  task_processing_enabled?: InputMaybe<Scalars['Boolean']['input']>;
  task_progress_sync_interval_seconds?: InputMaybe<Scalars['Int']['input']>;
  torrent_selection?: InputMaybe<ResourceRulesPatchInput>;
};

export type TestIndexerInput = {
  clientId?: InputMaybe<Scalars['ID']['input']>;
  connectionOnly?: Scalars['Boolean']['input'];
  definitionId?: InputMaybe<Scalars['String']['input']>;
  download?: Scalars['Boolean']['input'];
  draft?: InputMaybe<BuiltinIndexerPatchInput>;
  expectedConnectionRevision?: InputMaybe<Scalars['String']['input']>;
  expectedRevision: Scalars['String']['input'];
  instanceId?: InputMaybe<Scalars['ID']['input']>;
  keywords?: InputMaybe<Scalars['String']['input']>;
  requestId?: InputMaybe<Scalars['ID']['input']>;
  search?: Scalars['Boolean']['input'];
};

export type TestServiceConnectionInput = {
  apiKey?: InputMaybe<Scalars['String']['input']>;
  password?: InputMaybe<Scalars['String']['input']>;
  service: SettingsConnectionService;
  url: Scalars['String']['input'];
  username?: InputMaybe<Scalars['String']['input']>;
};

export type TitleClausePatchInput = {
  effect: TitleMatchEffect;
  pattern: Scalars['String']['input'];
  pattern_mode: TitleMatchPatternMode;
};

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
  Demote = 'DEMOTE',
  Exclude = 'EXCLUDE',
  Lock = 'LOCK',
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

export type TitleRulePatchInput = {
  clauses?: InputMaybe<Array<TitleClausePatchInput>>;
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
};

export type ToggleRulePatchInput = {
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
};

export enum TorrentAction {
  Pause = 'PAUSE',
  Reannounce = 'REANNOUNCE',
  Recheck = 'RECHECK',
  Remove = 'REMOVE',
  RemoveFiles = 'REMOVE_FILES',
  Resume = 'RESUME'
}

export enum TorrentFileMatchEffect {
  Avoid = 'AVOID',
  Demote = 'DEMOTE',
  Exclude = 'EXCLUDE',
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

/** A persisted, bounded search snapshot. Events invalidate this snapshot. */
export type TorrentSearch = {
  __typename?: 'TorrentSearch';
  cancelRequested: Scalars['Boolean']['output'];
  candidates: Array<TorrentSearchCandidate>;
  correlationId: Scalars['String']['output'];
  createdAt: Scalars['String']['output'];
  errorCode?: Maybe<Scalars['String']['output']>;
  expiresAt: Scalars['String']['output'];
  finishedAt?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  keywords: Scalars['String']['output'];
  limit: Scalars['Int']['output'];
  offset: Scalars['Int']['output'];
  partial: Scalars['Boolean']['output'];
  revision: Scalars['Long']['output'];
  sites: Array<TorrentSearchSite>;
  state: TorrentSearchState;
  taskId?: Maybe<Scalars['ID']['output']>;
  total: Scalars['Int']['output'];
  truncated: Scalars['Boolean']['output'];
  updatedAt: Scalars['String']['output'];
};

/** Only this server-owned candidate ID may be selected; locators and credentials are private. */
export type TorrentSearchCandidate = {
  __typename?: 'TorrentSearchCandidate';
  categories: Array<Scalars['Int']['output']>;
  detailsUrl?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  infoHash?: Maybe<Scalars['String']['output']>;
  instanceId: Scalars['ID']['output'];
  instanceName: Scalars['String']['output'];
  kind: IndexerSourceKind;
  peers?: Maybe<Scalars['Int']['output']>;
  publishedAt?: Maybe<Scalars['String']['output']>;
  seeders?: Maybe<Scalars['Int']['output']>;
  size?: Maybe<Scalars['Long']['output']>;
  title: Scalars['String']['output'];
};

export type TorrentSearchEvent = {
  __typename?: 'TorrentSearchEvent';
  searchId: Scalars['ID']['output'];
  sequence: Scalars['Long']['output'];
  streamId: Scalars['String']['output'];
};

/**
 * Case-insensitive structured query fields. Unknown names, duplicate case aliases,
 * top-level fields and derived variables are rejected before accepting a search.
 */
export type TorrentSearchParameterInput = {
  key: Scalars['String']['input'];
  value: Scalars['String']['input'];
};

export type TorrentSearchSite = {
  __typename?: 'TorrentSearchSite';
  cacheHit: Scalars['Boolean']['output'];
  correlationId?: Maybe<Scalars['String']['output']>;
  elapsedMs: Scalars['Long']['output'];
  errorCode?: Maybe<Scalars['String']['output']>;
  fetchedAt?: Maybe<Scalars['String']['output']>;
  instanceId: Scalars['ID']['output'];
  kind?: Maybe<IndexerSourceKind>;
  name: Scalars['String']['output'];
  partial: Scalars['Boolean']['output'];
  results: Scalars['Int']['output'];
  retryAt?: Maybe<Scalars['String']['output']>;
  state: TorrentSearchSiteState;
  truncated: Scalars['Boolean']['output'];
};

export enum TorrentSearchSiteState {
  Cancelled = 'CANCELLED',
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Interrupted = 'INTERRUPTED',
  Queued = 'QUEUED',
  Running = 'RUNNING',
  Skipped = 'SKIPPED'
}

export enum TorrentSearchSort {
  DateDesc = 'DATE_DESC',
  Relevance = 'RELEVANCE',
  SeedersDesc = 'SEEDERS_DESC',
  SizeDesc = 'SIZE_DESC'
}

export enum TorrentSearchState {
  Cancelled = 'CANCELLED',
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Interrupted = 'INTERRUPTED',
  Partial = 'PARTIAL',
  Queued = 'QUEUED',
  Running = 'RUNNING'
}

export enum TorrentSelectionDirection {
  Asc = 'ASC',
  Desc = 'DESC'
}

export type TorrentSelectionPreview = {
  __typename?: 'TorrentSelectionPreview';
  candidates: Array<TorrentSearchCandidate>;
  limit: Scalars['Int']['output'];
  meta: TorrentSelectionPreviewMeta;
  offset: Scalars['Int']['output'];
  searchId: Scalars['ID']['output'];
  searchRevision: Scalars['Long']['output'];
  total: Scalars['Int']['output'];
};

export type TorrentSelectionPreviewMeta = {
  __typename?: 'TorrentSelectionPreviewMeta';
  appliedFastRules: Scalars['Boolean']['output'];
  appliedFileRules: Scalars['Boolean']['output'];
  inspectableCount: Scalars['Int']['output'];
  inspectedCount: Scalars['Int']['output'];
  inspectionCandidateLimit: Scalars['Int']['output'];
};

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

export type TransferPatchInput = {
  action?: InputMaybe<Scalars['String']['input']>;
};

export type UpcomingPerformer = {
  __typename?: 'UpcomingPerformer';
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
};

export type UpcomingRelease = {
  __typename?: 'UpcomingRelease';
  code: Scalars['String']['output'];
  date: Scalars['String']['output'];
  decision: Scalars['String']['output'];
  errorCode: Scalars['String']['output'];
  imageUrl: Scalars['String']['output'];
  key: Scalars['ID']['output'];
  metadataCheckedAt: Scalars['String']['output'];
  performers: Array<UpcomingPerformer>;
  sourceUrl: Scalars['String']['output'];
  state: Scalars['String']['output'];
  title: Scalars['String']['output'];
};

export type UpcomingReleaseConnection = {
  __typename?: 'UpcomingReleaseConnection';
  automatic: Scalars['Boolean']['output'];
  items: Array<UpcomingRelease>;
  page: Scalars['Int']['output'];
  pageSize: Scalars['Int']['output'];
  performers: Array<UpcomingPerformer>;
  subscribedCount: Scalars['Int']['output'];
  timezone: Scalars['String']['output'];
  today: Scalars['String']['output'];
  totalCount: Scalars['Int']['output'];
  totalPages: Scalars['Int']['output'];
};

export type UpcomingReleaseQueryInput = {
  days?: InputMaybe<Scalars['Int']['input']>;
  page?: InputMaybe<Scalars['Int']['input']>;
  pageSize?: InputMaybe<Scalars['Int']['input']>;
  performerId?: InputMaybe<Scalars['ID']['input']>;
  search?: InputMaybe<Scalars['String']['input']>;
};

export type UpdateAutomationSettingsInput = {
  section?: InputMaybe<AutomationSettingsSection>;
  sourcingRetryIntervalHours?: Scalars['Int']['input'];
  stashBoxEndpoints: Array<Scalars['String']['input']>;
  subscriptionPollIntervalHours: Scalars['Int']['input'];
  subscriptionReleasePolicy: SubscriptionReleasePolicyInput;
  /** Independent switch; omitted by legacy clients to preserve the current setting */
  taskProcessingEnabled?: InputMaybe<Scalars['Boolean']['input']>;
  taskProgressSyncIntervalSeconds: Scalars['Int']['input'];
  torrentSelection: TorrentSelectionSettingsInput;
};

export type UpdateIngestSettingsInput = {
  downloads: DownloadsIngestSettingsInput;
  generateTranscodes?: InputMaybe<Scalars['Boolean']['input']>;
  keepSubtitles?: InputMaybe<Scalars['Boolean']['input']>;
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

export type UpdateSettingsSectionInput = {
  metadataPriority?: InputMaybe<Array<Scalars['String']['input']>>;
  queryOrder?: InputMaybe<Array<Scalars['String']['input']>>;
  resources?: InputMaybe<TorrentSelectionSettingsInput>;
  section: SettingsSection;
  taskDeletePolicy?: InputMaybe<TaskDeletePolicy>;
  tasks?: InputMaybe<TaskSettingsInput>;
};

export type UpdateStashSettingsInput = {
  apiKey?: InputMaybe<Scalars['String']['input']>;
  url: Scalars['String']['input'];
};

export type UpdateSystemSettingsInput = {
  developerMode?: InputMaybe<Scalars['Boolean']['input']>;
  imageCache?: InputMaybe<ImageCacheSettingsInput>;
  language?: InputMaybe<Scalars['String']['input']>;
  stashBoxDataCache?: InputMaybe<StashBoxDataCacheSettingsInput>;
  taskDeletePolicy?: InputMaybe<TaskDeletePolicy>;
};

export type DownloaderDocumentQueryVariables = Exact<{
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  search?: InputMaybe<Scalars['String']['input']>;
  taskId?: InputMaybe<Scalars['ID']['input']>;
  attentionOnly?: InputMaybe<Scalars['Boolean']['input']>;
}>;


export type DownloaderDocumentQuery = { __typename?: 'Query', downloader: { __typename?: 'DownloaderOverview', checkedAt: string, connected: boolean, stale: boolean, storageAvailable: boolean, freeSpace?: number | null, downloadSpeed: number, uploadSpeed: number, downloadLimit: number, uploadLimit: number, connectionStatus: string, dhtNodes: number, completedBytes: number, remainingBytes: number, total: number, attentionCount: number, policy: { __typename?: 'DownloaderPolicy', enabled: boolean, ratio: number, seedHours: number, matchAll: boolean, deleteFiles: boolean, afterIngest: boolean }, torrents: Array<{ __typename?: 'ManagedTorrent', hash: string, name: string, state: string, present: boolean, taskIds: Array<string>, taskCodes: Array<string>, allCompleted: boolean, canDeleteFiles: boolean, progress: number, downloadSpeed: number, uploadSpeed: number, size: number, completedBytes: number, downloaded: number, uploaded: number, ratio: number, seedingSeconds: number, eta: number, seeds: number, peers: number, availability: number, savePath: string, category: string, tags: string, lastSeenAt: string, removedAt: string, removalReason: string, cleanupError: boolean, needsAttention: boolean }> } };

export type RefreshDownloaderDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type RefreshDownloaderDocumentMutation = { __typename?: 'Mutation', refreshDownloader: boolean };

export type UpdateDownloaderPolicyDocumentMutationVariables = Exact<{
  input: DownloaderPolicyInput;
}>;


export type UpdateDownloaderPolicyDocumentMutation = { __typename?: 'Mutation', updateDownloaderPolicy: { __typename?: 'DownloaderPolicy', enabled: boolean, ratio: number, seedHours: number, matchAll: boolean, deleteFiles: boolean, afterIngest: boolean } };

export type ControlTorrentDocumentMutationVariables = Exact<{
  hash: Scalars['String']['input'];
  action: TorrentAction;
}>;


export type ControlTorrentDocumentMutation = { __typename?: 'Mutation', controlTorrent: boolean };

export type SetDownloaderSpeedLimitDocumentMutationVariables = Exact<{
  upload: Scalars['Boolean']['input'];
  kibPerSecond: Scalars['Int']['input'];
}>;


export type SetDownloaderSpeedLimitDocumentMutation = { __typename?: 'Mutation', setDownloaderSpeedLimit: boolean };

export type DownloaderSummaryDocumentQueryVariables = Exact<{ [key: string]: never; }>;


export type DownloaderSummaryDocumentQuery = { __typename?: 'Query', downloader: { __typename?: 'DownloaderOverview', checkedAt: string, connected: boolean, stale: boolean, storageAvailable: boolean, freeSpace?: number | null, downloadSpeed: number, uploadSpeed: number, completedBytes: number, remainingBytes: number, total: number, attentionCount: number, connectionStatus: string } };

export type DownloaderSettingsDocumentQueryVariables = Exact<{ [key: string]: never; }>;


export type DownloaderSettingsDocumentQuery = { __typename?: 'Query', downloader: { __typename?: 'DownloaderOverview', stale: boolean, downloadLimit: number, uploadLimit: number, storageAvailable: boolean, policy: { __typename?: 'DownloaderPolicy', enabled: boolean, ratio: number, seedHours: number, matchAll: boolean, deleteFiles: boolean, afterIngest: boolean } } };

export type HomeIndexerSummaryQueryVariables = Exact<{ [key: string]: never; }>;


export type HomeIndexerSummaryQuery = { __typename?: 'Query', homeIndexerSummary: { __typename?: 'HomeIndexerSummary', definitionErrorCode: string, revision: string, total?: number | null, builtinEnabled: number, builtinUnverified: number, builtinSearchSucceeded: number, healthErrorCode: string, attention: number, definitionJob?: { __typename?: 'IndexerDefinitionJob', id: string, state: IndexerDefinitionJobState, errorCode: string, correlationId: string, finishedAt?: string | null } | null, issues: Array<{ __typename?: 'HomeIndexerIssue', id: string, name: string, errorCode: string, operation: string, observation?: { __typename?: 'IndexerObservation', startedAt: string, finishedAt: string, errorCode: string, correlationId: string, retryAt?: string | null } | null }>, jackett?: { __typename?: 'HomeJackettSummary', sites?: number | null, stale: boolean, updatedAt?: string | null, errorCode: string } | null } };

export type HomeIndexerDefinitionEventsSubscriptionVariables = Exact<{ [key: string]: never; }>;


export type HomeIndexerDefinitionEventsSubscription = { __typename?: 'Subscription', indexerDefinitionEvents: { __typename?: 'IndexerDefinitionStatus', streamId: string, revision: number, job?: { __typename?: 'IndexerDefinitionJob', id: string, revision: number } | null } };

export type HomePageDocumentQueryVariables = Exact<{ [key: string]: never; }>;


export type HomePageDocumentQuery = { __typename?: 'Query', settings: { __typename?: 'Settings', system: { __typename?: 'SystemSettings', taskDeletePolicy: TaskDeletePolicy }, automation: { __typename?: 'AutomationSettings', taskProcessingEnabled: boolean }, stash: { __typename?: 'StashSettings', url: string }, ingest: { __typename?: 'IngestSettings', keepSubtitles: boolean, downloads: { __typename?: 'DownloadsIngestSettings', qbRoot: string, mojiRoot: string }, library: { __typename?: 'LibraryIngestSettings', mojiRoot: string, stashRoot: string }, transfer: { __typename?: 'TransferIngestSettings', action: string } }, jackett: { __typename?: 'JackettSettings', url: string }, qbittorrent: { __typename?: 'QBittorrentSettings', url: string, username: string, defaultSavePath: string } }, homeOverview: { __typename?: 'HomeTaskSnapshot', sequence: number, streamId: string, total: number, attention: number, active: number, waiting: number, completedLast24Hours: number, attentionTasks: Array<(
      { __typename?: 'TaskSummary' }
      & { ' $fragmentRefs'?: { 'HomeTaskFragment': HomeTaskFragment } }
    )>, activeTasks: Array<(
      { __typename?: 'TaskSummary' }
      & { ' $fragmentRefs'?: { 'HomeTaskFragment': HomeTaskFragment } }
    )>, recentTasks: Array<(
      { __typename?: 'TaskSummary' }
      & { ' $fragmentRefs'?: { 'HomeTaskFragment': HomeTaskFragment } }
    )> } };

export type HomeTaskFragment = (
  { __typename?: 'TaskSummary' }
  & { ' $fragmentRefs'?: { 'TaskListSummaryFragment': TaskListSummaryFragment } }
) & { ' $fragmentName'?: 'HomeTaskFragment' };

export type HomeSubscriptionSummaryQueryVariables = Exact<{ [key: string]: never; }>;


export type HomeSubscriptionSummaryQuery = { __typename?: 'Query', homeSubscriptionSummary: { __typename?: 'HomeSubscriptionSummary', total: number, attention: number, unchecked: number, lastCheckedAt?: string | null, attentionPerformerIds: Array<string> } };

export type HomeSubscriptionChecksQueryVariables = Exact<{ [key: string]: never; }>;


export type HomeSubscriptionChecksQuery = { __typename?: 'Query', homeSubscriptionChecks: { __typename?: 'HomeSubscriptionChecks', enabled: boolean, nextRunAt?: string | null, activeCount: number, latestScheduled?: (
      { __typename?: 'HomeSubscriptionCheckRun' }
      & { ' $fragmentRefs'?: { 'HomeSubscriptionCheckRunFieldsFragment': HomeSubscriptionCheckRunFieldsFragment } }
    ) | null, activeRuns: Array<(
      { __typename?: 'HomeSubscriptionCheckRun' }
      & { ' $fragmentRefs'?: { 'HomeSubscriptionCheckRunFieldsFragment': HomeSubscriptionCheckRunFieldsFragment } }
    )>, recentRuns: Array<(
      { __typename?: 'HomeSubscriptionCheckRun' }
      & { ' $fragmentRefs'?: { 'HomeSubscriptionCheckRunFieldsFragment': HomeSubscriptionCheckRunFieldsFragment } }
    )> } };

export type HomeSubscriptionCheckRunFieldsFragment = { __typename?: 'HomeSubscriptionCheckRun', id: string, trigger: SubscriptionCheckTrigger, scope: SubscriptionCheckScope, status: SubscriptionCheckStatus, startedAt: string, finishedAt?: string | null, totalCount: number, completedCount: number, succeededCount: number, failedCount: number, skippedCount: number, currentPerformers: Array<{ __typename?: 'HomeCheckPerformer', performerId: string, name: string }> } & { ' $fragmentName'?: 'HomeSubscriptionCheckRunFieldsFragment' };

export type HomeServiceStatusQueryVariables = Exact<{ [key: string]: never; }>;


export type HomeServiceStatusQuery = { __typename?: 'Query', settingsStatus: { __typename?: 'SettingsStatus', stash: { __typename?: 'ServiceStatus', configured: boolean, ready: boolean }, qbittorrent: { __typename?: 'ServiceStatus', configured: boolean, ready: boolean }, ingest: { __typename?: 'IngestStatus', configured: boolean }, stashStats: { __typename?: 'StashStats', version?: string | null, sceneCount?: number | null, pendingMojiScanCount: number, lastError?: string | null, okAt?: string | null } } };

export type IndexerCacheStatusFieldsFragment = { __typename?: 'IndexerCacheStatus', revision: string, enabled: boolean, ttlSeconds: number, emptyTtlSeconds: number, maxSizeMb: number, entries: number, usedBytes: number, activeRequests: number, activeWaiters: number } & { ' $fragmentName'?: 'IndexerCacheStatusFieldsFragment' };

export type IndexerCacheStatusQueryVariables = Exact<{ [key: string]: never; }>;


export type IndexerCacheStatusQuery = { __typename?: 'Query', indexerCacheStatus: (
    { __typename?: 'IndexerCacheStatus' }
    & { ' $fragmentRefs'?: { 'IndexerCacheStatusFieldsFragment': IndexerCacheStatusFieldsFragment } }
  ) };

export type ClearIndexerSearchCacheMutationVariables = Exact<{
  instanceId?: InputMaybe<Scalars['ID']['input']>;
}>;


export type ClearIndexerSearchCacheMutation = { __typename?: 'Mutation', clearIndexerSearchCache: (
    { __typename?: 'IndexerCacheStatus' }
    & { ' $fragmentRefs'?: { 'IndexerCacheStatusFieldsFragment': IndexerCacheStatusFieldsFragment } }
  ) };

export type UpdateIndexerSearchCacheMutationVariables = Exact<{
  expectedRevision: Scalars['String']['input'];
  input: IndexerSearchCachePatchInput;
}>;


export type UpdateIndexerSearchCacheMutation = { __typename?: 'Mutation', updateIndexerSearchCache: (
    { __typename?: 'IndexerCacheStatus' }
    & { ' $fragmentRefs'?: { 'IndexerCacheStatusFieldsFragment': IndexerCacheStatusFieldsFragment } }
  ) };

export type TestIndexerMutationVariables = Exact<{
  input: TestIndexerInput;
}>;


export type TestIndexerMutation = { __typename?: 'Mutation', testIndexer: { __typename?: 'IndexerCheckReport', id: string, instanceId: string, kind?: IndexerSourceKind | null, configRevision: string, ruleRevision: string, startedAt: string, finishedAt: string, results: number, partial: boolean, truncated: boolean, stages: Array<{ __typename?: 'IndexerCheckStage', name: IndexerCheckStageName, state: IndexerCheckStageState, errorCode: string, elapsedMs: number }> } };

export type IndexerCheckActivitySubscriptionVariables = Exact<{
  clientId: Scalars['ID']['input'];
}>;


export type IndexerCheckActivitySubscription = { __typename?: 'Subscription', indexerCheckActivity: Array<{ __typename?: 'IndexerCheckActivity', requestId: string, instanceId: string, state: IndexerCheckActivityState }> };

export type IndexerDefinitionPolicyFieldsFragment = { __typename?: 'IndexerDefinitionPolicy', revision: string, source: string, schemaVersion: number, checkIntervalHours: number, autoUpdate: boolean } & { ' $fragmentName'?: 'IndexerDefinitionPolicyFieldsFragment' };

export type IndexerDefinitionPolicyQueryVariables = Exact<{ [key: string]: never; }>;


export type IndexerDefinitionPolicyQuery = { __typename?: 'Query', indexerDefinitionPolicy: (
    { __typename?: 'IndexerDefinitionPolicy' }
    & { ' $fragmentRefs'?: { 'IndexerDefinitionPolicyFieldsFragment': IndexerDefinitionPolicyFieldsFragment } }
  ) };

export type UpdateIndexerDefinitionPolicyMutationVariables = Exact<{
  expectedRevision: Scalars['String']['input'];
  input: IndexerDefinitionPolicyPatchInput;
}>;


export type UpdateIndexerDefinitionPolicyMutation = { __typename?: 'Mutation', updateIndexerDefinitionPolicy: (
    { __typename?: 'IndexerDefinitionPolicy' }
    & { ' $fragmentRefs'?: { 'IndexerDefinitionPolicyFieldsFragment': IndexerDefinitionPolicyFieldsFragment } }
  ) };

export type IndexerDefinitionJobFieldsFragment = { __typename?: 'IndexerDefinitionJob', id: string, kind: IndexerDefinitionJobKind, state: IndexerDefinitionJobState, targetRevision: string, expectedActiveRevision: string, downloadedBytes: number, errorCode: string, correlationId: string, startedAt: string, finishedAt?: string | null, revision: number } & { ' $fragmentName'?: 'IndexerDefinitionJobFieldsFragment' };

export type IndexerDefinitionStatusFieldsFragment = { __typename?: 'IndexerDefinitionStatus', streamId: string, activeRevision: string, previousRevision: string, availableRevision: string, revision: number, available: boolean, lastCheckedAt?: string | null, checkErrorCode: string, errorCode: string, installed: Array<{ __typename?: 'IndexerDefinitionManifest', revision: string, schemaVersion: number, engineVersion: string, definitionCount: number, installedAt: string }>, job?: (
    { __typename?: 'IndexerDefinitionJob' }
    & { ' $fragmentRefs'?: { 'IndexerDefinitionJobFieldsFragment': IndexerDefinitionJobFieldsFragment } }
  ) | null } & { ' $fragmentName'?: 'IndexerDefinitionStatusFieldsFragment' };

export type IndexerDefinitionStatusQueryVariables = Exact<{ [key: string]: never; }>;


export type IndexerDefinitionStatusQuery = { __typename?: 'Query', indexerDefinitionStatus: (
    { __typename?: 'IndexerDefinitionStatus' }
    & { ' $fragmentRefs'?: { 'IndexerDefinitionStatusFieldsFragment': IndexerDefinitionStatusFieldsFragment } }
  ) };

export type IndexerDefinitionUpdateQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type IndexerDefinitionUpdateQuery = { __typename?: 'Query', indexerDefinitionUpdate?: (
    { __typename?: 'IndexerDefinitionJob' }
    & { ' $fragmentRefs'?: { 'IndexerDefinitionJobFieldsFragment': IndexerDefinitionJobFieldsFragment } }
  ) | null };

export type CheckIndexerDefinitionUpdateMutationVariables = Exact<{ [key: string]: never; }>;


export type CheckIndexerDefinitionUpdateMutation = { __typename?: 'Mutation', checkIndexerDefinitionUpdate: (
    { __typename?: 'IndexerDefinitionStatus' }
    & { ' $fragmentRefs'?: { 'IndexerDefinitionStatusFieldsFragment': IndexerDefinitionStatusFieldsFragment } }
  ) };

export type InstallIndexerDefinitionsMutationVariables = Exact<{
  input: IndexerDefinitionChangeInput;
}>;


export type InstallIndexerDefinitionsMutation = { __typename?: 'Mutation', installIndexerDefinitions: (
    { __typename?: 'IndexerDefinitionJob' }
    & { ' $fragmentRefs'?: { 'IndexerDefinitionJobFieldsFragment': IndexerDefinitionJobFieldsFragment } }
  ) };

export type UpdateIndexerDefinitionsMutationVariables = Exact<{
  input: IndexerDefinitionChangeInput;
}>;


export type UpdateIndexerDefinitionsMutation = { __typename?: 'Mutation', updateIndexerDefinitions: (
    { __typename?: 'IndexerDefinitionJob' }
    & { ' $fragmentRefs'?: { 'IndexerDefinitionJobFieldsFragment': IndexerDefinitionJobFieldsFragment } }
  ) };

export type RollbackIndexerDefinitionsMutationVariables = Exact<{
  input: IndexerDefinitionChangeInput;
}>;


export type RollbackIndexerDefinitionsMutation = { __typename?: 'Mutation', rollbackIndexerDefinitions: (
    { __typename?: 'IndexerDefinitionJob' }
    & { ' $fragmentRefs'?: { 'IndexerDefinitionJobFieldsFragment': IndexerDefinitionJobFieldsFragment } }
  ) };

export type RepairIndexerDefinitionsMutationVariables = Exact<{
  input: IndexerDefinitionChangeInput;
}>;


export type RepairIndexerDefinitionsMutation = { __typename?: 'Mutation', repairIndexerDefinitions: (
    { __typename?: 'IndexerDefinitionJob' }
    & { ' $fragmentRefs'?: { 'IndexerDefinitionJobFieldsFragment': IndexerDefinitionJobFieldsFragment } }
  ) };

export type CancelIndexerDefinitionUpdateMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type CancelIndexerDefinitionUpdateMutation = { __typename?: 'Mutation', cancelIndexerDefinitionUpdate: (
    { __typename?: 'IndexerDefinitionJob' }
    & { ' $fragmentRefs'?: { 'IndexerDefinitionJobFieldsFragment': IndexerDefinitionJobFieldsFragment } }
  ) };

export type IndexerDefinitionEventsSubscriptionVariables = Exact<{ [key: string]: never; }>;


export type IndexerDefinitionEventsSubscription = { __typename?: 'Subscription', indexerDefinitionEvents: (
    { __typename?: 'IndexerDefinitionStatus' }
    & { ' $fragmentRefs'?: { 'IndexerDefinitionStatusFieldsFragment': IndexerDefinitionStatusFieldsFragment } }
  ) };

export type IndexerEventsSubscriptionVariables = Exact<{ [key: string]: never; }>;


export type IndexerEventsSubscription = { __typename?: 'Subscription', indexerEvents: { __typename?: 'IndexerEvent', streamId: string, sequence: number, kind: IndexerEventKind } };

export type IndexerLoginResultFragment = { __typename?: 'IndexerLoginReport', id: string, instanceId: string, configRevision: string, ruleRevision: string, state: IndexerLoginState, expiresAt: string, challenge?: { __typename?: 'IndexerLoginChallenge', id: string, type: string, prompt: string, imageData?: string | null } | null } & { ' $fragmentName'?: 'IndexerLoginResultFragment' };

export type BeginIndexerLoginMutationVariables = Exact<{
  input: IndexerLoginInput;
}>;


export type BeginIndexerLoginMutation = { __typename?: 'Mutation', beginIndexerLogin: (
    { __typename?: 'IndexerLoginReport' }
    & { ' $fragmentRefs'?: { 'IndexerLoginResultFragment': IndexerLoginResultFragment } }
  ) };

export type AnswerIndexerChallengeMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: IndexerLoginInput;
  answer: Scalars['String']['input'];
}>;


export type AnswerIndexerChallengeMutation = { __typename?: 'Mutation', answerIndexerChallenge: (
    { __typename?: 'IndexerLoginReport' }
    & { ' $fragmentRefs'?: { 'IndexerLoginResultFragment': IndexerLoginResultFragment } }
  ) };

export type CancelIndexerLoginMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: IndexerLoginInput;
}>;


export type CancelIndexerLoginMutation = { __typename?: 'Mutation', cancelIndexerLogin: boolean };

export type IndexerLoginQueryVariables = Exact<{
  input: IndexerLoginInput;
}>;


export type IndexerLoginQuery = { __typename?: 'Query', indexerLogin?: (
    { __typename?: 'IndexerLoginReport' }
    & { ' $fragmentRefs'?: { 'IndexerLoginResultFragment': IndexerLoginResultFragment } }
  ) | null };

export type IndexerSettingFieldsFragment = { __typename?: 'IndexerSetting', name: string, type: string, label: string, secret: boolean, configured: boolean, value?: { __typename?: 'IndexerSettingValue', string?: string | null, boolean?: boolean | null, number?: number | null } | null, defaultValue?: { __typename?: 'IndexerSettingValue', string?: string | null, boolean?: boolean | null, number?: number | null } | null, options: Array<{ __typename?: 'IndexerSettingOption', value: string, label: string }> } & { ' $fragmentName'?: 'IndexerSettingFieldsFragment' };

export type IndexerInstanceFieldsFragment = { __typename?: 'IndexerInstance', connectionRevision: string, siteUrl: string, description: string, privacy: string, language: string, loginSupported: boolean, id: string, kind: IndexerSourceKind, name: string, enabled: boolean, definitionId: string, baseUrl: string, networkProfile: string, trustedOrigins: Array<string>, ruleDigest: string, errorCode: string, modes: Array<string>, health: { __typename?: 'IndexerHealth', search?: (
      { __typename?: 'IndexerObservation' }
      & { ' $fragmentRefs'?: { 'IndexerObservationFieldsFragment': IndexerObservationFieldsFragment } }
    ) | null, fetch?: (
      { __typename?: 'IndexerObservation' }
      & { ' $fragmentRefs'?: { 'IndexerObservationFieldsFragment': IndexerObservationFieldsFragment } }
    ) | null }, settings: Array<(
    { __typename?: 'IndexerSetting' }
    & { ' $fragmentRefs'?: { 'IndexerSettingFieldsFragment': IndexerSettingFieldsFragment } }
  )> } & { ' $fragmentName'?: 'IndexerInstanceFieldsFragment' };

export type IndexerInstancesFieldsFragment = { __typename?: 'IndexerInstances', healthErrorCode: string, revision: string, ruleRevision: string, ruleErrorCode: string, catalogErrorCode: string, jackettEnabled: boolean, order: Array<string>, networkProfiles: Array<string>, items: Array<(
    { __typename?: 'IndexerInstance' }
    & { ' $fragmentRefs'?: { 'IndexerInstanceFieldsFragment': IndexerInstanceFieldsFragment } }
  )> } & { ' $fragmentName'?: 'IndexerInstancesFieldsFragment' };

export type IndexerDefinitionsQueryVariables = Exact<{
  input?: InputMaybe<IndexerDefinitionFilterInput>;
}>;


export type IndexerDefinitionsQuery = { __typename?: 'Query', indexerDefinitions: { __typename?: 'IndexerDefinitionPage', revision: string, totalCount: number, offset: number, limit: number, items: Array<{ __typename?: 'IndexerDefinition', id: string, name: string, description: string, language: string, privacy: string, loginSupported: boolean, digest: string, links: Array<string>, categories: Array<string>, modes: Array<string>, warnings: Array<string>, settings: Array<(
        { __typename?: 'IndexerSetting' }
        & { ' $fragmentRefs'?: { 'IndexerSettingFieldsFragment': IndexerSettingFieldsFragment } }
      )> }> } };

export type IndexerInstancesQueryVariables = Exact<{ [key: string]: never; }>;


export type IndexerInstancesQuery = { __typename?: 'Query', indexerInstances: (
    { __typename?: 'IndexerInstances' }
    & { ' $fragmentRefs'?: { 'IndexerInstancesFieldsFragment': IndexerInstancesFieldsFragment } }
  ) };

export type AddBuiltinIndexerMutationVariables = Exact<{
  expectedRevision: Scalars['String']['input'];
  input: BuiltinIndexerInput;
}>;


export type AddBuiltinIndexerMutation = { __typename?: 'Mutation', addBuiltinIndexer: { __typename?: 'BuiltinIndexerResult', revision: string, instance: (
      { __typename?: 'IndexerInstance' }
      & { ' $fragmentRefs'?: { 'IndexerInstanceFieldsFragment': IndexerInstanceFieldsFragment } }
    ) } };

export type UpdateBuiltinIndexerMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  expectedRevision: Scalars['String']['input'];
  input: BuiltinIndexerPatchInput;
}>;


export type UpdateBuiltinIndexerMutation = { __typename?: 'Mutation', updateBuiltinIndexer: { __typename?: 'BuiltinIndexerResult', revision: string, instance: (
      { __typename?: 'IndexerInstance' }
      & { ' $fragmentRefs'?: { 'IndexerInstanceFieldsFragment': IndexerInstanceFieldsFragment } }
    ) } };

export type RemoveBuiltinIndexerMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  expectedRevision: Scalars['String']['input'];
}>;


export type RemoveBuiltinIndexerMutation = { __typename?: 'Mutation', removeBuiltinIndexer: (
    { __typename?: 'IndexerInstances' }
    & { ' $fragmentRefs'?: { 'IndexerInstancesFieldsFragment': IndexerInstancesFieldsFragment } }
  ) };

export type SetIndexerEnabledMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  expectedRevision: Scalars['String']['input'];
  enabled: Scalars['Boolean']['input'];
}>;


export type SetIndexerEnabledMutation = { __typename?: 'Mutation', setIndexerEnabled: (
    { __typename?: 'IndexerInstances' }
    & { ' $fragmentRefs'?: { 'IndexerInstancesFieldsFragment': IndexerInstancesFieldsFragment } }
  ) };

export type ReorderIndexersMutationVariables = Exact<{
  expectedRevision: Scalars['String']['input'];
  ids: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type ReorderIndexersMutation = { __typename?: 'Mutation', reorderIndexers: (
    { __typename?: 'IndexerInstances' }
    & { ' $fragmentRefs'?: { 'IndexerInstancesFieldsFragment': IndexerInstancesFieldsFragment } }
  ) };

export type RefreshJackettIndexersMutationVariables = Exact<{ [key: string]: never; }>;


export type RefreshJackettIndexersMutation = { __typename?: 'Mutation', refreshJackettIndexers: (
    { __typename?: 'IndexerInstances' }
    & { ' $fragmentRefs'?: { 'IndexerInstancesFieldsFragment': IndexerInstancesFieldsFragment } }
  ) };

export type IndexerObservationFieldsFragment = { __typename?: 'IndexerObservation', startedAt: string, finishedAt: string, errorCode: string, correlationId: string, retryAt?: string | null } & { ' $fragmentName'?: 'IndexerObservationFieldsFragment' };

export type IndexerNetworkSettingsFieldsFragment = { __typename?: 'IndexerNetworkSettings', revision: string, requestTimeoutSeconds: number, siteTimeoutSeconds: number, challengeTimeoutSeconds: number, profiles: Array<{ __typename?: 'IndexerNetworkProfile', id: string, proxyConfigured: boolean, flareSolverrConfigured: boolean, usedBy: number }> } & { ' $fragmentName'?: 'IndexerNetworkSettingsFieldsFragment' };

export type IndexerNetworkSettingsQueryVariables = Exact<{ [key: string]: never; }>;


export type IndexerNetworkSettingsQuery = { __typename?: 'Query', indexerNetworkSettings: (
    { __typename?: 'IndexerNetworkSettings' }
    & { ' $fragmentRefs'?: { 'IndexerNetworkSettingsFieldsFragment': IndexerNetworkSettingsFieldsFragment } }
  ) };

export type UpdateIndexerNetworkTimeoutsMutationVariables = Exact<{
  expectedRevision: Scalars['String']['input'];
  input: IndexerNetworkTimeoutPatchInput;
}>;


export type UpdateIndexerNetworkTimeoutsMutation = { __typename?: 'Mutation', updateIndexerNetworkTimeouts: (
    { __typename?: 'IndexerNetworkSettings' }
    & { ' $fragmentRefs'?: { 'IndexerNetworkSettingsFieldsFragment': IndexerNetworkSettingsFieldsFragment } }
  ) };

export type AddIndexerNetworkProfileMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  expectedRevision: Scalars['String']['input'];
  input: IndexerNetworkProfilePatchInput;
}>;


export type AddIndexerNetworkProfileMutation = { __typename?: 'Mutation', addIndexerNetworkProfile: (
    { __typename?: 'IndexerNetworkSettings' }
    & { ' $fragmentRefs'?: { 'IndexerNetworkSettingsFieldsFragment': IndexerNetworkSettingsFieldsFragment } }
  ) };

export type UpdateIndexerNetworkProfileMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  expectedRevision: Scalars['String']['input'];
  input: IndexerNetworkProfilePatchInput;
}>;


export type UpdateIndexerNetworkProfileMutation = { __typename?: 'Mutation', updateIndexerNetworkProfile: (
    { __typename?: 'IndexerNetworkSettings' }
    & { ' $fragmentRefs'?: { 'IndexerNetworkSettingsFieldsFragment': IndexerNetworkSettingsFieldsFragment } }
  ) };

export type RemoveIndexerNetworkProfileMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  expectedRevision: Scalars['String']['input'];
}>;


export type RemoveIndexerNetworkProfileMutation = { __typename?: 'Mutation', removeIndexerNetworkProfile: (
    { __typename?: 'IndexerNetworkSettings' }
    & { ' $fragmentRefs'?: { 'IndexerNetworkSettingsFieldsFragment': IndexerNetworkSettingsFieldsFragment } }
  ) };

export type LogEventsDocumentSubscriptionVariables = Exact<{ [key: string]: never; }>;


export type LogEventsDocumentSubscription = { __typename?: 'Subscription', logEvents: { __typename?: 'LogEvent', sequence: number, entry: { __typename?: 'LogEntry', sequence: number, time: string, level: LogLevel, message: string } } };

export type InterfaceSettingsQueryVariables = Exact<{ [key: string]: never; }>;


export type InterfaceSettingsQuery = { __typename?: 'Query', settings: { __typename?: 'Settings', system: { __typename?: 'SystemSettings', language: string, developerMode: boolean } } };

export type DatabasePerformanceDocumentSubscriptionVariables = Exact<{ [key: string]: never; }>;


export type DatabasePerformanceDocumentSubscription = { __typename?: 'Subscription', databasePerformance: { __typename?: 'DatabasePerformanceSnapshot', startedAt: string, sampledAt: string, intervalSeconds: number, lastError: string, databases: Array<{ __typename?: 'DatabasePerformance', role: string, databaseBytes: number, walBytes: number, freeBytes: number, writeAttempts: number, writeErrors: number, busyRetries: number, queueRejected: number, queueCanceled: number, queued: number, queueCapacity: number, writesPerSecond: number, averageWaitMilliseconds: number, averageExecutionMilliseconds: number, totalWaitMilliseconds: number, totalExecutionMilliseconds: number, reader: { __typename?: 'DatabasePoolMetrics', open: number, inUse: number, idle: number, limit: number, waitCount: number, waitMilliseconds: number }, writer: { __typename?: 'DatabasePoolMetrics', open: number, inUse: number, idle: number, limit: number, waitCount: number, waitMilliseconds: number } }> } };

export type PerformerSubscriptionEventsSubscriptionVariables = Exact<{ [key: string]: never; }>;


export type PerformerSubscriptionEventsSubscription = { __typename?: 'Subscription', performerSubscriptionEvents: { __typename?: 'PerformerSubscriptionEvent', sequence: number, type: PerformerSubscriptionEventType, performerId: string, state?: { __typename?: 'SubscribedPerformer', lastCheckedAt?: string | null, lastError?: string | null, reviewReleaseCount: number, pendingReleaseCount: number, processedReleaseCount: number, performer: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }, recentReleases: Array<{ __typename?: 'SubscriptionRelease', key: string, title: string, code?: string | null, date?: string | null, seenAt: string }> } | null } };

export type ReleaseReviewsQueryVariables = Exact<{
  input: ReleaseReviewQueryInput;
}>;


export type ReleaseReviewsQuery = { __typename?: 'Query', releaseReviews: { __typename?: 'ReleaseReviewConnection', totalCount: number, pendingCount: number, failedCount: number, page: number, pageSize: number, totalPages: number, performers: Array<{ __typename?: 'ReleaseReviewPerformer', id: string, name: string }>, items: Array<{ __typename?: 'ReleaseReview', key: string, code: string, title: string, date?: string | null, url?: string | null, imageUrl?: string | null, source: string, vr: boolean, status: ReleaseReviewStatus, reasonCode: string, error?: string | null, taskId?: string | null, seenAt: string, reviewedAt?: string | null, performerNames: Array<string>, performers: Array<{ __typename?: 'ReleaseReviewPerformer', id: string, name: string }> }> } };

export type ReleaseReviewSummaryQueryVariables = Exact<{ [key: string]: never; }>;


export type ReleaseReviewSummaryQuery = { __typename?: 'Query', releaseReviews: { __typename?: 'ReleaseReviewConnection', pendingCount: number, failedCount: number, performers: Array<{ __typename?: 'ReleaseReviewPerformer', id: string, pendingCount: number }> } };

export type ProcessReleaseReviewsMutationVariables = Exact<{
  input: ProcessReleaseReviewsInput;
}>;


export type ProcessReleaseReviewsMutation = { __typename?: 'Mutation', processReleaseReviews: { __typename?: 'ReleaseReviewPayload', succeededCount: number, skippedCount: number, failedCount: number, results: Array<{ __typename?: 'ReleaseReviewResult', key: string, outcome: string, reasonCode: string, release?: { __typename?: 'ReleaseReview', key: string, status: ReleaseReviewStatus, error?: string | null, taskId?: string | null } | null }> } };

export type DiscoverConfigDocumentQueryVariables = Exact<{ [key: string]: never; }>;


export type DiscoverConfigDocumentQuery = { __typename?: 'Query', settings: { __typename?: 'Settings', automation: { __typename?: 'AutomationSettings', torrentSelection: { __typename?: 'TorrentSelectionSettings', inspectionCandidateLimit: number } } } };

export type PerformersConfigDocumentQueryVariables = Exact<{ [key: string]: never; }>;


export type PerformersConfigDocumentQuery = { __typename?: 'Query', settings: { __typename?: 'Settings', stash: { __typename?: 'StashSettings', url: string } } };

export type StatsPageDocumentQueryVariables = Exact<{ [key: string]: never; }>;


export type StatsPageDocumentQuery = { __typename?: 'Query', dashboardStats: { __typename?: 'DashboardStats', active: number, completed: number, pendingScans: number, failed: number } };

export type TaskProcessingStateQueryVariables = Exact<{ [key: string]: never; }>;


export type TaskProcessingStateQuery = { __typename?: 'Query', settings: { __typename?: 'Settings', automation: { __typename?: 'AutomationSettings', taskProcessingEnabled: boolean } } };

export type DiscoverScenesDocumentQueryVariables = Exact<{
  input: DiscoverScenesInput;
}>;


export type DiscoverScenesDocumentQuery = { __typename?: 'Query', discoverScenes: { __typename?: 'DiscoverSceneConnection', fallbackCount: number, searchedQuery: string, items: Array<{ __typename?: 'DiscoveredScene', key: string, sceneId: string, stashBoxEndpoint: string, stashBoxName: string, title: string, durationSeconds?: number | null, code?: string | null, date?: string | null, studioName?: string | null, imageUrl?: string | null, url?: string | null, performerNames: Array<string>, derivedQuery: string }>, usedStashBox?: { __typename?: 'MatchedStashBox', name: string, endpoint: string, performerId: string, performerName: string } | null } };

export type QueueDiscoveredSceneDocumentMutationVariables = Exact<{
  input: QueueDiscoveredSceneInput;
}>;


export type QueueDiscoveredSceneDocumentMutation = { __typename?: 'Mutation', queueDiscoveredScene: { __typename?: 'Task', id: string, source: TaskSource, stage: TaskStage, stageStatus: TaskStageStatus, stageLabel: string, stageStatusLabel: string, code: string, torrentName?: string | null, progress: number, createdAt: string } };

export type ServiceStatusEventsSubscriptionVariables = Exact<{ [key: string]: never; }>;


export type ServiceStatusEventsSubscription = { __typename?: 'Subscription', serviceStatusEvents: { __typename?: 'ServiceStatusEvent', sequence: number, services: Array<ExternalService>, observedAt: string } };

export type ServiceConnectionSettingsQueryVariables = Exact<{ [key: string]: never; }>;


export type ServiceConnectionSettingsQuery = { __typename?: 'Query', settings: { __typename?: 'Settings', stash: { __typename?: 'StashSettings', configured: boolean, url: string, apiKeyConfigured: boolean, apiKey: string }, jackett: { __typename?: 'JackettSettings', configured: boolean, url: string, apiKeyConfigured: boolean, apiKey: string, passwordConfigured: boolean, password: string }, qbittorrent: { __typename?: 'QBittorrentSettings', configured: boolean, url: string, username: string, usernameConfigured: boolean, passwordConfigured: boolean, password: string, defaultSavePath: string, category: string, tags: string } } };

export type ServiceConnectionStatusQueryVariables = Exact<{ [key: string]: never; }>;


export type ServiceConnectionStatusQuery = { __typename?: 'Query', settingsStatus: { __typename?: 'SettingsStatus', stash: { __typename?: 'ServiceStatus', configured: boolean, ready: boolean }, jackett: { __typename?: 'ServiceStatus', configured: boolean, ready: boolean }, qbittorrent: { __typename?: 'ServiceStatus', configured: boolean, ready: boolean }, stashStats: { __typename?: 'StashStats', lastError?: string | null, okAt?: string | null }, jackettStats: { __typename?: 'JackettStats', lastError?: string | null, okAt?: string | null }, qbittorrentStats: { __typename?: 'QBittorrentStats', lastError?: string | null, okAt?: string | null } } };

export type LibraryDeliverySettingsQueryVariables = Exact<{ [key: string]: never; }>;


export type LibraryDeliverySettingsQuery = { __typename?: 'Query', settings: { __typename?: 'Settings', delivery: { __typename?: 'IngestSettings', keepSubtitles: boolean, generateTranscodes: boolean, downloads: { __typename?: 'DownloadsIngestSettings', qbRoot: string, mojiRoot: string }, library: { __typename?: 'LibraryIngestSettings', mojiRoot: string, stashRoot: string }, transfer: { __typename?: 'TransferIngestSettings', action: string } }, qbittorrent: { __typename?: 'QBittorrentSettings', defaultSavePath: string } }, settingsStatus: { __typename?: 'SettingsStatus', stashLibrariesLoadError?: string | null, stashLibraries: Array<{ __typename?: 'StashLibrary', path: string }> } };

export type SettingsPolicySectionQueryVariables = Exact<{ [key: string]: never; }>;


export type SettingsPolicySectionQuery = { __typename?: 'Query', settings: { __typename?: 'Settings', policies: { __typename?: 'AutomationSettings', taskProcessingEnabled: boolean, taskProgressSyncIntervalSeconds: number, subscriptionPollIntervalHours: number, sourcingRetryIntervalHours: number, stashBoxEndpoints: Array<string>, subscriptionReleasePolicy: { __typename?: 'SubscriptionReleasePolicy', soloBehavior: SubscriptionReleaseBehavior, groupBehavior: SubscriptionReleaseBehavior, compilationBehavior: SubscriptionReleaseBehavior, vrBehavior: SubscriptionReleaseBehavior, maxGroupPerformerCount: number, releaseDateRange: SubscriptionReleaseDateRange }, torrentSelection: { __typename?: 'TorrentSelectionSettings', enabled: boolean, inspectionCandidateLimit: number, fastRules: Array<{ __typename?: 'TorrentSelectionRule', type: TorrentSelectionRuleType, enabled: boolean, publishDate: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, seeders: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, size: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, indexerPreference: { __typename?: 'IndexerPreferenceRule', trackerIds: Array<string> }, titleMatch: { __typename?: 'TitleMatchRule', clauses: Array<{ __typename?: 'TitleMatchClause', pattern: string, patternMode: TitleMatchPatternMode, effect: TitleMatchEffect }> }, torrentFileNameMatch: { __typename?: 'TorrentFileNameMatchRule', clauses: Array<{ __typename?: 'TorrentFileNameMatchClause', pattern: string, patternMode: TitleMatchPatternMode, effect: TorrentFileMatchEffect }> } }>, torrentRules: Array<{ __typename?: 'TorrentSelectionRule', type: TorrentSelectionRuleType, enabled: boolean, publishDate: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, seeders: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, size: { __typename?: 'DirectionRule', direction: TorrentSelectionDirection }, indexerPreference: { __typename?: 'IndexerPreferenceRule', trackerIds: Array<string> }, titleMatch: { __typename?: 'TitleMatchRule', clauses: Array<{ __typename?: 'TitleMatchClause', pattern: string, patternMode: TitleMatchPatternMode, effect: TitleMatchEffect }> }, torrentFileNameMatch: { __typename?: 'TorrentFileNameMatchRule', clauses: Array<{ __typename?: 'TorrentFileNameMatchClause', pattern: string, patternMode: TitleMatchPatternMode, effect: TorrentFileMatchEffect }> } }> } } }, settingsStatus: { __typename?: 'SettingsStatus', taskProcessing: { __typename?: 'AutomationStatus', taskProgressSyncIntervalSeconds: number, taskProgressSyncEnabled: boolean, subscriptionPollIntervalHours: number, subscriptionPollEnabled: boolean }, metadataSources: { __typename?: 'StashBoxStatus', stashBoxesLoaded: boolean, stashBoxesLoadError?: string | null, stashBoxes: Array<{ __typename?: 'StashBoxEndpoint', name: string, endpoint: string, apiKeyConfigured: boolean }> } } };

export type SystemSettingsTabQueryVariables = Exact<{ [key: string]: never; }>;


export type SystemSettingsTabQuery = { __typename?: 'Query', settings: { __typename?: 'Settings', system: { __typename?: 'SystemSettings', language: string, releaseTimezone: string, effectiveReleaseTimezone: string, developerMode: boolean, taskDeletePolicy: TaskDeletePolicy, imageCache: { __typename?: 'ImageCacheSettings', enabled: boolean, maxSizeMb: number, retentionDays: number }, stashBoxDataCache: { __typename?: 'StashBoxDataCacheSettings', ttlHours: number } } }, settingsStatus: { __typename?: 'SettingsStatus', imageCache: { __typename?: 'ImageCacheStatus', usedBytes: number, entryCount: number, cacheDirectory: string, lastCleanupAt?: string | null, lastError?: string | null }, stashBoxDataCache: { __typename?: 'StashBoxDataCacheStatus', usedBytes: number, sceneCount: number, performerCount: number, snapshotCount: number, databasePath: string, lastCleanupAt?: string | null, lastError?: string | null } } };

export type AboutSettingsTabQueryVariables = Exact<{ [key: string]: never; }>;


export type AboutSettingsTabQuery = { __typename?: 'Query', version: string };

export type TaskDeletionSettingsQueryVariables = Exact<{ [key: string]: never; }>;


export type TaskDeletionSettingsQuery = { __typename?: 'Query', settings: { __typename?: 'Settings', system: { __typename?: 'SystemSettings', taskDeletePolicy: TaskDeletePolicy } } };

export type TestServiceConnectionMutationVariables = Exact<{
  input: TestServiceConnectionInput;
}>;


export type TestServiceConnectionMutation = { __typename?: 'Mutation', testServiceConnection: { __typename?: 'ServiceConnectionTestResult', code: ServiceConnectionTestCode } };

export type ClearImageCacheDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type ClearImageCacheDocumentMutation = { __typename?: 'Mutation', clearImageCache: { __typename?: 'ImageCacheStatus', usedBytes: number, entryCount: number, cacheDirectory: string, lastCleanupAt?: string | null, lastError?: string | null } };

export type ClearStashBoxDataCacheDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type ClearStashBoxDataCacheDocumentMutation = { __typename?: 'Mutation', clearStashBoxDataCache: { __typename?: 'StashBoxDataCacheStatus', usedBytes: number, sceneCount: number, performerCount: number, snapshotCount: number, databasePath: string, lastCleanupAt?: string | null, lastError?: string | null } };

export type RefreshStashBoxesMutationVariables = Exact<{ [key: string]: never; }>;


export type RefreshStashBoxesMutation = { __typename?: 'Mutation', refreshStashBoxes: { __typename?: 'Settings', automation: { __typename?: 'AutomationSettings', sourcingRetryIntervalHours: number, stashBoxEndpoints: Array<string> } } };

export type LogsDocumentQueryVariables = Exact<{
  limit?: InputMaybe<Scalars['Int']['input']>;
  minLevel?: InputMaybe<LogLevel>;
}>;


export type LogsDocumentQuery = { __typename?: 'Query', logs: Array<{ __typename?: 'LogEntry', sequence: number, time: string, level: LogLevel, message: string }> };

export type PatchSettingsMutationVariables = Exact<{
  input: SettingsPatchInput;
}>;


export type PatchSettingsMutation = { __typename?: 'Mutation', patchSettings: { __typename?: 'Settings', system: { __typename?: 'SystemSettings', language: string, developerMode: boolean } } };

export type SubscriptionCheckFieldsFragment = { __typename?: 'SubscriptionCheckRun', id: string, trigger: SubscriptionCheckTrigger, scope: SubscriptionCheckScope, status: SubscriptionCheckStatus, startedAt: string, finishedAt?: string | null, error?: string | null, totalCount: number, completedCount: number, succeededCount: number, failedCount: number, skippedCount: number, items: Array<{ __typename?: 'SubscriptionCheckItem', performerId: string, name: string, status: SubscriptionCheckStatus, error?: string | null, startedAt?: string | null, finishedAt?: string | null }> } & { ' $fragmentName'?: 'SubscriptionCheckFieldsFragment' };

export type SubscriptionCheckActivityQueryVariables = Exact<{
  ids: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type SubscriptionCheckActivityQuery = { __typename?: 'Query', subscriptionCheckActivity: { __typename?: 'SubscriptionCheckActivity', activeCount: number, allRunId?: string | null, performers: Array<{ __typename?: 'PerformerCheckActivity', performerId: string, runId: string, status: SubscriptionCheckStatus }> } };

export type SubscriptionCheckDetailQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type SubscriptionCheckDetailQuery = { __typename?: 'Query', subscriptionCheck?: (
    { __typename?: 'SubscriptionCheckRun' }
    & { ' $fragmentRefs'?: { 'SubscriptionCheckFieldsFragment': SubscriptionCheckFieldsFragment } }
  ) | null };

export type StartSubscriptionCheckMutationVariables = Exact<{
  ids?: InputMaybe<Array<Scalars['ID']['input']> | Scalars['ID']['input']>;
}>;


export type StartSubscriptionCheckMutation = { __typename?: 'Mutation', startSubscriptionCheck: { __typename?: 'SubscriptionCheckRun', id: string } };

export type StashPerformersQueryVariables = Exact<{
  search?: InputMaybe<Scalars['String']['input']>;
  page?: InputMaybe<Scalars['Int']['input']>;
  pageSize?: InputMaybe<Scalars['Int']['input']>;
}>;


export type StashPerformersQuery = { __typename?: 'Query', stashPerformers: { __typename?: 'StashPerformerConnection', page: number, pageSize: number, totalCount: number, totalPages: number, hasPrevPage: boolean, hasNextPage: boolean, items: Array<{ __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }> } };

export type PerformerWorkspaceQueryVariables = Exact<{
  search?: InputMaybe<Scalars['String']['input']>;
  page?: InputMaybe<Scalars['Int']['input']>;
  pageSize?: InputMaybe<Scalars['Int']['input']>;
  options?: InputMaybe<PerformerListOptions>;
}>;


export type PerformerWorkspaceQuery = { __typename?: 'Query', performerWorkspace: { __typename?: 'PerformerWorkspaceSnapshot', performers: { __typename?: 'StashPerformerConnection', page: number, pageSize: number, totalCount: number, totalPages: number, hasPrevPage: boolean, hasNextPage: boolean, items: Array<{ __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }> }, subscribedPerformers: Array<{ __typename?: 'SubscribedPerformer', lastCheckedAt?: string | null, lastError?: string | null, reviewReleaseCount: number, pendingReleaseCount: number, processedReleaseCount: number, performer: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }, recentReleases: Array<{ __typename?: 'SubscriptionRelease', key: string, title: string, code?: string | null, date?: string | null, seenAt: string }> }> } };

export type StashPerformerProfileQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type StashPerformerProfileQuery = { __typename?: 'Query', stashPerformerProfile: { __typename?: 'StashPerformerProfile', disambiguation?: string | null, birthdate?: string | null, ethnicity?: string | null, country?: string | null, eyeColor?: string | null, heightCm?: number | null, rating100?: number | null, urls: Array<string>, performer: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean } } };

export type StashPerformerDetailQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type StashPerformerDetailQuery = { __typename?: 'Query', stashPerformerDetail: { __typename?: 'StashPerformerDetail', disambiguation?: string | null, birthdate?: string | null, ethnicity?: string | null, country?: string | null, eyeColor?: string | null, heightCm?: number | null, rating100?: number | null, urls: Array<string>, totalSceneCount: number, stashSceneCount: number, stashBoxSceneCount: number, dedupedSceneCount: number, activeTaskCount: number, sceneStatsComplete: boolean, sceneStatsStale: boolean, performer: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }, matchedStashBox?: { __typename?: 'MatchedStashBox', name: string, endpoint: string, performerId: string, performerName: string } | null } };

export type StashPerformerScenesQueryVariables = Exact<{
  id: Scalars['ID']['input'];
  input: StashPerformerScenesInput;
}>;


export type StashPerformerScenesQuery = { __typename?: 'Query', stashPerformerScenes: { __typename?: 'StashPerformerSceneConnection', page: number, pageSize: number, totalCount: number, totalPages?: number | null, hasPrevPage: boolean, hasNextPage: boolean, stashSceneCount: number, stashBoxCount: number, dedupedCount: number, totalCountExact: boolean, stashBoxRemoteCount: number, stashBoxLoadedCount: number, cacheComplete: boolean, cacheUpdatedAt?: string | null, cacheStale: boolean, items: Array<{ __typename?: 'StashPerformerScene', key: string, primarySource: SceneSource, sourceSceneId: string, title?: string | null, code?: string | null, date?: string | null, studioName?: string | null, performerCount: number, tagCount: number, imageUrl?: string | null, url?: string | null, inLibrary: boolean, matchedStashSceneId?: string | null, hasStashSource: boolean, hasStashBoxSource: boolean, stashBoxSceneId?: string | null, stashBoxEndpoint?: string | null, sourceLabels: Array<string>, upcomingState?: string | null, performers: Array<{ __typename?: 'PerformerScenePerson', id: string, name: string }>, tags: Array<{ __typename?: 'PerformerSceneTag', id: string, name: string }>, stashIds: Array<{ __typename?: 'StashSceneID', endpoint: string, stashId: string }>, mojiTask?: { __typename?: 'PerformerSceneTask', id: string, stage: TaskStage, stageStatus: TaskStageStatus, stageLabel: string, stageStatusLabel: string, progress: number } | null }> } };

export type RefreshStashPerformerScenesMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: StashPerformerScenesInput;
}>;


export type RefreshStashPerformerScenesMutation = { __typename?: 'Mutation', refreshStashPerformerScenes: { __typename?: 'StashPerformerSceneConnection', page: number, pageSize: number, totalCount: number, totalPages?: number | null, hasPrevPage: boolean, hasNextPage: boolean, stashSceneCount: number, stashBoxCount: number, dedupedCount: number, totalCountExact: boolean, stashBoxRemoteCount: number, stashBoxLoadedCount: number, cacheComplete: boolean, cacheUpdatedAt?: string | null, cacheStale: boolean, items: Array<{ __typename?: 'StashPerformerScene', key: string }> } };

export type SubscribedPerformersQueryVariables = Exact<{ [key: string]: never; }>;


export type SubscribedPerformersQuery = { __typename?: 'Query', subscribedPerformers: Array<{ __typename?: 'SubscribedPerformer', lastCheckedAt?: string | null, lastError?: string | null, reviewReleaseCount: number, pendingReleaseCount: number, processedReleaseCount: number, performer: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }, recentReleases: Array<{ __typename?: 'SubscriptionRelease', key: string, title: string, code?: string | null, date?: string | null, seenAt: string }> }> };

export type SubscribePerformerMutationVariables = Exact<{
  stashPerformerID: Scalars['ID']['input'];
}>;


export type SubscribePerformerMutation = { __typename?: 'Mutation', subscribePerformer: { __typename?: 'SubscribedPerformer', lastCheckedAt?: string | null, lastError?: string | null, reviewReleaseCount: number, pendingReleaseCount: number, processedReleaseCount: number, performer: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }, recentReleases: Array<{ __typename?: 'SubscriptionRelease', key: string, title: string, code?: string | null, date?: string | null, seenAt: string }> } };

export type UnsubscribePerformerMutationVariables = Exact<{
  stashPerformerID: Scalars['ID']['input'];
}>;


export type UnsubscribePerformerMutation = { __typename?: 'Mutation', unsubscribePerformer: boolean };

export type RefreshSubscribedPerformerMutationVariables = Exact<{
  stashPerformerID: Scalars['ID']['input'];
}>;


export type RefreshSubscribedPerformerMutation = { __typename?: 'Mutation', refreshSubscribedPerformer: { __typename?: 'SubscribedPerformer', lastCheckedAt?: string | null, lastError?: string | null, reviewReleaseCount: number, pendingReleaseCount: number, processedReleaseCount: number, performer: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }, recentReleases: Array<{ __typename?: 'SubscriptionRelease', key: string, title: string, code?: string | null, date?: string | null, seenAt: string }> } };

export type RefreshSubscriptionsNowMutationVariables = Exact<{ [key: string]: never; }>;


export type RefreshSubscriptionsNowMutation = { __typename?: 'Mutation', refreshSubscriptionsNow: Array<{ __typename?: 'SubscribedPerformer', lastCheckedAt?: string | null, lastError?: string | null, reviewReleaseCount: number, pendingReleaseCount: number, processedReleaseCount: number, performer: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }, recentReleases: Array<{ __typename?: 'SubscriptionRelease', key: string, title: string, code?: string | null, date?: string | null, seenAt: string }> }> };

export type SubscribePerformersMutationVariables = Exact<{
  ids: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type SubscribePerformersMutation = { __typename?: 'Mutation', subscribePerformers: (
    { __typename?: 'PerformerBatchPayload' }
    & { ' $fragmentRefs'?: { 'PerformerBatchFieldsFragment': PerformerBatchFieldsFragment } }
  ) };

export type UnsubscribePerformersMutationVariables = Exact<{
  ids: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type UnsubscribePerformersMutation = { __typename?: 'Mutation', unsubscribePerformers: (
    { __typename?: 'PerformerBatchPayload' }
    & { ' $fragmentRefs'?: { 'PerformerBatchFieldsFragment': PerformerBatchFieldsFragment } }
  ) };

export type RefreshSubscribedPerformersMutationVariables = Exact<{
  ids: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type RefreshSubscribedPerformersMutation = { __typename?: 'Mutation', refreshSubscribedPerformers: (
    { __typename?: 'PerformerBatchPayload' }
    & { ' $fragmentRefs'?: { 'PerformerBatchFieldsFragment': PerformerBatchFieldsFragment } }
  ) };

export type PerformerBatchFieldsFragment = { __typename?: 'PerformerBatchPayload', batchId: string, summary: { __typename?: 'PerformerBatchSummary', requestedCount: number, succeededCount: number, skippedCount: number, failedCount: number }, results: Array<{ __typename?: 'PerformerBatchResult', performerId: string, status: PerformerBatchStatus, reasonCode: string, performer?: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean } | null, state?: { __typename?: 'SubscribedPerformer', lastCheckedAt?: string | null, lastError?: string | null, reviewReleaseCount: number, pendingReleaseCount: number, processedReleaseCount: number, performer: { __typename?: 'StashPerformer', id: string, name: string, aliasList: Array<string>, favorite: boolean, imagePath?: string | null, sceneCount: number, subscribed: boolean }, recentReleases: Array<{ __typename?: 'SubscriptionRelease', key: string, title: string, code?: string | null, date?: string | null, seenAt: string }> } | null }> } & { ' $fragmentName'?: 'PerformerBatchFieldsFragment' };

export type QueuePerformerScenesMutationVariables = Exact<{
  input: QueuePerformerScenesInput;
}>;


export type QueuePerformerScenesMutation = { __typename?: 'Mutation', queuePerformerScenes: { __typename?: 'QueuePerformerScenesPayload', queuedTasks: Array<{ __typename?: 'Task', id: string, source: TaskSource, stage: TaskStage, stageStatus: TaskStageStatus, code: string, createdAt: string }>, results: Array<{ __typename?: 'QueuePerformerSceneResult', key: string, status: QueuePerformerSceneStatus, reasonCode: string, resolvedCode?: string | null, task?: { __typename?: 'Task', id: string, stage: TaskStage, stageStatus: TaskStageStatus } | null }>, summary: { __typename?: 'QueuePerformerScenesSummary', requestedCount: number, queuedCount: number, skippedCount: number, failedCount: number } } };

export type TaskEventsSubscriptionVariables = Exact<{ [key: string]: never; }>;


export type TaskEventsSubscription = { __typename?: 'Subscription', taskEvents: { __typename?: 'TaskEvent', sequence: number, streamId: string, type: TaskEventType, taskId: string, listChanged: boolean, task?: (
      { __typename?: 'TaskSummary' }
      & { ' $fragmentRefs'?: { 'TaskListSummaryFragment': TaskListSummaryFragment } }
    ) | null, dashboardStats?: { __typename?: 'DashboardStats', total: number, active: number, completed: number, downloading: number, pendingScans: number, failed: number, attention: number, waiting: number } | null } };

export type TaskListSummaryFragment = { __typename?: 'TaskSummary', id: string, version: string, revision: number, source: TaskSource, code: string, stage: TaskStage, stageStatus: TaskStageStatus, stageLabel: string, stageStatusLabel: string, stageErrorCode?: string | null, stageErrorMessage?: string | null, torrentName?: string | null, progress: number, qbittorrentState?: string | null, stashIngestPhase?: string | null, stashSceneId?: string | null, sourcingNextCheckAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingCompletedAt?: string | null, downloadCompletedAt?: string | null, ingestCompletedAt?: string | null, lastProgressAt?: string | null, createdAt: string, updatedAt: string } & { ' $fragmentName'?: 'TaskListSummaryFragment' };

export type TasksWorkspaceQueryVariables = Exact<{
  input: TaskPageInput;
}>;


export type TasksWorkspaceQuery = { __typename?: 'Query', taskPage: { __typename?: 'TaskPage', totalCount: number, page: number, pageSize: number, totalPages: number, sequence: number, streamId: string, items: Array<(
      { __typename?: 'TaskSummary' }
      & { ' $fragmentRefs'?: { 'TaskListSummaryFragment': TaskListSummaryFragment } }
    )>, stats: { __typename?: 'DashboardStats', total: number, active: number, completed: number, downloading: number, pendingScans: number, failed: number, attention: number, waiting: number } }, settings: { __typename?: 'Settings', system: { __typename?: 'SystemSettings', taskDeletePolicy: TaskDeletePolicy }, automation: { __typename?: 'AutomationSettings', taskProcessingEnabled: boolean } }, settingsStatus: { __typename?: 'SettingsStatus', automation: { __typename?: 'AutomationStatus', taskProgressSyncIntervalSeconds: number, taskProgressSyncEnabled: boolean } } };

export type TaskSelectionSummariesQueryVariables = Exact<{
  ids: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type TaskSelectionSummariesQuery = { __typename?: 'Query', taskSelection: { __typename?: 'TaskPage', sequence: number, streamId: string, items: Array<(
      { __typename?: 'TaskSummary' }
      & { ' $fragmentRefs'?: { 'TaskListSummaryFragment': TaskListSummaryFragment } }
    )> } };

export type TaskAttentionQueueQueryVariables = Exact<{ [key: string]: never; }>;


export type TaskAttentionQueueQuery = { __typename?: 'Query', taskAttentionQueue: { __typename?: 'TaskAttentionQueue', sequence: number, streamId: string, items: Array<{ __typename?: 'TaskSummary', id: string, version: string, revision: number, source: TaskSource, code: string, stage: TaskStage, stageStatus: TaskStageStatus, progress: number, createdAt: string, updatedAt: string }> } };

export type TaskDetailDocumentQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type TaskDetailDocumentQuery = { __typename?: 'Query', taskDetailSnapshot: { __typename?: 'TaskDetailSnapshot', sequence: number, streamId: string, task?: { __typename?: 'Task', version: string, revision: number, id: string, source: TaskSource, code: string, stage: TaskStage, stageStatus: TaskStageStatus, stageLabel: string, stageStatusLabel: string, stageErrorCode?: string | null, stageErrorMessage?: string | null, torrentUrl: string, torrentName?: string | null, progress: number, qbittorrentState?: string | null, contentPath?: string | null, torrentHash?: string | null, savePath?: string | null, category?: string | null, tags?: string | null, downloadCompletedAt?: string | null, sourcingCompletedAt?: string | null, ingestCompletedAt?: string | null, mojiSourcePath?: string | null, transferAction?: string | null, mojiTransferPath?: string | null, transferError?: string | null, stashIngestPhase?: string | null, stashGenerateJobId?: string | null, stashIdentifyJobId?: string | null, stashSceneIds: Array<string>, stashScanJobId?: string | null, stashScanPath?: string | null, stashScanError?: string | null, stashScanHint?: string | null, stashScanStartedAt?: string | null, createdAt: string, updatedAt: string, lastProgressAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingNextCheckAt?: string | null, files: Array<{ __typename?: 'TaskFile', index: number, path: string, size: number, kind: string, part: number, selected: boolean, reason: string }>, candidate: { __typename?: 'DownloadCandidate', title: string, tracker: string, infoHash: string, link: string, magnetUri: string, size: number, seeders: number, peers: number } } | null } };

export type AddTorrentDocumentMutationVariables = Exact<{
  input: QBittorrentAddInput;
}>;


export type AddTorrentDocumentMutation = { __typename?: 'Mutation', addTorrent: { __typename?: 'Task', id: string, source: TaskSource, stage: TaskStage, stageStatus: TaskStageStatus, stageLabel: string, stageStatusLabel: string, stageErrorCode?: string | null, stageErrorMessage?: string | null, code: string, torrentName?: string | null, progress: number, createdAt: string, files: Array<{ __typename?: 'TaskFile', index: number, path: string, size: number, kind: string, part: number, selected: boolean, reason: string }> } };

export type SyncTaskProgressDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type SyncTaskProgressDocumentMutation = { __typename?: 'Mutation', syncTaskProgress: Array<{ __typename?: 'Task', id: string, source: TaskSource, stage: TaskStage, stageStatus: TaskStageStatus, stageLabel: string, stageStatusLabel: string, progress: number, qbittorrentState?: string | null, sourcingCompletedAt?: string | null, ingestCompletedAt?: string | null, updatedAt: string, lastProgressAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingNextCheckAt?: string | null }> };

export type TriggerStashScansDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type TriggerStashScansDocumentMutation = { __typename?: 'Mutation', triggerStashScans: Array<{ __typename?: 'Task', id: string, source: TaskSource, transferError?: string | null, stashIngestPhase?: string | null, stashGenerateJobId?: string | null, stashIdentifyJobId?: string | null, stashSceneIds: Array<string>, stashScanJobId?: string | null, stashScanPath?: string | null, stashScanError?: string | null, stashScanHint?: string | null, sourcingCompletedAt?: string | null, ingestCompletedAt?: string | null, updatedAt: string, lastProgressAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingNextCheckAt?: string | null }> };

export type TriggerTaskStashScanDocumentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type TriggerTaskStashScanDocumentMutation = { __typename?: 'Mutation', triggerTaskStashScan: { __typename?: 'Task', id: string, source: TaskSource, transferError?: string | null, stashIngestPhase?: string | null, stashGenerateJobId?: string | null, stashIdentifyJobId?: string | null, stashSceneIds: Array<string>, stashScanJobId?: string | null, stashScanPath?: string | null, stashScanError?: string | null, stashScanHint?: string | null, sourcingCompletedAt?: string | null, ingestCompletedAt?: string | null, updatedAt: string, lastProgressAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingNextCheckAt?: string | null } };

export type DeleteTaskDocumentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type DeleteTaskDocumentMutation = { __typename?: 'Mutation', deleteTask: { __typename?: 'Task', id: string, source: TaskSource, stage: TaskStage, stageStatus: TaskStageStatus, stageLabel: string, stageStatusLabel: string, code: string, sourcingCompletedAt?: string | null, ingestCompletedAt?: string | null, updatedAt: string, lastProgressAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingNextCheckAt?: string | null } };

export type RetryTaskDocumentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type RetryTaskDocumentMutation = { __typename?: 'Mutation', retryTask: { __typename?: 'Task', id: string, source: TaskSource, stage: TaskStage, stageStatus: TaskStageStatus, stageLabel: string, stageStatusLabel: string, stageErrorCode?: string | null, stageErrorMessage?: string | null, code: string, progress: number, qbittorrentState?: string | null, sourcingCompletedAt?: string | null, ingestCompletedAt?: string | null, updatedAt: string, lastProgressAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingNextCheckAt?: string | null, files: Array<{ __typename?: 'TaskFile', index: number, path: string, size: number, kind: string, part: number, selected: boolean, reason: string }> } };

export type RetryTasksDocumentMutationVariables = Exact<{
  ids: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type RetryTasksDocumentMutation = { __typename?: 'Mutation', retryTasks: { __typename?: 'TaskBatchPayload', batchId: string, summary: { __typename?: 'TaskBatchSummary', requestedCount: number, succeededCount: number, skippedCount: number, failedCount: number }, results: Array<{ __typename?: 'TaskBatchResult', taskId: string, status: TaskBatchStatus, reasonCode: string, task?: { __typename?: 'Task', id: string, source: TaskSource, code: string, stage: TaskStage, stageStatus: TaskStageStatus, stageLabel: string, stageStatusLabel: string, progress: number, qbittorrentState?: string | null, sourcingCompletedAt?: string | null, ingestCompletedAt?: string | null, updatedAt: string, lastProgressAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingNextCheckAt?: string | null } | null }> } };

export type ProcessTaskIngestDocumentMutationVariables = Exact<{
  ids: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type ProcessTaskIngestDocumentMutation = { __typename?: 'Mutation', processTaskIngest: { __typename?: 'TaskBatchPayload', batchId: string, summary: { __typename?: 'TaskBatchSummary', requestedCount: number, succeededCount: number, skippedCount: number, failedCount: number }, results: Array<{ __typename?: 'TaskBatchResult', taskId: string, status: TaskBatchStatus, reasonCode: string, task?: { __typename?: 'Task', id: string, source: TaskSource, code: string, stage: TaskStage, stageStatus: TaskStageStatus, stageLabel: string, stageStatusLabel: string, progress: number, transferError?: string | null, stashIngestPhase?: string | null, stashGenerateJobId?: string | null, stashIdentifyJobId?: string | null, stashSceneIds: Array<string>, stashScanJobId?: string | null, stashScanPath?: string | null, stashScanError?: string | null, stashScanHint?: string | null, sourcingCompletedAt?: string | null, ingestCompletedAt?: string | null, updatedAt: string, lastProgressAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingNextCheckAt?: string | null } | null }> } };

export type DeleteTasksDocumentMutationVariables = Exact<{
  ids: Array<Scalars['ID']['input']> | Scalars['ID']['input'];
}>;


export type DeleteTasksDocumentMutation = { __typename?: 'Mutation', deleteTasks: { __typename?: 'TaskBatchPayload', batchId: string, summary: { __typename?: 'TaskBatchSummary', requestedCount: number, succeededCount: number, skippedCount: number, failedCount: number }, results: Array<{ __typename?: 'TaskBatchResult', taskId: string, status: TaskBatchStatus, reasonCode: string, task?: { __typename?: 'Task', id: string, code: string, stage: TaskStage, stageStatus: TaskStageStatus, updatedAt: string, lastProgressAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingNextCheckAt?: string | null } | null }> } };

export type ResolveBlockedSourcingTaskDocumentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  input: ResolveBlockedSourcingTaskInput;
}>;


export type ResolveBlockedSourcingTaskDocumentMutation = { __typename?: 'Mutation', resolveBlockedSourcingTask: { __typename?: 'Task', id: string, source: TaskSource, stage: TaskStage, stageStatus: TaskStageStatus, stageLabel: string, stageStatusLabel: string, stageErrorCode?: string | null, stageErrorMessage?: string | null, code: string, torrentUrl: string, progress: number, qbittorrentState?: string | null, sourcingCompletedAt?: string | null, ingestCompletedAt?: string | null, updatedAt: string, lastProgressAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingNextCheckAt?: string | null, files: Array<{ __typename?: 'TaskFile', index: number, path: string, size: number, kind: string, part: number, selected: boolean, reason: string }>, candidate: { __typename?: 'DownloadCandidate', title: string, tracker: string, infoHash: string, size: number, seeders: number, peers: number } } };

export type ResolveTaskFilesDocumentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
  videoIndices: Array<Scalars['Int']['input']> | Scalars['Int']['input'];
}>;


export type ResolveTaskFilesDocumentMutation = { __typename?: 'Mutation', resolveTaskFiles: { __typename?: 'Task', id: string, stage: TaskStage, stageStatus: TaskStageStatus, stageErrorCode?: string | null, stageErrorMessage?: string | null, sourcingCompletedAt?: string | null, ingestCompletedAt?: string | null, updatedAt: string, lastProgressAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingNextCheckAt?: string | null, files: Array<{ __typename?: 'TaskFile', index: number, path: string, size: number, kind: string, part: number, selected: boolean, reason: string }> } };

export type TaskStashScenesDocumentQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type TaskStashScenesDocumentQuery = { __typename?: 'Query', taskStashScenes: Array<{ __typename?: 'TaskStashScene', id: string, title: string, code: string, date: string, studio: string, performers: Array<string>, imageUrl?: string | null, url?: string | null }> };

export type RestartTaskSourcingDocumentMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type RestartTaskSourcingDocumentMutation = { __typename?: 'Mutation', restartTaskSourcing: { __typename?: 'Task', id: string, source: TaskSource, code: string, stage: TaskStage, stageStatus: TaskStageStatus, stageLabel: string, stageStatusLabel: string, stageErrorCode?: string | null, stageErrorMessage?: string | null, torrentUrl: string, torrentName?: string | null, progress: number, qbittorrentState?: string | null, contentPath?: string | null, torrentHash?: string | null, savePath?: string | null, category?: string | null, tags?: string | null, downloadCompletedAt?: string | null, sourcingCompletedAt?: string | null, ingestCompletedAt?: string | null, mojiSourcePath?: string | null, transferAction?: string | null, mojiTransferPath?: string | null, transferError?: string | null, stashIngestPhase?: string | null, stashGenerateJobId?: string | null, stashIdentifyJobId?: string | null, stashSceneIds: Array<string>, stashScanJobId?: string | null, stashScanPath?: string | null, stashScanError?: string | null, stashScanHint?: string | null, stashScanStartedAt?: string | null, createdAt: string, updatedAt: string, lastProgressAt?: string | null, sourcingLastCheckedAt?: string | null, sourcingNextCheckAt?: string | null, files: Array<{ __typename?: 'TaskFile', index: number, path: string, size: number, kind: string, part: number, selected: boolean, reason: string }>, candidate: { __typename?: 'DownloadCandidate', title: string, tracker: string, infoHash: string, link: string, magnetUri: string, size: number, seeders: number, peers: number } } };

export type TorrentSearchFieldsFragment = { __typename?: 'TorrentSearch', id: string, state: TorrentSearchState, revision: number, keywords: string, taskId?: string | null, createdAt: string, updatedAt: string, expiresAt: string, finishedAt?: string | null, cancelRequested: boolean, partial: boolean, truncated: boolean, errorCode?: string | null, correlationId: string, total: number, offset: number, limit: number, sites: Array<{ __typename?: 'TorrentSearchSite', instanceId: string, name: string, kind?: IndexerSourceKind | null, state: TorrentSearchSiteState, results: number, cacheHit: boolean, fetchedAt?: string | null, elapsedMs: number, partial: boolean, truncated: boolean, errorCode?: string | null, correlationId?: string | null, retryAt?: string | null }>, candidates: Array<{ __typename?: 'TorrentSearchCandidate', id: string, instanceId: string, instanceName: string, kind: IndexerSourceKind, title: string, detailsUrl?: string | null, infoHash?: string | null, size?: number | null, seeders?: number | null, peers?: number | null, publishedAt?: string | null, categories: Array<number> }> } & { ' $fragmentName'?: 'TorrentSearchFieldsFragment' };

export type TorrentSearchQueryVariables = Exact<{
  id: Scalars['ID']['input'];
  offset?: InputMaybe<Scalars['Int']['input']>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  sortBy?: InputMaybe<TorrentSearchSort>;
}>;


export type TorrentSearchQuery = { __typename?: 'Query', torrentSearch: (
    { __typename?: 'TorrentSearch' }
    & { ' $fragmentRefs'?: { 'TorrentSearchFieldsFragment': TorrentSearchFieldsFragment } }
  ) };

export type StartTorrentSearchMutationVariables = Exact<{
  input: StartTorrentSearchInput;
}>;


export type StartTorrentSearchMutation = { __typename?: 'Mutation', startTorrentSearch: (
    { __typename?: 'TorrentSearch', id: string }
    & { ' $fragmentRefs'?: { 'TorrentSearchFieldsFragment': TorrentSearchFieldsFragment } }
  ) };

export type CancelTorrentSearchMutationVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type CancelTorrentSearchMutation = { __typename?: 'Mutation', cancelTorrentSearch: (
    { __typename?: 'TorrentSearch', id: string }
    & { ' $fragmentRefs'?: { 'TorrentSearchFieldsFragment': TorrentSearchFieldsFragment } }
  ) };

export type QueueTorrentCandidateMutationVariables = Exact<{
  input: QueueTorrentCandidateInput;
}>;


export type QueueTorrentCandidateMutation = { __typename?: 'Mutation', queueTorrentCandidate: { __typename?: 'Task', id: string, code: string, stage: TaskStage, stageStatus: TaskStageStatus } };

export type TorrentSearchEventsSubscriptionVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type TorrentSearchEventsSubscription = { __typename?: 'Subscription', torrentSearchEvents: { __typename?: 'TorrentSearchEvent', streamId: string, sequence: number, searchId: string } };

export type PreviewTorrentSelectionMutationVariables = Exact<{
  input: PreviewTorrentSelectionInput;
}>;


export type PreviewTorrentSelectionMutation = { __typename?: 'Mutation', previewTorrentSelection: { __typename?: 'TorrentSelectionPreview', searchId: string, searchRevision: number, total: number, offset: number, limit: number, candidates: Array<{ __typename?: 'TorrentSearchCandidate', id: string, instanceId: string, instanceName: string, kind: IndexerSourceKind, title: string, detailsUrl?: string | null, infoHash?: string | null, size?: number | null, seeders?: number | null, peers?: number | null, publishedAt?: string | null, categories: Array<number> }>, meta: { __typename?: 'TorrentSelectionPreviewMeta', appliedFastRules: boolean, appliedFileRules: boolean, inspectedCount: number, inspectableCount: number, inspectionCandidateLimit: number } } };

export type TorrentSearchSourcesQueryVariables = Exact<{ [key: string]: never; }>;


export type TorrentSearchSourcesQuery = { __typename?: 'Query', indexerInstances: { __typename?: 'IndexerInstances', revision: string, ruleErrorCode: string, catalogErrorCode: string, items: Array<{ __typename?: 'IndexerInstance', id: string, name: string, kind: IndexerSourceKind, enabled: boolean, errorCode: string }> } };

export type ResolveTaskSourcingCandidateMutationVariables = Exact<{
  input: ResolveTaskSourcingCandidateInput;
}>;


export type ResolveTaskSourcingCandidateMutation = { __typename?: 'Mutation', resolveTaskSourcingCandidate: { __typename?: 'Task', id: string, code: string, stage: TaskStage, stageStatus: TaskStageStatus, revision: number, version: string, sourcingNextCheckAt?: string | null, stageErrorCode?: string | null, stageErrorMessage?: string | null, updatedAt: string } };

export type PrepareTorrentDownloadMutationVariables = Exact<{
  input: PrepareTorrentDownloadInput;
}>;


export type PrepareTorrentDownloadMutation = { __typename?: 'Mutation', prepareTorrentDownload: { __typename?: 'PreparedTorrentDownload', kind: PreparedTorrentDownloadKind, downloadUrl: string, expiresAt: string } };

export type UpcomingReleasesQueryVariables = Exact<{
  input: UpcomingReleaseQueryInput;
}>;


export type UpcomingReleasesQuery = { __typename?: 'Query', upcomingReleases: { __typename?: 'UpcomingReleaseConnection', totalCount: number, page: number, pageSize: number, totalPages: number, subscribedCount: number, automatic: boolean, timezone: string, today: string, performers: Array<{ __typename?: 'UpcomingPerformer', id: string, name: string }>, items: Array<{ __typename?: 'UpcomingRelease', key: string, code: string, title: string, date: string, imageUrl: string, sourceUrl: string, state: string, decision: string, errorCode: string, metadataCheckedAt: string, performers: Array<{ __typename?: 'UpcomingPerformer', id: string, name: string }> }> } };

export type RefreshUpcomingReleaseMutationVariables = Exact<{
  key: Scalars['ID']['input'];
}>;


export type RefreshUpcomingReleaseMutation = { __typename?: 'Mutation', refreshUpcomingRelease: { __typename?: 'UpcomingRelease', key: string, code: string, title: string, date: string, imageUrl: string, sourceUrl: string, state: string, decision: string, errorCode: string, metadataCheckedAt: string, performers: Array<{ __typename?: 'UpcomingPerformer', id: string, name: string }> } };

export const TaskListSummaryFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TaskListSummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"TaskSummary"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"stashIngestPhase"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneId"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"downloadCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]} as unknown as DocumentNode<TaskListSummaryFragment, unknown>;
export const HomeTaskFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"HomeTask"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"TaskSummary"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TaskListSummary"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TaskListSummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"TaskSummary"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"stashIngestPhase"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneId"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"downloadCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]} as unknown as DocumentNode<HomeTaskFragment, unknown>;
export const HomeSubscriptionCheckRunFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"HomeSubscriptionCheckRunFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"HomeSubscriptionCheckRun"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"trigger"}},{"kind":"Field","name":{"kind":"Name","value":"scope"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"completedCount"}},{"kind":"Field","name":{"kind":"Name","value":"succeededCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"currentPerformers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]} as unknown as DocumentNode<HomeSubscriptionCheckRunFieldsFragment, unknown>;
export const IndexerCacheStatusFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerCacheStatusFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerCacheStatus"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"ttlSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"emptyTtlSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"maxSizeMb"}},{"kind":"Field","name":{"kind":"Name","value":"entries"}},{"kind":"Field","name":{"kind":"Name","value":"usedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"activeRequests"}},{"kind":"Field","name":{"kind":"Name","value":"activeWaiters"}}]}}]} as unknown as DocumentNode<IndexerCacheStatusFieldsFragment, unknown>;
export const IndexerDefinitionPolicyFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionPolicyFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionPolicy"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"schemaVersion"}},{"kind":"Field","name":{"kind":"Name","value":"checkIntervalHours"}},{"kind":"Field","name":{"kind":"Name","value":"autoUpdate"}}]}}]} as unknown as DocumentNode<IndexerDefinitionPolicyFieldsFragment, unknown>;
export const IndexerDefinitionJobFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionJobFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionJob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"targetRevision"}},{"kind":"Field","name":{"kind":"Name","value":"expectedActiveRevision"}},{"kind":"Field","name":{"kind":"Name","value":"downloadedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}}]}}]} as unknown as DocumentNode<IndexerDefinitionJobFieldsFragment, unknown>;
export const IndexerDefinitionStatusFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionStatusFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionStatus"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"streamId"}},{"kind":"Field","name":{"kind":"Name","value":"activeRevision"}},{"kind":"Field","name":{"kind":"Name","value":"previousRevision"}},{"kind":"Field","name":{"kind":"Name","value":"availableRevision"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"checkErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"installed"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"schemaVersion"}},{"kind":"Field","name":{"kind":"Name","value":"engineVersion"}},{"kind":"Field","name":{"kind":"Name","value":"definitionCount"}},{"kind":"Field","name":{"kind":"Name","value":"installedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"job"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionJobFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionJobFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionJob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"targetRevision"}},{"kind":"Field","name":{"kind":"Name","value":"expectedActiveRevision"}},{"kind":"Field","name":{"kind":"Name","value":"downloadedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}}]}}]} as unknown as DocumentNode<IndexerDefinitionStatusFieldsFragment, unknown>;
export const IndexerLoginResultFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerLoginResult"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerLoginReport"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"configRevision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleRevision"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"expiresAt"}},{"kind":"Field","name":{"kind":"Name","value":"challenge"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"prompt"}},{"kind":"Field","name":{"kind":"Name","value":"imageData"}}]}}]}}]} as unknown as DocumentNode<IndexerLoginResultFragment, unknown>;
export const IndexerObservationFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerObservationFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerObservation"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}}]} as unknown as DocumentNode<IndexerObservationFieldsFragment, unknown>;
export const IndexerSettingFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerSettingFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerSetting"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"secret"}},{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"defaultValue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"options"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"label"}}]}}]}}]} as unknown as DocumentNode<IndexerSettingFieldsFragment, unknown>;
export const IndexerInstanceFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstanceFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstance"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fetch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectionRevision"}},{"kind":"Field","name":{"kind":"Name","value":"siteUrl"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"privacy"}},{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"loginSupported"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"definitionId"}},{"kind":"Field","name":{"kind":"Name","value":"baseUrl"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfile"}},{"kind":"Field","name":{"kind":"Name","value":"trustedOrigins"}},{"kind":"Field","name":{"kind":"Name","value":"ruleDigest"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"modes"}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerSettingFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerObservationFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerObservation"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerSettingFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerSetting"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"secret"}},{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"defaultValue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"options"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"label"}}]}}]}}]} as unknown as DocumentNode<IndexerInstanceFieldsFragment, unknown>;
export const IndexerInstancesFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstancesFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstances"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"healthErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleRevision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"catalogErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"jackettEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"order"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfiles"}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstanceFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerObservationFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerObservation"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerSettingFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerSetting"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"secret"}},{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"defaultValue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"options"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"label"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstanceFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstance"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fetch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectionRevision"}},{"kind":"Field","name":{"kind":"Name","value":"siteUrl"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"privacy"}},{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"loginSupported"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"definitionId"}},{"kind":"Field","name":{"kind":"Name","value":"baseUrl"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfile"}},{"kind":"Field","name":{"kind":"Name","value":"trustedOrigins"}},{"kind":"Field","name":{"kind":"Name","value":"ruleDigest"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"modes"}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerSettingFields"}}]}}]}}]} as unknown as DocumentNode<IndexerInstancesFieldsFragment, unknown>;
export const IndexerNetworkSettingsFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerNetworkSettingsFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerNetworkSettings"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"requestTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"siteTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"challengeTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"profiles"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"proxyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"flareSolverrConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"usedBy"}}]}}]}}]} as unknown as DocumentNode<IndexerNetworkSettingsFieldsFragment, unknown>;
export const SubscriptionCheckFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"SubscriptionCheckFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"SubscriptionCheckRun"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"trigger"}},{"kind":"Field","name":{"kind":"Name","value":"scope"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"completedCount"}},{"kind":"Field","name":{"kind":"Name","value":"succeededCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}}]}}]}}]} as unknown as DocumentNode<SubscriptionCheckFieldsFragment, unknown>;
export const PerformerBatchFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"PerformerBatchFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"PerformerBatchPayload"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"batchId"}},{"kind":"Field","name":{"kind":"Name","value":"summary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"requestedCount"}},{"kind":"Field","name":{"kind":"Name","value":"succeededCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}}]}},{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reasonCode"}},{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"state"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"reviewReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<PerformerBatchFieldsFragment, unknown>;
export const TorrentSearchFieldsFragmentDoc = {"kind":"Document","definitions":[{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TorrentSearchFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"TorrentSearch"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"keywords"}},{"kind":"Field","name":{"kind":"Name","value":"taskId"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"expiresAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"cancelRequested"}},{"kind":"Field","name":{"kind":"Name","value":"partial"}},{"kind":"Field","name":{"kind":"Name","value":"truncated"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"sites"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"results"}},{"kind":"Field","name":{"kind":"Name","value":"cacheHit"}},{"kind":"Field","name":{"kind":"Name","value":"fetchedAt"}},{"kind":"Field","name":{"kind":"Name","value":"elapsedMs"}},{"kind":"Field","name":{"kind":"Name","value":"partial"}},{"kind":"Field","name":{"kind":"Name","value":"truncated"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"candidates"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"instanceName"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"detailsUrl"}},{"kind":"Field","name":{"kind":"Name","value":"infoHash"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}},{"kind":"Field","name":{"kind":"Name","value":"publishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"categories"}}]}}]}}]} as unknown as DocumentNode<TorrentSearchFieldsFragment, unknown>;
export const DownloaderDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DownloaderDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}},"defaultValue":{"kind":"IntValue","value":"0"}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}},"defaultValue":{"kind":"IntValue","value":"50"}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"search"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"taskId"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"attentionOnly"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}},"defaultValue":{"kind":"BooleanValue","value":false}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"downloader"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}},{"kind":"Argument","name":{"kind":"Name","value":"search"},"value":{"kind":"Variable","name":{"kind":"Name","value":"search"}}},{"kind":"Argument","name":{"kind":"Name","value":"taskId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"taskId"}}},{"kind":"Argument","name":{"kind":"Name","value":"attentionOnly"},"value":{"kind":"Variable","name":{"kind":"Name","value":"attentionOnly"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"checkedAt"}},{"kind":"Field","name":{"kind":"Name","value":"connected"}},{"kind":"Field","name":{"kind":"Name","value":"stale"}},{"kind":"Field","name":{"kind":"Name","value":"storageAvailable"}},{"kind":"Field","name":{"kind":"Name","value":"freeSpace"}},{"kind":"Field","name":{"kind":"Name","value":"downloadSpeed"}},{"kind":"Field","name":{"kind":"Name","value":"uploadSpeed"}},{"kind":"Field","name":{"kind":"Name","value":"downloadLimit"}},{"kind":"Field","name":{"kind":"Name","value":"uploadLimit"}},{"kind":"Field","name":{"kind":"Name","value":"connectionStatus"}},{"kind":"Field","name":{"kind":"Name","value":"dhtNodes"}},{"kind":"Field","name":{"kind":"Name","value":"completedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"remainingBytes"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"attentionCount"}},{"kind":"Field","name":{"kind":"Name","value":"policy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"ratio"}},{"kind":"Field","name":{"kind":"Name","value":"seedHours"}},{"kind":"Field","name":{"kind":"Name","value":"matchAll"}},{"kind":"Field","name":{"kind":"Name","value":"deleteFiles"}},{"kind":"Field","name":{"kind":"Name","value":"afterIngest"}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"hash"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"present"}},{"kind":"Field","name":{"kind":"Name","value":"taskIds"}},{"kind":"Field","name":{"kind":"Name","value":"taskCodes"}},{"kind":"Field","name":{"kind":"Name","value":"allCompleted"}},{"kind":"Field","name":{"kind":"Name","value":"canDeleteFiles"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"downloadSpeed"}},{"kind":"Field","name":{"kind":"Name","value":"uploadSpeed"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"completedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"downloaded"}},{"kind":"Field","name":{"kind":"Name","value":"uploaded"}},{"kind":"Field","name":{"kind":"Name","value":"ratio"}},{"kind":"Field","name":{"kind":"Name","value":"seedingSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"eta"}},{"kind":"Field","name":{"kind":"Name","value":"seeds"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}},{"kind":"Field","name":{"kind":"Name","value":"availability"}},{"kind":"Field","name":{"kind":"Name","value":"savePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"lastSeenAt"}},{"kind":"Field","name":{"kind":"Name","value":"removedAt"}},{"kind":"Field","name":{"kind":"Name","value":"removalReason"}},{"kind":"Field","name":{"kind":"Name","value":"cleanupError"}},{"kind":"Field","name":{"kind":"Name","value":"needsAttention"}}]}}]}}]}}]} as unknown as DocumentNode<DownloaderDocumentQuery, DownloaderDocumentQueryVariables>;
export const RefreshDownloaderDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshDownloaderDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshDownloader"}}]}}]} as unknown as DocumentNode<RefreshDownloaderDocumentMutation, RefreshDownloaderDocumentMutationVariables>;
export const UpdateDownloaderPolicyDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateDownloaderPolicyDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"DownloaderPolicyInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateDownloaderPolicy"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"ratio"}},{"kind":"Field","name":{"kind":"Name","value":"seedHours"}},{"kind":"Field","name":{"kind":"Name","value":"matchAll"}},{"kind":"Field","name":{"kind":"Name","value":"deleteFiles"}},{"kind":"Field","name":{"kind":"Name","value":"afterIngest"}}]}}]}}]} as unknown as DocumentNode<UpdateDownloaderPolicyDocumentMutation, UpdateDownloaderPolicyDocumentMutationVariables>;
export const ControlTorrentDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ControlTorrentDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"hash"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"action"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"TorrentAction"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"controlTorrent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"hash"},"value":{"kind":"Variable","name":{"kind":"Name","value":"hash"}}},{"kind":"Argument","name":{"kind":"Name","value":"action"},"value":{"kind":"Variable","name":{"kind":"Name","value":"action"}}}]}]}}]} as unknown as DocumentNode<ControlTorrentDocumentMutation, ControlTorrentDocumentMutationVariables>;
export const SetDownloaderSpeedLimitDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SetDownloaderSpeedLimitDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"upload"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"kibPerSecond"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"setDownloaderSpeedLimit"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"upload"},"value":{"kind":"Variable","name":{"kind":"Name","value":"upload"}}},{"kind":"Argument","name":{"kind":"Name","value":"kibPerSecond"},"value":{"kind":"Variable","name":{"kind":"Name","value":"kibPerSecond"}}}]}]}}]} as unknown as DocumentNode<SetDownloaderSpeedLimitDocumentMutation, SetDownloaderSpeedLimitDocumentMutationVariables>;
export const DownloaderSummaryDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DownloaderSummaryDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"downloader"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"checkedAt"}},{"kind":"Field","name":{"kind":"Name","value":"connected"}},{"kind":"Field","name":{"kind":"Name","value":"stale"}},{"kind":"Field","name":{"kind":"Name","value":"storageAvailable"}},{"kind":"Field","name":{"kind":"Name","value":"freeSpace"}},{"kind":"Field","name":{"kind":"Name","value":"downloadSpeed"}},{"kind":"Field","name":{"kind":"Name","value":"uploadSpeed"}},{"kind":"Field","name":{"kind":"Name","value":"completedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"remainingBytes"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"attentionCount"}},{"kind":"Field","name":{"kind":"Name","value":"connectionStatus"}}]}}]}}]} as unknown as DocumentNode<DownloaderSummaryDocumentQuery, DownloaderSummaryDocumentQueryVariables>;
export const DownloaderSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DownloaderSettingsDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"downloader"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stale"}},{"kind":"Field","name":{"kind":"Name","value":"downloadLimit"}},{"kind":"Field","name":{"kind":"Name","value":"uploadLimit"}},{"kind":"Field","name":{"kind":"Name","value":"storageAvailable"}},{"kind":"Field","name":{"kind":"Name","value":"policy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"ratio"}},{"kind":"Field","name":{"kind":"Name","value":"seedHours"}},{"kind":"Field","name":{"kind":"Name","value":"matchAll"}},{"kind":"Field","name":{"kind":"Name","value":"deleteFiles"}},{"kind":"Field","name":{"kind":"Name","value":"afterIngest"}}]}}]}}]}}]} as unknown as DocumentNode<DownloaderSettingsDocumentQuery, DownloaderSettingsDocumentQueryVariables>;
export const HomeIndexerSummaryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"HomeIndexerSummary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"homeIndexerSummary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"definitionJob"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"definitionErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"builtinEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"builtinUnverified"}},{"kind":"Field","name":{"kind":"Name","value":"builtinSearchSucceeded"}},{"kind":"Field","name":{"kind":"Name","value":"healthErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"attention"}},{"kind":"Field","name":{"kind":"Name","value":"issues"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"operation"}},{"kind":"Field","name":{"kind":"Name","value":"observation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sites"}},{"kind":"Field","name":{"kind":"Name","value":"stale"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}}]}}]}}]}}]} as unknown as DocumentNode<HomeIndexerSummaryQuery, HomeIndexerSummaryQueryVariables>;
export const HomeIndexerDefinitionEventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"subscription","name":{"kind":"Name","value":"HomeIndexerDefinitionEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerDefinitionEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"streamId"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"job"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}}]}}]}}]}}]} as unknown as DocumentNode<HomeIndexerDefinitionEventsSubscription, HomeIndexerDefinitionEventsSubscriptionVariables>;
export const HomePageDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"HomePageDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"system"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskDeletePolicy"}}]}},{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskProcessingEnabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"url"}}]}},{"kind":"Field","name":{"kind":"Name","value":"ingest"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"keepSubtitles"}},{"kind":"Field","name":{"kind":"Name","value":"downloads"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"qbRoot"}},{"kind":"Field","name":{"kind":"Name","value":"mojiRoot"}}]}},{"kind":"Field","name":{"kind":"Name","value":"library"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"mojiRoot"}},{"kind":"Field","name":{"kind":"Name","value":"stashRoot"}}]}},{"kind":"Field","name":{"kind":"Name","value":"transfer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"action"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"url"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"username"}},{"kind":"Field","name":{"kind":"Name","value":"defaultSavePath"}}]}}]}},{"kind":"Field","alias":{"kind":"Name","value":"homeOverview"},"name":{"kind":"Name","value":"homeTaskSnapshot"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"streamId"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"attention"}},{"kind":"Field","name":{"kind":"Name","value":"active"}},{"kind":"Field","name":{"kind":"Name","value":"waiting"}},{"kind":"Field","name":{"kind":"Name","value":"completedLast24Hours"}},{"kind":"Field","name":{"kind":"Name","value":"attentionTasks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"HomeTask"}}]}},{"kind":"Field","name":{"kind":"Name","value":"activeTasks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"HomeTask"}}]}},{"kind":"Field","name":{"kind":"Name","value":"recentTasks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"HomeTask"}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TaskListSummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"TaskSummary"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"stashIngestPhase"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneId"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"downloadCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"HomeTask"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"TaskSummary"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TaskListSummary"}}]}}]} as unknown as DocumentNode<HomePageDocumentQuery, HomePageDocumentQueryVariables>;
export const HomeSubscriptionSummaryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"HomeSubscriptionSummary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"homeSubscriptionSummary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"attention"}},{"kind":"Field","name":{"kind":"Name","value":"unchecked"}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"attentionPerformerIds"}}]}}]}}]} as unknown as DocumentNode<HomeSubscriptionSummaryQuery, HomeSubscriptionSummaryQueryVariables>;
export const HomeSubscriptionChecksDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"HomeSubscriptionChecks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"homeSubscriptionChecks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"nextRunAt"}},{"kind":"Field","name":{"kind":"Name","value":"activeCount"}},{"kind":"Field","name":{"kind":"Name","value":"latestScheduled"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"HomeSubscriptionCheckRunFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"activeRuns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"HomeSubscriptionCheckRunFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"recentRuns"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"HomeSubscriptionCheckRunFields"}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"HomeSubscriptionCheckRunFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"HomeSubscriptionCheckRun"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"trigger"}},{"kind":"Field","name":{"kind":"Name","value":"scope"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"completedCount"}},{"kind":"Field","name":{"kind":"Name","value":"succeededCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"currentPerformers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]} as unknown as DocumentNode<HomeSubscriptionChecksQuery, HomeSubscriptionChecksQueryVariables>;
export const HomeServiceStatusDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"HomeServiceStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"settingsStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"ready"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"ready"}}]}},{"kind":"Field","name":{"kind":"Name","value":"ingest"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stashStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingMojiScanCount"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"okAt"}}]}}]}}]}}]} as unknown as DocumentNode<HomeServiceStatusQuery, HomeServiceStatusQueryVariables>;
export const IndexerCacheStatusDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"IndexerCacheStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerCacheStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerCacheStatusFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerCacheStatusFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerCacheStatus"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"ttlSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"emptyTtlSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"maxSizeMb"}},{"kind":"Field","name":{"kind":"Name","value":"entries"}},{"kind":"Field","name":{"kind":"Name","value":"usedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"activeRequests"}},{"kind":"Field","name":{"kind":"Name","value":"activeWaiters"}}]}}]} as unknown as DocumentNode<IndexerCacheStatusQuery, IndexerCacheStatusQueryVariables>;
export const ClearIndexerSearchCacheDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ClearIndexerSearchCache"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"instanceId"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clearIndexerSearchCache"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"instanceId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"instanceId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerCacheStatusFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerCacheStatusFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerCacheStatus"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"ttlSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"emptyTtlSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"maxSizeMb"}},{"kind":"Field","name":{"kind":"Name","value":"entries"}},{"kind":"Field","name":{"kind":"Name","value":"usedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"activeRequests"}},{"kind":"Field","name":{"kind":"Name","value":"activeWaiters"}}]}}]} as unknown as DocumentNode<ClearIndexerSearchCacheMutation, ClearIndexerSearchCacheMutationVariables>;
export const UpdateIndexerSearchCacheDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateIndexerSearchCache"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerSearchCachePatchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateIndexerSearchCache"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"expectedRevision"},"value":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerCacheStatusFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerCacheStatusFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerCacheStatus"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"ttlSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"emptyTtlSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"maxSizeMb"}},{"kind":"Field","name":{"kind":"Name","value":"entries"}},{"kind":"Field","name":{"kind":"Name","value":"usedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"activeRequests"}},{"kind":"Field","name":{"kind":"Name","value":"activeWaiters"}}]}}]} as unknown as DocumentNode<UpdateIndexerSearchCacheMutation, UpdateIndexerSearchCacheMutationVariables>;
export const TestIndexerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TestIndexer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"TestIndexerInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"testIndexer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"configRevision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleRevision"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"results"}},{"kind":"Field","name":{"kind":"Name","value":"partial"}},{"kind":"Field","name":{"kind":"Name","value":"truncated"}},{"kind":"Field","name":{"kind":"Name","value":"stages"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"elapsedMs"}}]}}]}}]}}]} as unknown as DocumentNode<TestIndexerMutation, TestIndexerMutationVariables>;
export const IndexerCheckActivityDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"subscription","name":{"kind":"Name","value":"IndexerCheckActivity"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"clientId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerCheckActivity"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"clientId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"clientId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"requestId"}},{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"state"}}]}}]}}]} as unknown as DocumentNode<IndexerCheckActivitySubscription, IndexerCheckActivitySubscriptionVariables>;
export const IndexerDefinitionPolicyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"IndexerDefinitionPolicy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerDefinitionPolicy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionPolicyFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionPolicyFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionPolicy"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"schemaVersion"}},{"kind":"Field","name":{"kind":"Name","value":"checkIntervalHours"}},{"kind":"Field","name":{"kind":"Name","value":"autoUpdate"}}]}}]} as unknown as DocumentNode<IndexerDefinitionPolicyQuery, IndexerDefinitionPolicyQueryVariables>;
export const UpdateIndexerDefinitionPolicyDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateIndexerDefinitionPolicy"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionPolicyPatchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateIndexerDefinitionPolicy"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"expectedRevision"},"value":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionPolicyFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionPolicyFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionPolicy"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"schemaVersion"}},{"kind":"Field","name":{"kind":"Name","value":"checkIntervalHours"}},{"kind":"Field","name":{"kind":"Name","value":"autoUpdate"}}]}}]} as unknown as DocumentNode<UpdateIndexerDefinitionPolicyMutation, UpdateIndexerDefinitionPolicyMutationVariables>;
export const IndexerDefinitionStatusDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"IndexerDefinitionStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerDefinitionStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionStatusFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionJobFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionJob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"targetRevision"}},{"kind":"Field","name":{"kind":"Name","value":"expectedActiveRevision"}},{"kind":"Field","name":{"kind":"Name","value":"downloadedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionStatusFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionStatus"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"streamId"}},{"kind":"Field","name":{"kind":"Name","value":"activeRevision"}},{"kind":"Field","name":{"kind":"Name","value":"previousRevision"}},{"kind":"Field","name":{"kind":"Name","value":"availableRevision"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"checkErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"installed"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"schemaVersion"}},{"kind":"Field","name":{"kind":"Name","value":"engineVersion"}},{"kind":"Field","name":{"kind":"Name","value":"definitionCount"}},{"kind":"Field","name":{"kind":"Name","value":"installedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"job"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionJobFields"}}]}}]}}]} as unknown as DocumentNode<IndexerDefinitionStatusQuery, IndexerDefinitionStatusQueryVariables>;
export const IndexerDefinitionUpdateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"IndexerDefinitionUpdate"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerDefinitionUpdate"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionJobFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionJobFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionJob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"targetRevision"}},{"kind":"Field","name":{"kind":"Name","value":"expectedActiveRevision"}},{"kind":"Field","name":{"kind":"Name","value":"downloadedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}}]}}]} as unknown as DocumentNode<IndexerDefinitionUpdateQuery, IndexerDefinitionUpdateQueryVariables>;
export const CheckIndexerDefinitionUpdateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CheckIndexerDefinitionUpdate"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"checkIndexerDefinitionUpdate"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionStatusFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionJobFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionJob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"targetRevision"}},{"kind":"Field","name":{"kind":"Name","value":"expectedActiveRevision"}},{"kind":"Field","name":{"kind":"Name","value":"downloadedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionStatusFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionStatus"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"streamId"}},{"kind":"Field","name":{"kind":"Name","value":"activeRevision"}},{"kind":"Field","name":{"kind":"Name","value":"previousRevision"}},{"kind":"Field","name":{"kind":"Name","value":"availableRevision"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"checkErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"installed"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"schemaVersion"}},{"kind":"Field","name":{"kind":"Name","value":"engineVersion"}},{"kind":"Field","name":{"kind":"Name","value":"definitionCount"}},{"kind":"Field","name":{"kind":"Name","value":"installedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"job"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionJobFields"}}]}}]}}]} as unknown as DocumentNode<CheckIndexerDefinitionUpdateMutation, CheckIndexerDefinitionUpdateMutationVariables>;
export const InstallIndexerDefinitionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"InstallIndexerDefinitions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionChangeInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"installIndexerDefinitions"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionJobFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionJobFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionJob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"targetRevision"}},{"kind":"Field","name":{"kind":"Name","value":"expectedActiveRevision"}},{"kind":"Field","name":{"kind":"Name","value":"downloadedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}}]}}]} as unknown as DocumentNode<InstallIndexerDefinitionsMutation, InstallIndexerDefinitionsMutationVariables>;
export const UpdateIndexerDefinitionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateIndexerDefinitions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionChangeInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateIndexerDefinitions"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionJobFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionJobFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionJob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"targetRevision"}},{"kind":"Field","name":{"kind":"Name","value":"expectedActiveRevision"}},{"kind":"Field","name":{"kind":"Name","value":"downloadedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}}]}}]} as unknown as DocumentNode<UpdateIndexerDefinitionsMutation, UpdateIndexerDefinitionsMutationVariables>;
export const RollbackIndexerDefinitionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RollbackIndexerDefinitions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionChangeInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"rollbackIndexerDefinitions"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionJobFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionJobFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionJob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"targetRevision"}},{"kind":"Field","name":{"kind":"Name","value":"expectedActiveRevision"}},{"kind":"Field","name":{"kind":"Name","value":"downloadedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}}]}}]} as unknown as DocumentNode<RollbackIndexerDefinitionsMutation, RollbackIndexerDefinitionsMutationVariables>;
export const RepairIndexerDefinitionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RepairIndexerDefinitions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionChangeInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"repairIndexerDefinitions"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionJobFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionJobFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionJob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"targetRevision"}},{"kind":"Field","name":{"kind":"Name","value":"expectedActiveRevision"}},{"kind":"Field","name":{"kind":"Name","value":"downloadedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}}]}}]} as unknown as DocumentNode<RepairIndexerDefinitionsMutation, RepairIndexerDefinitionsMutationVariables>;
export const CancelIndexerDefinitionUpdateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CancelIndexerDefinitionUpdate"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancelIndexerDefinitionUpdate"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionJobFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionJobFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionJob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"targetRevision"}},{"kind":"Field","name":{"kind":"Name","value":"expectedActiveRevision"}},{"kind":"Field","name":{"kind":"Name","value":"downloadedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}}]}}]} as unknown as DocumentNode<CancelIndexerDefinitionUpdateMutation, CancelIndexerDefinitionUpdateMutationVariables>;
export const IndexerDefinitionEventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"subscription","name":{"kind":"Name","value":"IndexerDefinitionEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerDefinitionEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionStatusFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionJobFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionJob"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"targetRevision"}},{"kind":"Field","name":{"kind":"Name","value":"expectedActiveRevision"}},{"kind":"Field","name":{"kind":"Name","value":"downloadedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerDefinitionStatusFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionStatus"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"streamId"}},{"kind":"Field","name":{"kind":"Name","value":"activeRevision"}},{"kind":"Field","name":{"kind":"Name","value":"previousRevision"}},{"kind":"Field","name":{"kind":"Name","value":"availableRevision"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"available"}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"checkErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"installed"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"schemaVersion"}},{"kind":"Field","name":{"kind":"Name","value":"engineVersion"}},{"kind":"Field","name":{"kind":"Name","value":"definitionCount"}},{"kind":"Field","name":{"kind":"Name","value":"installedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"job"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerDefinitionJobFields"}}]}}]}}]} as unknown as DocumentNode<IndexerDefinitionEventsSubscription, IndexerDefinitionEventsSubscriptionVariables>;
export const IndexerEventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"subscription","name":{"kind":"Name","value":"IndexerEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"streamId"}},{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}}]}}]}}]} as unknown as DocumentNode<IndexerEventsSubscription, IndexerEventsSubscriptionVariables>;
export const BeginIndexerLoginDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"BeginIndexerLogin"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerLoginInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"beginIndexerLogin"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerLoginResult"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerLoginResult"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerLoginReport"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"configRevision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleRevision"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"expiresAt"}},{"kind":"Field","name":{"kind":"Name","value":"challenge"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"prompt"}},{"kind":"Field","name":{"kind":"Name","value":"imageData"}}]}}]}}]} as unknown as DocumentNode<BeginIndexerLoginMutation, BeginIndexerLoginMutationVariables>;
export const AnswerIndexerChallengeDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AnswerIndexerChallenge"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerLoginInput"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"answer"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"answerIndexerChallenge"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}},{"kind":"Argument","name":{"kind":"Name","value":"answer"},"value":{"kind":"Variable","name":{"kind":"Name","value":"answer"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerLoginResult"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerLoginResult"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerLoginReport"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"configRevision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleRevision"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"expiresAt"}},{"kind":"Field","name":{"kind":"Name","value":"challenge"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"prompt"}},{"kind":"Field","name":{"kind":"Name","value":"imageData"}}]}}]}}]} as unknown as DocumentNode<AnswerIndexerChallengeMutation, AnswerIndexerChallengeMutationVariables>;
export const CancelIndexerLoginDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CancelIndexerLogin"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerLoginInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancelIndexerLogin"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}]}]}}]} as unknown as DocumentNode<CancelIndexerLoginMutation, CancelIndexerLoginMutationVariables>;
export const IndexerLoginDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"IndexerLogin"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerLoginInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerLogin"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerLoginResult"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerLoginResult"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerLoginReport"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"configRevision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleRevision"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"expiresAt"}},{"kind":"Field","name":{"kind":"Name","value":"challenge"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"prompt"}},{"kind":"Field","name":{"kind":"Name","value":"imageData"}}]}}]}}]} as unknown as DocumentNode<IndexerLoginQuery, IndexerLoginQueryVariables>;
export const IndexerDefinitionsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"IndexerDefinitions"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerDefinitionFilterInput"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerDefinitions"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"privacy"}},{"kind":"Field","name":{"kind":"Name","value":"loginSupported"}},{"kind":"Field","name":{"kind":"Name","value":"digest"}},{"kind":"Field","name":{"kind":"Name","value":"links"}},{"kind":"Field","name":{"kind":"Name","value":"categories"}},{"kind":"Field","name":{"kind":"Name","value":"modes"}},{"kind":"Field","name":{"kind":"Name","value":"warnings"}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerSettingFields"}}]}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerSettingFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerSetting"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"secret"}},{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"defaultValue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"options"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"label"}}]}}]}}]} as unknown as DocumentNode<IndexerDefinitionsQuery, IndexerDefinitionsQueryVariables>;
export const IndexerInstancesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"IndexerInstances"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerInstances"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstancesFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerObservationFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerObservation"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerSettingFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerSetting"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"secret"}},{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"defaultValue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"options"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"label"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstanceFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstance"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fetch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectionRevision"}},{"kind":"Field","name":{"kind":"Name","value":"siteUrl"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"privacy"}},{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"loginSupported"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"definitionId"}},{"kind":"Field","name":{"kind":"Name","value":"baseUrl"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfile"}},{"kind":"Field","name":{"kind":"Name","value":"trustedOrigins"}},{"kind":"Field","name":{"kind":"Name","value":"ruleDigest"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"modes"}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerSettingFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstancesFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstances"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"healthErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleRevision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"catalogErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"jackettEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"order"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfiles"}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstanceFields"}}]}}]}}]} as unknown as DocumentNode<IndexerInstancesQuery, IndexerInstancesQueryVariables>;
export const AddBuiltinIndexerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddBuiltinIndexer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"BuiltinIndexerInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"addBuiltinIndexer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"expectedRevision"},"value":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"instance"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstanceFields"}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerObservationFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerObservation"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerSettingFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerSetting"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"secret"}},{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"defaultValue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"options"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"label"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstanceFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstance"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fetch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectionRevision"}},{"kind":"Field","name":{"kind":"Name","value":"siteUrl"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"privacy"}},{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"loginSupported"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"definitionId"}},{"kind":"Field","name":{"kind":"Name","value":"baseUrl"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfile"}},{"kind":"Field","name":{"kind":"Name","value":"trustedOrigins"}},{"kind":"Field","name":{"kind":"Name","value":"ruleDigest"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"modes"}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerSettingFields"}}]}}]}}]} as unknown as DocumentNode<AddBuiltinIndexerMutation, AddBuiltinIndexerMutationVariables>;
export const UpdateBuiltinIndexerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateBuiltinIndexer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"BuiltinIndexerPatchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateBuiltinIndexer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"expectedRevision"},"value":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"instance"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstanceFields"}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerObservationFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerObservation"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerSettingFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerSetting"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"secret"}},{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"defaultValue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"options"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"label"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstanceFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstance"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fetch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectionRevision"}},{"kind":"Field","name":{"kind":"Name","value":"siteUrl"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"privacy"}},{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"loginSupported"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"definitionId"}},{"kind":"Field","name":{"kind":"Name","value":"baseUrl"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfile"}},{"kind":"Field","name":{"kind":"Name","value":"trustedOrigins"}},{"kind":"Field","name":{"kind":"Name","value":"ruleDigest"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"modes"}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerSettingFields"}}]}}]}}]} as unknown as DocumentNode<UpdateBuiltinIndexerMutation, UpdateBuiltinIndexerMutationVariables>;
export const RemoveBuiltinIndexerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RemoveBuiltinIndexer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"removeBuiltinIndexer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"expectedRevision"},"value":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstancesFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerObservationFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerObservation"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerSettingFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerSetting"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"secret"}},{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"defaultValue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"options"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"label"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstanceFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstance"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fetch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectionRevision"}},{"kind":"Field","name":{"kind":"Name","value":"siteUrl"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"privacy"}},{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"loginSupported"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"definitionId"}},{"kind":"Field","name":{"kind":"Name","value":"baseUrl"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfile"}},{"kind":"Field","name":{"kind":"Name","value":"trustedOrigins"}},{"kind":"Field","name":{"kind":"Name","value":"ruleDigest"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"modes"}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerSettingFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstancesFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstances"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"healthErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleRevision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"catalogErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"jackettEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"order"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfiles"}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstanceFields"}}]}}]}}]} as unknown as DocumentNode<RemoveBuiltinIndexerMutation, RemoveBuiltinIndexerMutationVariables>;
export const SetIndexerEnabledDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SetIndexerEnabled"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"enabled"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Boolean"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"setIndexerEnabled"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"expectedRevision"},"value":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}}},{"kind":"Argument","name":{"kind":"Name","value":"enabled"},"value":{"kind":"Variable","name":{"kind":"Name","value":"enabled"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstancesFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerObservationFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerObservation"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerSettingFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerSetting"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"secret"}},{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"defaultValue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"options"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"label"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstanceFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstance"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fetch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectionRevision"}},{"kind":"Field","name":{"kind":"Name","value":"siteUrl"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"privacy"}},{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"loginSupported"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"definitionId"}},{"kind":"Field","name":{"kind":"Name","value":"baseUrl"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfile"}},{"kind":"Field","name":{"kind":"Name","value":"trustedOrigins"}},{"kind":"Field","name":{"kind":"Name","value":"ruleDigest"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"modes"}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerSettingFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstancesFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstances"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"healthErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleRevision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"catalogErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"jackettEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"order"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfiles"}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstanceFields"}}]}}]}}]} as unknown as DocumentNode<SetIndexerEnabledMutation, SetIndexerEnabledMutationVariables>;
export const ReorderIndexersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ReorderIndexers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"reorderIndexers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"expectedRevision"},"value":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}}},{"kind":"Argument","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstancesFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerObservationFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerObservation"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerSettingFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerSetting"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"secret"}},{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"defaultValue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"options"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"label"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstanceFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstance"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fetch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectionRevision"}},{"kind":"Field","name":{"kind":"Name","value":"siteUrl"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"privacy"}},{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"loginSupported"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"definitionId"}},{"kind":"Field","name":{"kind":"Name","value":"baseUrl"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfile"}},{"kind":"Field","name":{"kind":"Name","value":"trustedOrigins"}},{"kind":"Field","name":{"kind":"Name","value":"ruleDigest"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"modes"}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerSettingFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstancesFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstances"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"healthErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleRevision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"catalogErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"jackettEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"order"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfiles"}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstanceFields"}}]}}]}}]} as unknown as DocumentNode<ReorderIndexersMutation, ReorderIndexersMutationVariables>;
export const RefreshJackettIndexersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshJackettIndexers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshJackettIndexers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstancesFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerObservationFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerObservation"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerSettingFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerSetting"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"secret"}},{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"value"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"defaultValue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"string"}},{"kind":"Field","name":{"kind":"Name","value":"boolean"}},{"kind":"Field","name":{"kind":"Name","value":"number"}}]}},{"kind":"Field","name":{"kind":"Name","value":"options"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"label"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstanceFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstance"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fetch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerObservationFields"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"connectionRevision"}},{"kind":"Field","name":{"kind":"Name","value":"siteUrl"}},{"kind":"Field","name":{"kind":"Name","value":"description"}},{"kind":"Field","name":{"kind":"Name","value":"privacy"}},{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"loginSupported"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"definitionId"}},{"kind":"Field","name":{"kind":"Name","value":"baseUrl"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfile"}},{"kind":"Field","name":{"kind":"Name","value":"trustedOrigins"}},{"kind":"Field","name":{"kind":"Name","value":"ruleDigest"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"modes"}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerSettingFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerInstancesFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerInstances"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"healthErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleRevision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"catalogErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"jackettEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"order"}},{"kind":"Field","name":{"kind":"Name","value":"networkProfiles"}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerInstanceFields"}}]}}]}}]} as unknown as DocumentNode<RefreshJackettIndexersMutation, RefreshJackettIndexersMutationVariables>;
export const IndexerNetworkSettingsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"IndexerNetworkSettings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerNetworkSettings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerNetworkSettingsFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerNetworkSettingsFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerNetworkSettings"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"requestTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"siteTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"challengeTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"profiles"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"proxyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"flareSolverrConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"usedBy"}}]}}]}}]} as unknown as DocumentNode<IndexerNetworkSettingsQuery, IndexerNetworkSettingsQueryVariables>;
export const UpdateIndexerNetworkTimeoutsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateIndexerNetworkTimeouts"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerNetworkTimeoutPatchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateIndexerNetworkTimeouts"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"expectedRevision"},"value":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerNetworkSettingsFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerNetworkSettingsFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerNetworkSettings"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"requestTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"siteTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"challengeTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"profiles"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"proxyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"flareSolverrConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"usedBy"}}]}}]}}]} as unknown as DocumentNode<UpdateIndexerNetworkTimeoutsMutation, UpdateIndexerNetworkTimeoutsMutationVariables>;
export const AddIndexerNetworkProfileDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddIndexerNetworkProfile"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerNetworkProfilePatchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"addIndexerNetworkProfile"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"expectedRevision"},"value":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerNetworkSettingsFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerNetworkSettingsFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerNetworkSettings"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"requestTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"siteTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"challengeTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"profiles"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"proxyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"flareSolverrConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"usedBy"}}]}}]}}]} as unknown as DocumentNode<AddIndexerNetworkProfileMutation, AddIndexerNetworkProfileMutationVariables>;
export const UpdateIndexerNetworkProfileDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateIndexerNetworkProfile"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerNetworkProfilePatchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateIndexerNetworkProfile"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"expectedRevision"},"value":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerNetworkSettingsFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerNetworkSettingsFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerNetworkSettings"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"requestTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"siteTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"challengeTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"profiles"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"proxyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"flareSolverrConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"usedBy"}}]}}]}}]} as unknown as DocumentNode<UpdateIndexerNetworkProfileMutation, UpdateIndexerNetworkProfileMutationVariables>;
export const RemoveIndexerNetworkProfileDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RemoveIndexerNetworkProfile"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"removeIndexerNetworkProfile"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"expectedRevision"},"value":{"kind":"Variable","name":{"kind":"Name","value":"expectedRevision"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"IndexerNetworkSettingsFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"IndexerNetworkSettingsFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"IndexerNetworkSettings"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"requestTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"siteTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"challengeTimeoutSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"profiles"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"proxyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"flareSolverrConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"usedBy"}}]}}]}}]} as unknown as DocumentNode<RemoveIndexerNetworkProfileMutation, RemoveIndexerNetworkProfileMutationVariables>;
export const LogEventsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"subscription","name":{"kind":"Name","value":"LogEventsDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"logEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"entry"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"level"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]}}]} as unknown as DocumentNode<LogEventsDocumentSubscription, LogEventsDocumentSubscriptionVariables>;
export const InterfaceSettingsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"InterfaceSettings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"system"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"developerMode"}}]}}]}}]}}]} as unknown as DocumentNode<InterfaceSettingsQuery, InterfaceSettingsQueryVariables>;
export const DatabasePerformanceDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"subscription","name":{"kind":"Name","value":"DatabasePerformanceDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"databasePerformance"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sampledAt"}},{"kind":"Field","name":{"kind":"Name","value":"intervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"databases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"role"}},{"kind":"Field","name":{"kind":"Name","value":"databaseBytes"}},{"kind":"Field","name":{"kind":"Name","value":"walBytes"}},{"kind":"Field","name":{"kind":"Name","value":"freeBytes"}},{"kind":"Field","name":{"kind":"Name","value":"writeAttempts"}},{"kind":"Field","name":{"kind":"Name","value":"writeErrors"}},{"kind":"Field","name":{"kind":"Name","value":"busyRetries"}},{"kind":"Field","name":{"kind":"Name","value":"queueRejected"}},{"kind":"Field","name":{"kind":"Name","value":"queueCanceled"}},{"kind":"Field","name":{"kind":"Name","value":"queued"}},{"kind":"Field","name":{"kind":"Name","value":"queueCapacity"}},{"kind":"Field","name":{"kind":"Name","value":"writesPerSecond"}},{"kind":"Field","name":{"kind":"Name","value":"averageWaitMilliseconds"}},{"kind":"Field","name":{"kind":"Name","value":"averageExecutionMilliseconds"}},{"kind":"Field","name":{"kind":"Name","value":"totalWaitMilliseconds"}},{"kind":"Field","name":{"kind":"Name","value":"totalExecutionMilliseconds"}},{"kind":"Field","name":{"kind":"Name","value":"reader"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"open"}},{"kind":"Field","name":{"kind":"Name","value":"inUse"}},{"kind":"Field","name":{"kind":"Name","value":"idle"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"waitCount"}},{"kind":"Field","name":{"kind":"Name","value":"waitMilliseconds"}}]}},{"kind":"Field","name":{"kind":"Name","value":"writer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"open"}},{"kind":"Field","name":{"kind":"Name","value":"inUse"}},{"kind":"Field","name":{"kind":"Name","value":"idle"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"waitCount"}},{"kind":"Field","name":{"kind":"Name","value":"waitMilliseconds"}}]}}]}}]}}]}}]} as unknown as DocumentNode<DatabasePerformanceDocumentSubscription, DatabasePerformanceDocumentSubscriptionVariables>;
export const PerformerSubscriptionEventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"subscription","name":{"kind":"Name","value":"PerformerSubscriptionEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performerSubscriptionEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"state"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"reviewReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<PerformerSubscriptionEventsSubscription, PerformerSubscriptionEventsSubscriptionVariables>;
export const ReleaseReviewsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ReleaseReviews"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ReleaseReviewQueryInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"releaseReviews"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}},{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"pageSize"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"performers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"imageUrl"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"vr"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reasonCode"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"taskId"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}},{"kind":"Field","name":{"kind":"Name","value":"reviewedAt"}},{"kind":"Field","name":{"kind":"Name","value":"performerNames"}},{"kind":"Field","name":{"kind":"Name","value":"performers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]} as unknown as DocumentNode<ReleaseReviewsQuery, ReleaseReviewsQueryVariables>;
export const ReleaseReviewSummaryDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ReleaseReviewSummary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"releaseReviews"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"pageSize"},"value":{"kind":"IntValue","value":"1"}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pendingCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}},{"kind":"Field","name":{"kind":"Name","value":"performers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"pendingCount"}}]}}]}}]}}]} as unknown as DocumentNode<ReleaseReviewSummaryQuery, ReleaseReviewSummaryQueryVariables>;
export const ProcessReleaseReviewsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ProcessReleaseReviews"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ProcessReleaseReviewsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"processReleaseReviews"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"succeededCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}},{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"outcome"}},{"kind":"Field","name":{"kind":"Name","value":"reasonCode"}},{"kind":"Field","name":{"kind":"Name","value":"release"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"taskId"}}]}}]}}]}}]}}]} as unknown as DocumentNode<ProcessReleaseReviewsMutation, ProcessReleaseReviewsMutationVariables>;
export const DiscoverConfigDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DiscoverConfigDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"torrentSelection"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"inspectionCandidateLimit"}}]}}]}}]}}]}}]} as unknown as DocumentNode<DiscoverConfigDocumentQuery, DiscoverConfigDocumentQueryVariables>;
export const PerformersConfigDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"PerformersConfigDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"url"}}]}}]}}]}}]} as unknown as DocumentNode<PerformersConfigDocumentQuery, PerformersConfigDocumentQueryVariables>;
export const StatsPageDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"StatsPageDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"dashboardStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"active"}},{"kind":"Field","name":{"kind":"Name","value":"completed"}},{"kind":"Field","name":{"kind":"Name","value":"pendingScans"}},{"kind":"Field","name":{"kind":"Name","value":"failed"}}]}}]}}]} as unknown as DocumentNode<StatsPageDocumentQuery, StatsPageDocumentQueryVariables>;
export const TaskProcessingStateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TaskProcessingState"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskProcessingEnabled"}}]}}]}}]}}]} as unknown as DocumentNode<TaskProcessingStateQuery, TaskProcessingStateQueryVariables>;
export const DiscoverScenesDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DiscoverScenesDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"DiscoverScenesInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"discoverScenes"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"sceneId"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxEndpoint"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxName"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"durationSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"studioName"}},{"kind":"Field","name":{"kind":"Name","value":"imageUrl"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"performerNames"}},{"kind":"Field","name":{"kind":"Name","value":"derivedQuery"}}]}},{"kind":"Field","name":{"kind":"Name","value":"usedStashBox"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"performerName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"fallbackCount"}},{"kind":"Field","name":{"kind":"Name","value":"searchedQuery"}}]}}]}}]} as unknown as DocumentNode<DiscoverScenesDocumentQuery, DiscoverScenesDocumentQueryVariables>;
export const QueueDiscoveredSceneDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"QueueDiscoveredSceneDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"QueueDiscoveredSceneInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"queueDiscoveredScene"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<QueueDiscoveredSceneDocumentMutation, QueueDiscoveredSceneDocumentMutationVariables>;
export const ServiceStatusEventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"subscription","name":{"kind":"Name","value":"ServiceStatusEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"serviceStatusEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"services"}},{"kind":"Field","name":{"kind":"Name","value":"observedAt"}}]}}]}}]} as unknown as DocumentNode<ServiceStatusEventsSubscription, ServiceStatusEventsSubscriptionVariables>;
export const ServiceConnectionSettingsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ServiceConnectionSettings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"apiKey"}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"apiKey"}},{"kind":"Field","name":{"kind":"Name","value":"passwordConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"password"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"username"}},{"kind":"Field","name":{"kind":"Name","value":"usernameConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"passwordConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"password"}},{"kind":"Field","name":{"kind":"Name","value":"defaultSavePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}}]}}]}}]}}]} as unknown as DocumentNode<ServiceConnectionSettingsQuery, ServiceConnectionSettingsQueryVariables>;
export const ServiceConnectionStatusDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ServiceConnectionStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"settingsStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"ready"}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"ready"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"ready"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stashStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"okAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackettStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"okAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"okAt"}}]}}]}}]}}]} as unknown as DocumentNode<ServiceConnectionStatusQuery, ServiceConnectionStatusQueryVariables>;
export const LibraryDeliverySettingsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"LibraryDeliverySettings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"delivery"},"name":{"kind":"Name","value":"ingest"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"keepSubtitles"}},{"kind":"Field","name":{"kind":"Name","value":"generateTranscodes"}},{"kind":"Field","name":{"kind":"Name","value":"downloads"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"qbRoot"}},{"kind":"Field","name":{"kind":"Name","value":"mojiRoot"}}]}},{"kind":"Field","name":{"kind":"Name","value":"library"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"mojiRoot"}},{"kind":"Field","name":{"kind":"Name","value":"stashRoot"}}]}},{"kind":"Field","name":{"kind":"Name","value":"transfer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"action"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"defaultSavePath"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"settingsStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashLibraries"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"path"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stashLibrariesLoadError"}}]}}]}}]} as unknown as DocumentNode<LibraryDeliverySettingsQuery, LibraryDeliverySettingsQueryVariables>;
export const SettingsPolicySectionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SettingsPolicySection"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"policies"},"name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskProcessingEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionPollIntervalHours"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingRetryIntervalHours"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxEndpoints"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionReleasePolicy"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"soloBehavior"}},{"kind":"Field","name":{"kind":"Name","value":"groupBehavior"}},{"kind":"Field","name":{"kind":"Name","value":"compilationBehavior"}},{"kind":"Field","name":{"kind":"Name","value":"vrBehavior"}},{"kind":"Field","name":{"kind":"Name","value":"maxGroupPerformerCount"}},{"kind":"Field","name":{"kind":"Name","value":"releaseDateRange"}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentSelection"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"inspectionCandidateLimit"}},{"kind":"Field","name":{"kind":"Name","value":"fastRules"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"publishDate"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"seeders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"size"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"indexerPreference"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"trackerIds"}}]}},{"kind":"Field","name":{"kind":"Name","value":"titleMatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clauses"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pattern"}},{"kind":"Field","name":{"kind":"Name","value":"patternMode"}},{"kind":"Field","name":{"kind":"Name","value":"effect"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentFileNameMatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clauses"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pattern"}},{"kind":"Field","name":{"kind":"Name","value":"patternMode"}},{"kind":"Field","name":{"kind":"Name","value":"effect"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentRules"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"publishDate"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"seeders"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"size"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"direction"}}]}},{"kind":"Field","name":{"kind":"Name","value":"indexerPreference"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"trackerIds"}}]}},{"kind":"Field","name":{"kind":"Name","value":"titleMatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clauses"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pattern"}},{"kind":"Field","name":{"kind":"Name","value":"patternMode"}},{"kind":"Field","name":{"kind":"Name","value":"effect"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentFileNameMatch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clauses"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"pattern"}},{"kind":"Field","name":{"kind":"Name","value":"patternMode"}},{"kind":"Field","name":{"kind":"Name","value":"effect"}}]}}]}}]}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"settingsStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"taskProcessing"},"name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncEnabled"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionPollIntervalHours"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionPollEnabled"}}]}},{"kind":"Field","alias":{"kind":"Name","value":"metadataSources"},"name":{"kind":"Name","value":"stashBox"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashBoxes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxesLoaded"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxesLoadError"}}]}}]}}]}}]} as unknown as DocumentNode<SettingsPolicySectionQuery, SettingsPolicySectionQueryVariables>;
export const SystemSettingsTabDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SystemSettingsTab"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"system"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"releaseTimezone"}},{"kind":"Field","name":{"kind":"Name","value":"effectiveReleaseTimezone"}},{"kind":"Field","name":{"kind":"Name","value":"developerMode"}},{"kind":"Field","name":{"kind":"Name","value":"taskDeletePolicy"}},{"kind":"Field","name":{"kind":"Name","value":"imageCache"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"maxSizeMb"}},{"kind":"Field","name":{"kind":"Name","value":"retentionDays"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxDataCache"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ttlHours"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"settingsStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"imageCache"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"entryCount"}},{"kind":"Field","name":{"kind":"Name","value":"cacheDirectory"}},{"kind":"Field","name":{"kind":"Name","value":"lastCleanupAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}}]}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxDataCache"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"performerCount"}},{"kind":"Field","name":{"kind":"Name","value":"snapshotCount"}},{"kind":"Field","name":{"kind":"Name","value":"databasePath"}},{"kind":"Field","name":{"kind":"Name","value":"lastCleanupAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}}]}}]}}]}}]} as unknown as DocumentNode<SystemSettingsTabQuery, SystemSettingsTabQueryVariables>;
export const AboutSettingsTabDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"AboutSettingsTab"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"version"}}]}}]} as unknown as DocumentNode<AboutSettingsTabQuery, AboutSettingsTabQueryVariables>;
export const TaskDeletionSettingsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TaskDeletionSettings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"system"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskDeletePolicy"}}]}}]}}]}}]} as unknown as DocumentNode<TaskDeletionSettingsQuery, TaskDeletionSettingsQueryVariables>;
export const TestServiceConnectionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TestServiceConnection"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"TestServiceConnectionInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"testServiceConnection"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"code"}}]}}]}}]} as unknown as DocumentNode<TestServiceConnectionMutation, TestServiceConnectionMutationVariables>;
export const ClearImageCacheDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ClearImageCacheDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clearImageCache"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"entryCount"}},{"kind":"Field","name":{"kind":"Name","value":"cacheDirectory"}},{"kind":"Field","name":{"kind":"Name","value":"lastCleanupAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}}]}}]}}]} as unknown as DocumentNode<ClearImageCacheDocumentMutation, ClearImageCacheDocumentMutationVariables>;
export const ClearStashBoxDataCacheDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ClearStashBoxDataCacheDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"clearStashBoxDataCache"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"usedBytes"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"performerCount"}},{"kind":"Field","name":{"kind":"Name","value":"snapshotCount"}},{"kind":"Field","name":{"kind":"Name","value":"databasePath"}},{"kind":"Field","name":{"kind":"Name","value":"lastCleanupAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}}]}}]}}]} as unknown as DocumentNode<ClearStashBoxDataCacheDocumentMutation, ClearStashBoxDataCacheDocumentMutationVariables>;
export const RefreshStashBoxesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshStashBoxes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshStashBoxes"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sourcingRetryIntervalHours"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxEndpoints"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshStashBoxesMutation, RefreshStashBoxesMutationVariables>;
export const LogsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"LogsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"minLevel"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"LogLevel"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"logs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}},{"kind":"Argument","name":{"kind":"Name","value":"minLevel"},"value":{"kind":"Variable","name":{"kind":"Name","value":"minLevel"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"time"}},{"kind":"Field","name":{"kind":"Name","value":"level"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}}]}}]} as unknown as DocumentNode<LogsDocumentQuery, LogsDocumentQueryVariables>;
export const PatchSettingsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"PatchSettings"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"SettingsPatchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"patchSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"system"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"language"}},{"kind":"Field","name":{"kind":"Name","value":"developerMode"}}]}}]}}]}}]} as unknown as DocumentNode<PatchSettingsMutation, PatchSettingsMutationVariables>;
export const SubscriptionCheckActivityDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SubscriptionCheckActivity"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscriptionCheckActivity"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"activeCount"}},{"kind":"Field","name":{"kind":"Name","value":"allRunId"}},{"kind":"Field","name":{"kind":"Name","value":"performers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"runId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}}]}}]}}]}}]} as unknown as DocumentNode<SubscriptionCheckActivityQuery, SubscriptionCheckActivityQueryVariables>;
export const SubscriptionCheckDetailDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SubscriptionCheckDetail"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscriptionCheck"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"SubscriptionCheckFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"SubscriptionCheckFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"SubscriptionCheckRun"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"trigger"}},{"kind":"Field","name":{"kind":"Name","value":"scope"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"completedCount"}},{"kind":"Field","name":{"kind":"Name","value":"succeededCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}}]}}]}}]} as unknown as DocumentNode<SubscriptionCheckDetailQuery, SubscriptionCheckDetailQueryVariables>;
export const StartSubscriptionCheckDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"StartSubscriptionCheck"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startSubscriptionCheck"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}}]}}]}}]} as unknown as DocumentNode<StartSubscriptionCheckMutation, StartSubscriptionCheckMutationVariables>;
export const StashPerformersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"StashPerformers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"search"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashPerformers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"search"},"value":{"kind":"Variable","name":{"kind":"Name","value":"search"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}},{"kind":"Argument","name":{"kind":"Name","value":"pageSize"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"pageSize"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"hasPrevPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}}]}}]}}]} as unknown as DocumentNode<StashPerformersQuery, StashPerformersQueryVariables>;
export const PerformerWorkspaceDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"PerformerWorkspace"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"search"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"page"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"options"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"PerformerListOptions"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performerWorkspace"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"search"},"value":{"kind":"Variable","name":{"kind":"Name","value":"search"}}},{"kind":"Argument","name":{"kind":"Name","value":"page"},"value":{"kind":"Variable","name":{"kind":"Name","value":"page"}}},{"kind":"Argument","name":{"kind":"Name","value":"pageSize"},"value":{"kind":"Variable","name":{"kind":"Name","value":"pageSize"}}},{"kind":"Argument","name":{"kind":"Name","value":"options"},"value":{"kind":"Variable","name":{"kind":"Name","value":"options"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"pageSize"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"hasPrevPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}}]}},{"kind":"Field","name":{"kind":"Name","value":"subscribedPerformers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"reviewReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<PerformerWorkspaceQuery, PerformerWorkspaceQueryVariables>;
export const StashPerformerProfileDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"StashPerformerProfile"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashPerformerProfile"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"disambiguation"}},{"kind":"Field","name":{"kind":"Name","value":"birthdate"}},{"kind":"Field","name":{"kind":"Name","value":"ethnicity"}},{"kind":"Field","name":{"kind":"Name","value":"country"}},{"kind":"Field","name":{"kind":"Name","value":"eyeColor"}},{"kind":"Field","name":{"kind":"Name","value":"heightCm"}},{"kind":"Field","name":{"kind":"Name","value":"rating100"}},{"kind":"Field","name":{"kind":"Name","value":"urls"}}]}}]}}]} as unknown as DocumentNode<StashPerformerProfileQuery, StashPerformerProfileQueryVariables>;
export const StashPerformerDetailDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"StashPerformerDetail"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashPerformerDetail"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"disambiguation"}},{"kind":"Field","name":{"kind":"Name","value":"birthdate"}},{"kind":"Field","name":{"kind":"Name","value":"ethnicity"}},{"kind":"Field","name":{"kind":"Name","value":"country"}},{"kind":"Field","name":{"kind":"Name","value":"eyeColor"}},{"kind":"Field","name":{"kind":"Name","value":"heightCm"}},{"kind":"Field","name":{"kind":"Name","value":"rating100"}},{"kind":"Field","name":{"kind":"Name","value":"urls"}},{"kind":"Field","name":{"kind":"Name","value":"matchedStashBox"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"performerName"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalSceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxSceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"dedupedSceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"activeTaskCount"}},{"kind":"Field","name":{"kind":"Name","value":"sceneStatsComplete"}},{"kind":"Field","name":{"kind":"Name","value":"sceneStatsStale"}}]}}]}}]} as unknown as DocumentNode<StashPerformerDetailQuery, StashPerformerDetailQueryVariables>;
export const StashPerformerScenesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"StashPerformerScenes"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"StashPerformerScenesInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stashPerformerScenes"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"primarySource"}},{"kind":"Field","name":{"kind":"Name","value":"sourceSceneId"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"studioName"}},{"kind":"Field","name":{"kind":"Name","value":"performerCount"}},{"kind":"Field","name":{"kind":"Name","value":"tagCount"}},{"kind":"Field","name":{"kind":"Name","value":"performers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"tags"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"imageUrl"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"inLibrary"}},{"kind":"Field","name":{"kind":"Name","value":"matchedStashSceneId"}},{"kind":"Field","name":{"kind":"Name","value":"hasStashSource"}},{"kind":"Field","name":{"kind":"Name","value":"hasStashBoxSource"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxSceneId"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxEndpoint"}},{"kind":"Field","name":{"kind":"Name","value":"sourceLabels"}},{"kind":"Field","name":{"kind":"Name","value":"stashIds"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"stashId"}}]}},{"kind":"Field","name":{"kind":"Name","value":"upcomingState"}},{"kind":"Field","name":{"kind":"Name","value":"mojiTask"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"pageSize"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"hasPrevPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxCount"}},{"kind":"Field","name":{"kind":"Name","value":"dedupedCount"}},{"kind":"Field","name":{"kind":"Name","value":"totalCountExact"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxRemoteCount"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxLoadedCount"}},{"kind":"Field","name":{"kind":"Name","value":"cacheComplete"}},{"kind":"Field","name":{"kind":"Name","value":"cacheUpdatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"cacheStale"}}]}}]}}]} as unknown as DocumentNode<StashPerformerScenesQuery, StashPerformerScenesQueryVariables>;
export const RefreshStashPerformerScenesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshStashPerformerScenes"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"StashPerformerScenesInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshStashPerformerScenes"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}}]}},{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"pageSize"}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"hasPrevPage"}},{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxCount"}},{"kind":"Field","name":{"kind":"Name","value":"dedupedCount"}},{"kind":"Field","name":{"kind":"Name","value":"totalCountExact"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxRemoteCount"}},{"kind":"Field","name":{"kind":"Name","value":"stashBoxLoadedCount"}},{"kind":"Field","name":{"kind":"Name","value":"cacheComplete"}},{"kind":"Field","name":{"kind":"Name","value":"cacheUpdatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"cacheStale"}}]}}]}}]} as unknown as DocumentNode<RefreshStashPerformerScenesMutation, RefreshStashPerformerScenesMutationVariables>;
export const SubscribedPerformersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SubscribedPerformers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscribedPerformers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"reviewReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]} as unknown as DocumentNode<SubscribedPerformersQuery, SubscribedPerformersQueryVariables>;
export const SubscribePerformerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SubscribePerformer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscribePerformer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"stashPerformerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"reviewReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]} as unknown as DocumentNode<SubscribePerformerMutation, SubscribePerformerMutationVariables>;
export const UnsubscribePerformerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnsubscribePerformer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unsubscribePerformer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"stashPerformerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}}}]}]}}]} as unknown as DocumentNode<UnsubscribePerformerMutation, UnsubscribePerformerMutationVariables>;
export const RefreshSubscribedPerformerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshSubscribedPerformer"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshSubscribedPerformer"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"stashPerformerID"},"value":{"kind":"Variable","name":{"kind":"Name","value":"stashPerformerID"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"reviewReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshSubscribedPerformerMutation, RefreshSubscribedPerformerMutationVariables>;
export const RefreshSubscriptionsNowDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshSubscriptionsNow"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshSubscriptionsNow"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"reviewReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshSubscriptionsNowMutation, RefreshSubscriptionsNowMutationVariables>;
export const SubscribePerformersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SubscribePerformers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscribePerformers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"PerformerBatchFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"PerformerBatchFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"PerformerBatchPayload"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"batchId"}},{"kind":"Field","name":{"kind":"Name","value":"summary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"requestedCount"}},{"kind":"Field","name":{"kind":"Name","value":"succeededCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}}]}},{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reasonCode"}},{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"state"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"reviewReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<SubscribePerformersMutation, SubscribePerformersMutationVariables>;
export const UnsubscribePerformersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UnsubscribePerformers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"unsubscribePerformers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"PerformerBatchFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"PerformerBatchFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"PerformerBatchPayload"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"batchId"}},{"kind":"Field","name":{"kind":"Name","value":"summary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"requestedCount"}},{"kind":"Field","name":{"kind":"Name","value":"succeededCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}}]}},{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reasonCode"}},{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"state"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"reviewReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<UnsubscribePerformersMutation, UnsubscribePerformersMutationVariables>;
export const RefreshSubscribedPerformersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshSubscribedPerformers"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshSubscribedPerformers"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"PerformerBatchFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"PerformerBatchFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"PerformerBatchPayload"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"batchId"}},{"kind":"Field","name":{"kind":"Name","value":"summary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"requestedCount"}},{"kind":"Field","name":{"kind":"Name","value":"succeededCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}}]}},{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performerId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reasonCode"}},{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"state"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"performer"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"aliasList"}},{"kind":"Field","name":{"kind":"Name","value":"favorite"}},{"kind":"Field","name":{"kind":"Name","value":"imagePath"}},{"kind":"Field","name":{"kind":"Name","value":"sceneCount"}},{"kind":"Field","name":{"kind":"Name","value":"subscribed"}}]}},{"kind":"Field","name":{"kind":"Name","value":"lastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastError"}},{"kind":"Field","name":{"kind":"Name","value":"reviewReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"pendingReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"processedReleaseCount"}},{"kind":"Field","name":{"kind":"Name","value":"recentReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"seenAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<RefreshSubscribedPerformersMutation, RefreshSubscribedPerformersMutationVariables>;
export const QueuePerformerScenesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"QueuePerformerScenes"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"QueuePerformerScenesInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"queuePerformerScenes"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"queuedTasks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reasonCode"}},{"kind":"Field","name":{"kind":"Name","value":"resolvedCode"}},{"kind":"Field","name":{"kind":"Name","value":"task"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"summary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"requestedCount"}},{"kind":"Field","name":{"kind":"Name","value":"queuedCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}}]}}]}}]}}]} as unknown as DocumentNode<QueuePerformerScenesMutation, QueuePerformerScenesMutationVariables>;
export const TaskEventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"subscription","name":{"kind":"Name","value":"TaskEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskEvents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"streamId"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"taskId"}},{"kind":"Field","name":{"kind":"Name","value":"listChanged"}},{"kind":"Field","alias":{"kind":"Name","value":"task"},"name":{"kind":"Name","value":"summary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TaskListSummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"dashboardStats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"active"}},{"kind":"Field","name":{"kind":"Name","value":"completed"}},{"kind":"Field","name":{"kind":"Name","value":"downloading"}},{"kind":"Field","name":{"kind":"Name","value":"pendingScans"}},{"kind":"Field","name":{"kind":"Name","value":"failed"}},{"kind":"Field","name":{"kind":"Name","value":"attention"}},{"kind":"Field","name":{"kind":"Name","value":"waiting"}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TaskListSummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"TaskSummary"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"stashIngestPhase"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneId"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"downloadCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]} as unknown as DocumentNode<TaskEventsSubscription, TaskEventsSubscriptionVariables>;
export const TasksWorkspaceDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TasksWorkspace"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"TaskPageInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskPage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TaskListSummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"pageSize"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"streamId"}},{"kind":"Field","name":{"kind":"Name","value":"stats"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"active"}},{"kind":"Field","name":{"kind":"Name","value":"completed"}},{"kind":"Field","name":{"kind":"Name","value":"downloading"}},{"kind":"Field","name":{"kind":"Name","value":"pendingScans"}},{"kind":"Field","name":{"kind":"Name","value":"failed"}},{"kind":"Field","name":{"kind":"Name","value":"attention"}},{"kind":"Field","name":{"kind":"Name","value":"waiting"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"system"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskDeletePolicy"}}]}},{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskProcessingEnabled"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"settingsStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"automation"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"taskProgressSyncEnabled"}}]}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TaskListSummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"TaskSummary"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"stashIngestPhase"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneId"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"downloadCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]} as unknown as DocumentNode<TasksWorkspaceQuery, TasksWorkspaceQueryVariables>;
export const TaskSelectionSummariesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TaskSelectionSummaries"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","alias":{"kind":"Name","value":"taskSelection"},"name":{"kind":"Name","value":"taskPage"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"ObjectValue","fields":[{"kind":"ObjectField","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}},{"kind":"ObjectField","name":{"kind":"Name","value":"pageSize"},"value":{"kind":"IntValue","value":"100"}}]}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TaskListSummary"}}]}},{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"streamId"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TaskListSummary"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"TaskSummary"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"stashIngestPhase"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneId"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"downloadCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]} as unknown as DocumentNode<TaskSelectionSummariesQuery, TaskSelectionSummariesQueryVariables>;
export const TaskAttentionQueueDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TaskAttentionQueue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskAttentionQueue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"streamId"}}]}}]}}]} as unknown as DocumentNode<TaskAttentionQueueQuery, TaskAttentionQueueQueryVariables>;
export const TaskDetailDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TaskDetailDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskDetailSnapshot"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"streamId"}},{"kind":"Field","name":{"kind":"Name","value":"task"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"files"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"index"}},{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"part"}},{"kind":"Field","name":{"kind":"Name","value":"selected"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}}]}},{"kind":"Field","name":{"kind":"Name","value":"candidate"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"tracker"}},{"kind":"Field","name":{"kind":"Name","value":"infoHash"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"magnetUri"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentUrl"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"contentPath"}},{"kind":"Field","name":{"kind":"Name","value":"torrentHash"}},{"kind":"Field","name":{"kind":"Name","value":"savePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"downloadCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"mojiSourcePath"}},{"kind":"Field","name":{"kind":"Name","value":"transferAction"}},{"kind":"Field","name":{"kind":"Name","value":"mojiTransferPath"}},{"kind":"Field","name":{"kind":"Name","value":"transferError"}},{"kind":"Field","name":{"kind":"Name","value":"stashIngestPhase"}},{"kind":"Field","name":{"kind":"Name","value":"stashGenerateJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashIdentifyJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneIds"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanHint"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStartedAt"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}}]}}]}}]}}]} as unknown as DocumentNode<TaskDetailDocumentQuery, TaskDetailDocumentQueryVariables>;
export const AddTorrentDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddTorrentDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"QBittorrentAddInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"addTorrent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"files"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"index"}},{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"part"}},{"kind":"Field","name":{"kind":"Name","value":"selected"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}}]}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<AddTorrentDocumentMutation, AddTorrentDocumentMutationVariables>;
export const SyncTaskProgressDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SyncTaskProgressDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"syncTaskProgress"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}}]}}]}}]} as unknown as DocumentNode<SyncTaskProgressDocumentMutation, SyncTaskProgressDocumentMutationVariables>;
export const TriggerStashScansDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TriggerStashScansDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggerStashScans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"transferError"}},{"kind":"Field","name":{"kind":"Name","value":"stashIngestPhase"}},{"kind":"Field","name":{"kind":"Name","value":"stashGenerateJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashIdentifyJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneIds"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanHint"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}}]}}]}}]} as unknown as DocumentNode<TriggerStashScansDocumentMutation, TriggerStashScansDocumentMutationVariables>;
export const TriggerTaskStashScanDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TriggerTaskStashScanDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggerTaskStashScan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"transferError"}},{"kind":"Field","name":{"kind":"Name","value":"stashIngestPhase"}},{"kind":"Field","name":{"kind":"Name","value":"stashGenerateJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashIdentifyJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneIds"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanHint"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}}]}}]}}]} as unknown as DocumentNode<TriggerTaskStashScanDocumentMutation, TriggerTaskStashScanDocumentMutationVariables>;
export const DeleteTaskDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteTaskDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteTask"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}}]}}]}}]} as unknown as DocumentNode<DeleteTaskDocumentMutation, DeleteTaskDocumentMutationVariables>;
export const RetryTaskDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RetryTaskDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"retryTask"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"files"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"index"}},{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"part"}},{"kind":"Field","name":{"kind":"Name","value":"selected"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}}]}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}}]}}]}}]} as unknown as DocumentNode<RetryTaskDocumentMutation, RetryTaskDocumentMutationVariables>;
export const RetryTasksDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RetryTasksDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"retryTasks"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"batchId"}},{"kind":"Field","name":{"kind":"Name","value":"summary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"requestedCount"}},{"kind":"Field","name":{"kind":"Name","value":"succeededCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}}]}},{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reasonCode"}},{"kind":"Field","name":{"kind":"Name","value":"task"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<RetryTasksDocumentMutation, RetryTasksDocumentMutationVariables>;
export const ProcessTaskIngestDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ProcessTaskIngestDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"processTaskIngest"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"batchId"}},{"kind":"Field","name":{"kind":"Name","value":"summary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"requestedCount"}},{"kind":"Field","name":{"kind":"Name","value":"succeededCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}}]}},{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reasonCode"}},{"kind":"Field","name":{"kind":"Name","value":"task"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"transferError"}},{"kind":"Field","name":{"kind":"Name","value":"stashIngestPhase"}},{"kind":"Field","name":{"kind":"Name","value":"stashGenerateJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashIdentifyJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneIds"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanHint"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<ProcessTaskIngestDocumentMutation, ProcessTaskIngestDocumentMutationVariables>;
export const DeleteTasksDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteTasksDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"ids"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteTasks"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"ids"},"value":{"kind":"Variable","name":{"kind":"Name","value":"ids"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"batchId"}},{"kind":"Field","name":{"kind":"Name","value":"summary"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"requestedCount"}},{"kind":"Field","name":{"kind":"Name","value":"succeededCount"}},{"kind":"Field","name":{"kind":"Name","value":"skippedCount"}},{"kind":"Field","name":{"kind":"Name","value":"failedCount"}}]}},{"kind":"Field","name":{"kind":"Name","value":"results"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskId"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"reasonCode"}},{"kind":"Field","name":{"kind":"Name","value":"task"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}}]}}]}}]}}]}}]} as unknown as DocumentNode<DeleteTasksDocumentMutation, DeleteTasksDocumentMutationVariables>;
export const ResolveBlockedSourcingTaskDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ResolveBlockedSourcingTaskDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ResolveBlockedSourcingTaskInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"resolveBlockedSourcingTask"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"files"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"index"}},{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"part"}},{"kind":"Field","name":{"kind":"Name","value":"selected"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}}]}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"candidate"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"tracker"}},{"kind":"Field","name":{"kind":"Name","value":"infoHash"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentUrl"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}}]}}]}}]} as unknown as DocumentNode<ResolveBlockedSourcingTaskDocumentMutation, ResolveBlockedSourcingTaskDocumentMutationVariables>;
export const ResolveTaskFilesDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ResolveTaskFilesDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"videoIndices"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"resolveTaskFiles"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"videoIndices"},"value":{"kind":"Variable","name":{"kind":"Name","value":"videoIndices"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"files"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"index"}},{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"part"}},{"kind":"Field","name":{"kind":"Name","value":"selected"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}}]}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}}]}}]}}]} as unknown as DocumentNode<ResolveTaskFilesDocumentMutation, ResolveTaskFilesDocumentMutationVariables>;
export const TaskStashScenesDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TaskStashScenesDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"taskStashScenes"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"studio"}},{"kind":"Field","name":{"kind":"Name","value":"performers"}},{"kind":"Field","name":{"kind":"Name","value":"imageUrl"}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}}]}}]} as unknown as DocumentNode<TaskStashScenesDocumentQuery, TaskStashScenesDocumentQueryVariables>;
export const RestartTaskSourcingDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RestartTaskSourcingDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"restartTaskSourcing"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"source"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stageLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatusLabel"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"files"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"index"}},{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"part"}},{"kind":"Field","name":{"kind":"Name","value":"selected"}},{"kind":"Field","name":{"kind":"Name","value":"reason"}}]}},{"kind":"Field","name":{"kind":"Name","value":"candidate"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"tracker"}},{"kind":"Field","name":{"kind":"Name","value":"infoHash"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"magnetUri"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentUrl"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"contentPath"}},{"kind":"Field","name":{"kind":"Name","value":"torrentHash"}},{"kind":"Field","name":{"kind":"Name","value":"savePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"downloadCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"ingestCompletedAt"}},{"kind":"Field","name":{"kind":"Name","value":"mojiSourcePath"}},{"kind":"Field","name":{"kind":"Name","value":"transferAction"}},{"kind":"Field","name":{"kind":"Name","value":"mojiTransferPath"}},{"kind":"Field","name":{"kind":"Name","value":"transferError"}},{"kind":"Field","name":{"kind":"Name","value":"stashIngestPhase"}},{"kind":"Field","name":{"kind":"Name","value":"stashGenerateJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashIdentifyJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashSceneIds"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanPath"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanHint"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStartedAt"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"lastProgressAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingLastCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}}]}}]}}]} as unknown as DocumentNode<RestartTaskSourcingDocumentMutation, RestartTaskSourcingDocumentMutationVariables>;
export const TorrentSearchDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TorrentSearch"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"offset"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"limit"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"sortBy"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"TorrentSearchSort"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"torrentSearch"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"offset"},"value":{"kind":"Variable","name":{"kind":"Name","value":"offset"}}},{"kind":"Argument","name":{"kind":"Name","value":"limit"},"value":{"kind":"Variable","name":{"kind":"Name","value":"limit"}}},{"kind":"Argument","name":{"kind":"Name","value":"sortBy"},"value":{"kind":"Variable","name":{"kind":"Name","value":"sortBy"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"FragmentSpread","name":{"kind":"Name","value":"TorrentSearchFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TorrentSearchFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"TorrentSearch"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"keywords"}},{"kind":"Field","name":{"kind":"Name","value":"taskId"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"expiresAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"cancelRequested"}},{"kind":"Field","name":{"kind":"Name","value":"partial"}},{"kind":"Field","name":{"kind":"Name","value":"truncated"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"sites"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"results"}},{"kind":"Field","name":{"kind":"Name","value":"cacheHit"}},{"kind":"Field","name":{"kind":"Name","value":"fetchedAt"}},{"kind":"Field","name":{"kind":"Name","value":"elapsedMs"}},{"kind":"Field","name":{"kind":"Name","value":"partial"}},{"kind":"Field","name":{"kind":"Name","value":"truncated"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"candidates"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"instanceName"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"detailsUrl"}},{"kind":"Field","name":{"kind":"Name","value":"infoHash"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}},{"kind":"Field","name":{"kind":"Name","value":"publishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"categories"}}]}}]}}]} as unknown as DocumentNode<TorrentSearchQuery, TorrentSearchQueryVariables>;
export const StartTorrentSearchDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"StartTorrentSearch"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"StartTorrentSearchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startTorrentSearch"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"TorrentSearchFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TorrentSearchFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"TorrentSearch"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"keywords"}},{"kind":"Field","name":{"kind":"Name","value":"taskId"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"expiresAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"cancelRequested"}},{"kind":"Field","name":{"kind":"Name","value":"partial"}},{"kind":"Field","name":{"kind":"Name","value":"truncated"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"sites"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"results"}},{"kind":"Field","name":{"kind":"Name","value":"cacheHit"}},{"kind":"Field","name":{"kind":"Name","value":"fetchedAt"}},{"kind":"Field","name":{"kind":"Name","value":"elapsedMs"}},{"kind":"Field","name":{"kind":"Name","value":"partial"}},{"kind":"Field","name":{"kind":"Name","value":"truncated"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"candidates"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"instanceName"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"detailsUrl"}},{"kind":"Field","name":{"kind":"Name","value":"infoHash"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}},{"kind":"Field","name":{"kind":"Name","value":"publishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"categories"}}]}}]}}]} as unknown as DocumentNode<StartTorrentSearchMutation, StartTorrentSearchMutationVariables>;
export const CancelTorrentSearchDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CancelTorrentSearch"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"cancelTorrentSearch"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"FragmentSpread","name":{"kind":"Name","value":"TorrentSearchFields"}}]}}]}},{"kind":"FragmentDefinition","name":{"kind":"Name","value":"TorrentSearchFields"},"typeCondition":{"kind":"NamedType","name":{"kind":"Name","value":"TorrentSearch"}},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"keywords"}},{"kind":"Field","name":{"kind":"Name","value":"taskId"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}},{"kind":"Field","name":{"kind":"Name","value":"expiresAt"}},{"kind":"Field","name":{"kind":"Name","value":"finishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"cancelRequested"}},{"kind":"Field","name":{"kind":"Name","value":"partial"}},{"kind":"Field","name":{"kind":"Name","value":"truncated"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"sites"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"results"}},{"kind":"Field","name":{"kind":"Name","value":"cacheHit"}},{"kind":"Field","name":{"kind":"Name","value":"fetchedAt"}},{"kind":"Field","name":{"kind":"Name","value":"elapsedMs"}},{"kind":"Field","name":{"kind":"Name","value":"partial"}},{"kind":"Field","name":{"kind":"Name","value":"truncated"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"correlationId"}},{"kind":"Field","name":{"kind":"Name","value":"retryAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"candidates"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"instanceName"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"detailsUrl"}},{"kind":"Field","name":{"kind":"Name","value":"infoHash"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}},{"kind":"Field","name":{"kind":"Name","value":"publishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"categories"}}]}}]}}]} as unknown as DocumentNode<CancelTorrentSearchMutation, CancelTorrentSearchMutationVariables>;
export const QueueTorrentCandidateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"QueueTorrentCandidate"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"QueueTorrentCandidateInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"queueTorrentCandidate"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}}]}}]}}]} as unknown as DocumentNode<QueueTorrentCandidateMutation, QueueTorrentCandidateMutationVariables>;
export const TorrentSearchEventsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"subscription","name":{"kind":"Name","value":"TorrentSearchEvents"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"torrentSearchEvents"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"streamId"}},{"kind":"Field","name":{"kind":"Name","value":"sequence"}},{"kind":"Field","name":{"kind":"Name","value":"searchId"}}]}}]}}]} as unknown as DocumentNode<TorrentSearchEventsSubscription, TorrentSearchEventsSubscriptionVariables>;
export const PreviewTorrentSelectionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"PreviewTorrentSelection"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"PreviewTorrentSelectionInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"previewTorrentSelection"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"searchId"}},{"kind":"Field","name":{"kind":"Name","value":"searchRevision"}},{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"offset"}},{"kind":"Field","name":{"kind":"Name","value":"limit"}},{"kind":"Field","name":{"kind":"Name","value":"candidates"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"instanceId"}},{"kind":"Field","name":{"kind":"Name","value":"instanceName"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"detailsUrl"}},{"kind":"Field","name":{"kind":"Name","value":"infoHash"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}},{"kind":"Field","name":{"kind":"Name","value":"publishedAt"}},{"kind":"Field","name":{"kind":"Name","value":"categories"}}]}},{"kind":"Field","name":{"kind":"Name","value":"meta"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"appliedFastRules"}},{"kind":"Field","name":{"kind":"Name","value":"appliedFileRules"}},{"kind":"Field","name":{"kind":"Name","value":"inspectedCount"}},{"kind":"Field","name":{"kind":"Name","value":"inspectableCount"}},{"kind":"Field","name":{"kind":"Name","value":"inspectionCandidateLimit"}}]}}]}}]}}]} as unknown as DocumentNode<PreviewTorrentSelectionMutation, PreviewTorrentSelectionMutationVariables>;
export const TorrentSearchSourcesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TorrentSearchSources"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"indexerInstances"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"ruleErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"catalogErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}}]}}]}}]}}]} as unknown as DocumentNode<TorrentSearchSourcesQuery, TorrentSearchSourcesQueryVariables>;
export const ResolveTaskSourcingCandidateDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ResolveTaskSourcingCandidate"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ResolveTaskSourcingCandidateInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"resolveTaskSourcingCandidate"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"stage"}},{"kind":"Field","name":{"kind":"Name","value":"stageStatus"}},{"kind":"Field","name":{"kind":"Name","value":"revision"}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"sourcingNextCheckAt"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorCode"}},{"kind":"Field","name":{"kind":"Name","value":"stageErrorMessage"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<ResolveTaskSourcingCandidateMutation, ResolveTaskSourcingCandidateMutationVariables>;
export const PrepareTorrentDownloadDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"PrepareTorrentDownload"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"PrepareTorrentDownloadInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"prepareTorrentDownload"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"kind"}},{"kind":"Field","name":{"kind":"Name","value":"downloadUrl"}},{"kind":"Field","name":{"kind":"Name","value":"expiresAt"}}]}}]}}]} as unknown as DocumentNode<PrepareTorrentDownloadMutation, PrepareTorrentDownloadMutationVariables>;
export const UpcomingReleasesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"UpcomingReleases"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpcomingReleaseQueryInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"upcomingReleases"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"page"}},{"kind":"Field","name":{"kind":"Name","value":"pageSize"}},{"kind":"Field","name":{"kind":"Name","value":"totalPages"}},{"kind":"Field","name":{"kind":"Name","value":"subscribedCount"}},{"kind":"Field","name":{"kind":"Name","value":"automatic"}},{"kind":"Field","name":{"kind":"Name","value":"timezone"}},{"kind":"Field","name":{"kind":"Name","value":"today"}},{"kind":"Field","name":{"kind":"Name","value":"performers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"imageUrl"}},{"kind":"Field","name":{"kind":"Name","value":"sourceUrl"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"decision"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"metadataCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"performers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]}}]} as unknown as DocumentNode<UpcomingReleasesQuery, UpcomingReleasesQueryVariables>;
export const RefreshUpcomingReleaseDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"RefreshUpcomingRelease"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"key"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ID"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"refreshUpcomingRelease"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"key"},"value":{"kind":"Variable","name":{"kind":"Name","value":"key"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"code"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"date"}},{"kind":"Field","name":{"kind":"Name","value":"imageUrl"}},{"kind":"Field","name":{"kind":"Name","value":"sourceUrl"}},{"kind":"Field","name":{"kind":"Name","value":"state"}},{"kind":"Field","name":{"kind":"Name","value":"decision"}},{"kind":"Field","name":{"kind":"Name","value":"errorCode"}},{"kind":"Field","name":{"kind":"Name","value":"metadataCheckedAt"}},{"kind":"Field","name":{"kind":"Name","value":"performers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}}]}}]}}]}}]} as unknown as DocumentNode<RefreshUpcomingReleaseMutation, RefreshUpcomingReleaseMutationVariables>;