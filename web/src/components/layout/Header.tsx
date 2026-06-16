import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faChartColumn,
  faCircleQuestion,
  faGear
} from "@fortawesome/free-solid-svg-icons";
import { NAV_TABS } from "../../constants";
import type { DrawerKey, TabKey } from "../../types";

interface HeaderProps {
  tab: TabKey;
  onTabChange: (tab: TabKey) => void;
  onOpenDrawer: (drawer: Exclude<DrawerKey, null>) => void;
}

export function Header({ tab, onTabChange, onOpenDrawer }: HeaderProps) {
  return (
    <header className="masthead">
      <div className="masthead__brand">
        <div className="title-row">
          <h1>Moji</h1>
        </div>
      </div>
      <div className="masthead__actions" aria-label="主导航">
        <div className="masthead__navgroup">
          <div className="tab-group">
            {NAV_TABS.map((item) => (
              <button
                key={item}
                type="button"
                className={`nav-tab ${tab === item ? "is-active" : ""}`}
                onClick={() => onTabChange(item)}
              >
                {item}
              </button>
            ))}
          </div>
        </div>

        <div className="masthead__toolgroup">
          <div className="utility-group">
            <button
              type="button"
              className="utility-button utility-icon-button"
              onClick={() => onOpenDrawer("stats")}
              aria-label="统计"
              title="统计"
            >
              <FontAwesomeIcon icon={faChartColumn} aria-hidden="true" />
            </button>
            <button
              type="button"
              className="utility-button utility-icon-button"
              onClick={() => onOpenDrawer("settings")}
              aria-label="设置"
              title="设置"
            >
              <FontAwesomeIcon icon={faGear} aria-hidden="true" />
            </button>
            <button
              type="button"
              className="utility-button utility-icon-button"
              onClick={() => onOpenDrawer("help")}
              aria-label="帮助"
              title="帮助"
            >
              <FontAwesomeIcon icon={faCircleQuestion} aria-hidden="true" />
            </button>
          </div>
        </div>
      </div>
    </header>
  );
}
