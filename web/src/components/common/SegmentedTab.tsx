interface SegmentedTabOption<T extends string> {
  value: T;
  label: string;
}

interface SegmentedTabProps<T extends string> {
  value: T;
  options: ReadonlyArray<SegmentedTabOption<T>>;
  onChange: (value: T) => void;
  size?: "sm" | "md";
  ariaLabel?: string;
}

/**
 * 受控分段控件：少量互斥选项之间切换（如搜索模式 StashBox / Jackett）。
 * 视觉与 .nav-tab 共享 token，但形态是粘合按钮组而非分离 tab。
 */
export function SegmentedTab<T extends string>({
  value,
  options,
  onChange,
  size = "sm",
  ariaLabel
}: SegmentedTabProps<T>) {
  return (
    <div
      className="segmented-tab"
      data-size={size}
      role="tablist"
      aria-label={ariaLabel}
    >
      {options.map((option) => {
        const active = option.value === value;
        return (
          <button
            key={option.value}
            type="button"
            role="tab"
            aria-selected={active}
            className={`segmented-tab__btn${active ? " is-active" : ""}`}
            onClick={() => onChange(option.value)}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}