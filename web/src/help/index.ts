import introductionEn from "./en/introduction.md?raw";
import firstSetupEn from "./en/getting-started.md?raw";
import homeEn from "./en/home.md?raw";
import discoverEn from "./en/discover.md?raw";
import performersEn from "./en/performers.md?raw";
import tasksEn from "./en/tasks.md?raw";
import servicesDeliveryEn from "./en/services-and-delivery.md?raw";
import taskSubscriptionEn from "./en/tasks-and-subscriptions.md?raw";
import systemEn from "./en/system.md?raw";
import troubleshootingEn from "./en/troubleshooting.md?raw";
import introductionZh from "./zh/introduction.md?raw";
import firstSetupZh from "./zh/getting-started.md?raw";
import homeZh from "./zh/home.md?raw";
import discoverZh from "./zh/discover.md?raw";
import performersZh from "./zh/performers.md?raw";
import tasksZh from "./zh/tasks.md?raw";
import servicesDeliveryZh from "./zh/services-and-delivery.md?raw";
import taskSubscriptionZh from "./zh/tasks-and-subscriptions.md?raw";
import systemZh from "./zh/system.md?raw";
import troubleshootingZh from "./zh/troubleshooting.md?raw";

import type { HelpLanguage, HelpTopicId } from "./types";
export type { HelpLanguage, HelpTopicId } from "./types";

export interface HelpTopic {
  id: HelpTopicId;
  title: Record<HelpLanguage, string>;
  markdown: Record<HelpLanguage, string>;
}

export interface HelpCategory {
  id: string;
  title: Record<HelpLanguage, string>;
  topics: readonly HelpTopic[];
}

export const HELP_CATEGORIES: readonly HelpCategory[] = [
  {
    id: "getting-started",
    title: { en: "Getting started", zh: "开始使用" },
    topics: [
      { id: "introduction", title: { en: "Introduction", zh: "产品简介" }, markdown: { en: introductionEn, zh: introductionZh } },
      { id: "first-setup", title: { en: "First setup", zh: "首次配置" }, markdown: { en: firstSetupEn, zh: firstSetupZh } }
    ]
  },
  {
    id: "user-guide",
    title: { en: "User guide", zh: "用户指南" },
    topics: [
      { id: "home", title: { en: "Home", zh: "主页" }, markdown: { en: homeEn, zh: homeZh } },
      { id: "discover", title: { en: "Discover", zh: "发现" }, markdown: { en: discoverEn, zh: discoverZh } },
      { id: "performers", title: { en: "Performers", zh: "演员" }, markdown: { en: performersEn, zh: performersZh } },
      { id: "tasks", title: { en: "Tasks", zh: "任务" }, markdown: { en: tasksEn, zh: tasksZh } }
    ]
  },
  {
    id: "configuration",
    title: { en: "Configuration", zh: "配置" },
    topics: [
      { id: "services-and-delivery", title: { en: "Services & delivery", zh: "服务与交付" }, markdown: { en: servicesDeliveryEn, zh: servicesDeliveryZh } },
      { id: "tasks-and-subscriptions", title: { en: "Tasks & subscriptions", zh: "任务与订阅" }, markdown: { en: taskSubscriptionEn, zh: taskSubscriptionZh } },
      { id: "system", title: { en: "System", zh: "系统" }, markdown: { en: systemEn, zh: systemZh } }
    ]
  },
  {
    id: "support",
    title: { en: "Support", zh: "支持" },
    topics: [
      { id: "troubleshooting", title: { en: "Troubleshooting", zh: "故障排除" }, markdown: { en: troubleshootingEn, zh: troubleshootingZh } }
    ]
  }
] as const;

export const HELP_TOPICS: readonly HelpTopic[] = HELP_CATEGORIES.flatMap((category) => category.topics);
