import { useRef } from "react";
import { useClickOutside } from "../../hooks/useClickOutside";
import { useTranslation } from "react-i18next";

interface SearchHistoryDropdownProps {
  history: string[];
  visible: boolean;
  onPick: (query: string) => void;
  onRemove: (query: string) => void;
  onClear: () => void;
  onDismiss: () => void;
}

/**
 * 搜索历史下拉：聚焦 + 空 query 时显示，点击条目直接发起搜索。
 *  - 通过 useClickOutside + Esc 关闭；
 *  - 单条删除按钮避免清空整段；
 *  - 空状态不渲染。
 */
export function SearchHistoryDropdown({
  history,
  visible,
  onPick,
  onRemove,
  onClear,
  onDismiss
}: SearchHistoryDropdownProps) {
  const { t } = useTranslation();
  const rootRef = useRef<HTMLDivElement>(null);

  useClickOutside([rootRef], onDismiss, visible);

  if (!visible || history.length === 0) return null;

  return (
    <div ref={rootRef} className="search-history" role="listbox" aria-label={t("discoverUi.history.aria")}>
      <div className="search-history__head">
        <span>{t("discoverUi.history.recent")}</span>
        <button type="button" className="search-history__clear" onClick={onClear}>
          {t("discoverUi.history.clear")}
        </button>
      </div>
      <ul className="search-history__list">
        {history.map((entry) => (
          <li key={entry} className="search-history__item">
            <button
              type="button"
              className="search-history__pick"
              onClick={() => onPick(entry)}
            >
              {entry}
            </button>
            <button
              type="button"
              className="search-history__remove"
              aria-label={t("discoverUi.history.remove", { entry })}
              onClick={(event) => {
                event.stopPropagation();
                onRemove(entry);
              }}
            >
              ×
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
