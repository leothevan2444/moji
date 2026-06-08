import quickstart from "./quickstart.md?raw";
import tasks from "./tasks.md?raw";
import stash from "./stash.md?raw";
import troubleshooting from "./troubleshooting.md?raw";

export const HELP_TOPICS = [
  {
    id: "quickstart",
    title: "快速开始",
    markdown: quickstart
  },
  {
    id: "tasks",
    title: "任务说明",
    markdown: tasks
  },
  {
    id: "stash",
    title: "Stash 集成",
    markdown: stash
  },
  {
    id: "troubleshooting",
    title: "排障指南",
    markdown: troubleshooting
  }
] as const;

export type HelpTopic = (typeof HELP_TOPICS)[number];
export type HelpTopicId = HelpTopic["id"];
