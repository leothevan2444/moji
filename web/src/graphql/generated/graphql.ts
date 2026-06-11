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
  /** Start a Stash metadata scan */
  stashMetadataScan: Scalars['ID']['output'];
  /** Synchronize Moji task progress from qBittorrent */
  syncTaskProgress: Array<Task>;
  /** Trigger Stash metadata scans for completed Moji tasks */
  triggerStashScans: Array<Task>;
  /** Update Jackett settings and persist them to backend config */
  updateJackettSettings: Settings;
  /** Update qBittorrent settings and persist them to backend config */
  updateQBittorrentSettings: Settings;
  /** Update Stash settings and persist them to backend config */
  updateStashSettings: Settings;
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


export type MutationStashMetadataScanArgs = {
  input: StashMetadataScanInput;
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
  health: Health;
  /** Search torrents via Jackett */
  jackettSearch: Array<JackettSearchResult>;
  /** List torrents from qBittorrent */
  qbittorrentTorrents: Array<QbTorrent>;
  /** Get runtime capability and configuration state for the Settings surface */
  settings: Settings;
  /** Get a Stash background job by id */
  stashJob?: Maybe<StashJob>;
  /** Get a Moji download task by id */
  task?: Maybe<Task>;
  /** List Moji download tasks, newest first */
  tasks: Array<Task>;
  version: Scalars['String']['output'];
};


export type QueryJackettSearchArgs = {
  input: JackettSearchInput;
};


export type QueryQbittorrentTorrentsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
};


export type QueryStashJobArgs = {
  id: Scalars['ID']['input'];
};


export type QueryTaskArgs = {
  id: Scalars['ID']['input'];
};

