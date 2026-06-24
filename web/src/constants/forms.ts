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

export const EMPTY_SUBSCRIPTION_FORM = {
  stashBoxEndpoints: [] as string[]
};

export const EMPTY_AUTOMATION_FORM = {
  taskProgressSyncIntervalSeconds: "60",
  subscriptionPollIntervalHours: "1"
};

export const TOAST_LIFETIME_MS = 10000;
export const TOAST_EXIT_MS = 480;

export const SUBSCRIPTION_PAGE_SIZE_OPTIONS = [12, 24, 48, 96] as const;

import { LogLevel } from "../graphql/generated/graphql";
export const LOG_LEVEL_OPTIONS: LogLevel[] = [LogLevel.Debug, LogLevel.Info, LogLevel.Warning, LogLevel.Error];
