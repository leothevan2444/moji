import { useRef, useState, type ReactNode } from "react";
import { useClickOutside } from "../../hooks/useClickOutside";

export interface MenuItem {
  key: string;
  label: ReactNode;
  onSelect: () => void;
  disabled?: boolean;
  hint?: ReactNode;
}

interface MenuProps {
  /** 触发按钮的内容（任意节点，比如带图标的 label）。 */
  triggerLabel: ReactNode;
  /** 触发按钮的可访问名。 */
  triggerAriaLabel?: string;
  /** 菜单条目。 */
  items: MenuItem[];
  /** 菜单的整体标签，用于 listbox aria-label。 */
  ariaLabel: string;
  /** 菜单对齐触发器：左对齐或右对齐。默认右对齐。 */
  align?: "start" | "end";
}

/**
 * 通用下拉菜单：
 *  - 触发器始终渲染为 `<button type="button">`，避免误触发表单提交；
 *  - 打开后通过 `useClickOutside` 监听外部点击与 Esc 键关闭；
 *  - 菜单项使用 `role="option"` + `aria-disabled`，键盘 / 屏幕阅读器可识别；
 *  - 由父组件控制 `open` 状态——外部亦可强制关闭（例如保存后）。
 */
export function Menu({ triggerLabel, triggerAriaLabel, items, ariaLabel, align = "end" }: MenuProps) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  useClickOutside([rootRef], () => setOpen(false), open);

  return (
    <div ref={rootRef} className={`menu menu--${align}${open ? " is-open" : ""}`}>
      <button
        type="button"
        className="menu__trigger"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={triggerAriaLabel}
        onClick={() => setOpen((current) => !current)}
      >
        {triggerLabel}
      </button>
      {open ? (
        <ul className="menu__list" role="listbox" aria-label={ariaLabel}>
          {items.map((item) => (
            <li key={item.key} role="option" aria-disabled={item.disabled ?? false} className="menu__option-wrap">
              <button
                type="button"
                className="menu__option"
                disabled={item.disabled}
                onClick={() => {
                  if (item.disabled) return;
                  setOpen(false);
                  item.onSelect();
                }}
              >
                <span className="menu__option-label">{item.label}</span>
                {item.hint ? <span className="menu__option-hint">{item.hint}</span> : null}
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}