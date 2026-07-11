import { useState } from "react";
import { HELP_CATEGORIES, HELP_TOPICS, type HelpLanguage, type HelpTopicId } from "../../help";
import { MarkdownBlock } from "../MarkdownBlock";

interface HelpDrawerProps {
  topicId: HelpTopicId;
  onTopicChange: (topicId: HelpTopicId) => void;
}

export function HelpDrawer({ topicId, onTopicChange }: HelpDrawerProps) {
  const [language, setLanguage] = useState<HelpLanguage>(() => {
    if (typeof navigator !== "undefined" && navigator.language.toLowerCase().startsWith("zh")) return "zh";
    return "en";
  });
  const selectedTopic = HELP_TOPICS.find((topic) => topic.id === topicId) ?? HELP_TOPICS[0];

  return (
    <div className="help__layout">
      <div className="help__tabs">
        <div className="help__languages" role="group" aria-label="Help language">
          <button type="button" className={language === "en" ? "is-active" : ""} onClick={() => setLanguage("en")}>EN</button>
          <button type="button" className={language === "zh" ? "is-active" : ""} onClick={() => setLanguage("zh")}>中文</button>
        </div>
        <nav className="help__navigation" aria-label={language === "zh" ? "帮助文档" : "Help documentation"}>
          {HELP_CATEGORIES.map((category) => (
            <details key={category.id} className="help__category" open>
              <summary>{category.title[language]}</summary>
              <div className="help__category-topics">
                {category.topics.map((topic) => (
                  <button
                    key={topic.id}
                    type="button"
                    className={`help__tab ${topicId === topic.id ? "is-active" : ""}`}
                    aria-current={topicId === topic.id ? "page" : undefined}
                    onClick={() => onTopicChange(topic.id)}
                  >
                    {topic.title[language]}
                  </button>
                ))}
              </div>
            </details>
          ))}
        </nav>
      </div>
      <article className="drawer-card help__card">
        <MarkdownBlock markdown={selectedTopic.markdown[language]} />
      </article>
    </div>
  );
}
