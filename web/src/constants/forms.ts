import {
  TaskProcessingFrequency,
  LogLevel,
  SubscriptionReleaseBehavior,
  SubscriptionReleaseDateRange,
  type ResourceRuleInput,
 type ResourceRuleGroupInput,
} from "../graphql/generated/graphql";

export const DEFAULT_RESOURCE_SELECTION = {
 version:7, activeGroup:"", rules:[] as ResourceRuleInput[], groups:[] as ResourceRuleGroupInput[]
};

export const EMPTY_POLICY_SETTINGS_FORM = {
  sourcingRetryIntervalHours: "6",
  taskProcessingFrequency: TaskProcessingFrequency.Medium,

  stashBoxEndpoints: [] as string[],
  subscriptionReleasePolicy: {
    soloBehavior: SubscriptionReleaseBehavior.Download,
    groupBehavior: SubscriptionReleaseBehavior.Review,
    compilationBehavior: SubscriptionReleaseBehavior.Block,
    vrBehavior: SubscriptionReleaseBehavior.Review,
    maxGroupPerformerCount: "3",
    releaseDateRange: SubscriptionReleaseDateRange.All
  },
  torrentSelection: structuredClone(DEFAULT_RESOURCE_SELECTION)
};

export const SUBSCRIPTION_PAGE_SIZE_OPTIONS = [12, 24, 48, 96] as const;
export const LOG_LEVEL_OPTIONS: LogLevel[] = [LogLevel.Debug, LogLevel.Info, LogLevel.Warning, LogLevel.Error];
