import { HELP_TOPICS, type HelpTopicId } from "../../help";
import { MarkdownBlock } from "../MarkdownBlock";

interface HelpDrawerProps {
  topicId: HelpTopicId;
  onTopicChange: (topicId: HelpTopicId) => void;
}

export function HelpDrawer({ topicId, onTopicChange }: HelpDrawerProps) {
  const selectedTopic = HELP_TOPICS.find((topic) => topic.id === topicId) ?? HELP_TOPICS[0];

  return (
    <div className="help-layout">
      <div className="help-tabs">
        {HELP_TOPICS.map((topic) => (
          <button
            key={topic.id}
            type="button"
            className={`help-tab ${topicId === topic.id ? "is-active" : ""}`}
            onClick={() => onTopicChange(topic.id)}
          >
            {topic.title}
          </button>
        ))}
      </div>
      <article className="drawer-card help-card">
        <MarkdownBlock markdown={selectedTopic.markdown} />
      </article>
    </div>
  );
}
