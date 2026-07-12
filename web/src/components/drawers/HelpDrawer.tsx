import { HELP_CATEGORIES, HELP_TOPICS, type HelpLanguage, type HelpTopicId } from "../../help";
import { MarkdownBlock } from "../MarkdownBlock";
import { useLocale } from "../../i18n/LocaleProvider";
import { useTranslation } from "react-i18next";

interface HelpDrawerProps {
  topicId: HelpTopicId;
  onTopicChange: (topicId: HelpTopicId) => void;
}

export function HelpDrawer({ topicId, onTopicChange }: HelpDrawerProps) {
  const { locale } = useLocale();
  const { t } = useTranslation();
  const language: HelpLanguage = locale === "zh-CN" ? "zh" : "en";
  const selectedTopic = HELP_TOPICS.find((topic) => topic.id === topicId) ?? HELP_TOPICS[0];

  return (
    <div className="help__layout">
      <div className="help__tabs">
        <nav className="help__navigation" aria-label={t("help.navigation")}>
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