export type Settings = {
  __typename?: 'Settings';
  jackett: JackettSettings;
  qbittorrent: QBittorrentSettings;
  stash: StashSettings;
  system: SystemSettings;
  tasks: TaskSettings;
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

export type StashSettings = {
  __typename?: 'StashSettings';
  apiKeyConfigured: Scalars['Boolean']['output'];
  configured: Scalars['Boolean']['output'];
  enabled: Scalars['Boolean']['output'];
  libraryPath: Scalars['String']['output'];
  url: Scalars['String']['output'];
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
  jsonPath: Scalars['String']['output'];
  progressSyncEnabled: Scalars['Boolean']['output'];
  progressSyncIntervalSeconds: Scalars['Int']['output'];
  store: Scalars['String']['output'];
};

export type UpdateJackettSettingsInput = {
  apiKey?: InputMaybe<Scalars['String']['input']>;
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
  url: Scalars['String']['input'];
};

export type DashboardDocumentQueryVariables = Exact<{ [key: string]: never; }>;


export type DashboardDocumentQuery = { __typename?: 'Query', version: string, health: { __typename?: 'Health', ok: boolean, message: string }, settings: { __typename?: 'Settings', stash: { __typename?: 'StashSettings', configured: boolean, enabled: boolean, url: string, apiKeyConfigured: boolean, libraryPath: string }, jackett: { __typename?: 'JackettSettings', configured: boolean, enabled: boolean, url: string, apiKeyConfigured: boolean }, qbittorrent: { __typename?: 'QBittorrentSettings', configured: boolean, enabled: boolean, url: string, username: string, usernameConfigured: boolean, passwordConfigured: boolean, defaultSavePath: string, category: string, tags: string }, tasks: { __typename?: 'TaskSettings', store: string, jsonPath: string, progressSyncIntervalSeconds: number, progressSyncEnabled: boolean }, system: { __typename?: 'SystemSettings', appVersion: string } }, tasks: Array<{ __typename?: 'Task', id: string, query: string, status: string, torrentName: string, progress: number, qbittorrentState: string, contentPath: string, torrentHash: string, savePath: string, category: string, tags: string, error: string, completedAt?: string | null, stashJobId: string, stashScanStatus: string, stashScanError: string, createdAt: string, updatedAt: string }> };

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

export type AddTorrentDocumentMutationVariables = Exact<{
  input: QBittorrentAddInput;
}>;


export type AddTorrentDocumentMutation = { __typename?: 'Mutation', addTorrent: { __typename?: 'Task', id: string, status: string, query: string, torrentName: string, progress: number, stashScanStatus: string, createdAt: string } };

export type SyncTaskProgressDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type SyncTaskProgressDocumentMutation = { __typename?: 'Mutation', syncTaskProgress: Array<{ __typename?: 'Task', id: string, status: string, progress: number, qbittorrentState: string, updatedAt: string }> };

export type TriggerStashScansDocumentMutationVariables = Exact<{ [key: string]: never; }>;


export type TriggerStashScansDocumentMutation = { __typename?: 'Mutation', triggerStashScans: Array<{ __typename?: 'Task', id: string, stashJobId: string, stashScanStatus: string, stashScanError: string, updatedAt: string }> };


export const DashboardDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DashboardDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ok"}},{"kind":"Field","name":{"kind":"Name","value":"message"}}]}},{"kind":"Field","name":{"kind":"Name","value":"version"}},{"kind":"Field","name":{"kind":"Name","value":"settings"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"libraryPath"}}]}},{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"username"}},{"kind":"Field","name":{"kind":"Name","value":"usernameConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"passwordConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"defaultSavePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}}]}},{"kind":"Field","name":{"kind":"Name","value":"tasks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"store"}},{"kind":"Field","name":{"kind":"Name","value":"jsonPath"}},{"kind":"Field","name":{"kind":"Name","value":"progressSyncIntervalSeconds"}},{"kind":"Field","name":{"kind":"Name","value":"progressSyncEnabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"system"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"appVersion"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"tasks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"contentPath"}},{"kind":"Field","name":{"kind":"Name","value":"torrentHash"}},{"kind":"Field","name":{"kind":"Name","value":"savePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}},{"kind":"Field","name":{"kind":"Name","value":"error"}},{"kind":"Field","name":{"kind":"Name","value":"completedAt"}},{"kind":"Field","name":{"kind":"Name","value":"stashJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<DashboardDocumentQuery, DashboardDocumentQueryVariables>;
export const SearchDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"SearchDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"JackettSearchInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jackettSearch"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"peers"}},{"kind":"Field","name":{"kind":"Name","value":"tracker"}},{"kind":"Field","name":{"kind":"Name","value":"categoryDesc"}},{"kind":"Field","name":{"kind":"Name","value":"publishDate"}},{"kind":"Field","name":{"kind":"Name","value":"link"}},{"kind":"Field","name":{"kind":"Name","value":"magnetUri"}}]}}]}}]} as unknown as DocumentNode<SearchDocumentQuery, SearchDocumentQueryVariables>;
export const UpdateStashSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateStashSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateStashSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateStashSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"stash"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"libraryPath"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateStashSettingsDocumentMutation, UpdateStashSettingsDocumentMutationVariables>;
export const UpdateJackettSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateJackettSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateJackettSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateJackettSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jackett"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"apiKeyConfigured"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateJackettSettingsDocumentMutation, UpdateJackettSettingsDocumentMutationVariables>;
export const UpdateQBittorrentSettingsDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateQBittorrentSettingsDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"UpdateQBittorrentSettingsInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateQBittorrentSettings"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"qbittorrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"configured"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"username"}},{"kind":"Field","name":{"kind":"Name","value":"usernameConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"passwordConfigured"}},{"kind":"Field","name":{"kind":"Name","value":"defaultSavePath"}},{"kind":"Field","name":{"kind":"Name","value":"category"}},{"kind":"Field","name":{"kind":"Name","value":"tags"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateQBittorrentSettingsDocumentMutation, UpdateQBittorrentSettingsDocumentMutationVariables>;
export const AddTorrentDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"AddTorrentDocument"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"QBittorrentAddInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"addTorrent"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"query"}},{"kind":"Field","name":{"kind":"Name","value":"torrentName"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}}]}}]} as unknown as DocumentNode<AddTorrentDocumentMutation, AddTorrentDocumentMutationVariables>;
export const SyncTaskProgressDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"SyncTaskProgressDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"syncTaskProgress"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"progress"}},{"kind":"Field","name":{"kind":"Name","value":"qbittorrentState"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<SyncTaskProgressDocumentMutation, SyncTaskProgressDocumentMutationVariables>;
export const TriggerStashScansDocumentDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TriggerStashScansDocument"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"triggerStashScans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"stashJobId"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanStatus"}},{"kind":"Field","name":{"kind":"Name","value":"stashScanError"}},{"kind":"Field","name":{"kind":"Name","value":"updatedAt"}}]}}]}}]} as unknown as DocumentNode<TriggerStashScansDocumentMutation, TriggerStashScansDocumentMutationVariables>;