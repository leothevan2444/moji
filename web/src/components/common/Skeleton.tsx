type SkeletonVariant = "block" | "title" | "meta" | "poster" | "chip" | "row";

interface SkeletonProps {
  variant?: SkeletonVariant;
  count?: number;
}

/**
 * 单个或一组骨架占位元素。变体对应不同形态（block/title/meta/poster/chip/row），
 * count 控制连续渲染次数（如 6 张骨架卡片）。
 */
export function Skeleton({ variant = "block", count = 1 }: SkeletonProps) {
  if (count <= 1) {
    return <div className={`skeleton skeleton--${variant}`} aria-hidden="true" />;
  }
  return (
    <>
      {Array.from({ length: count }, (_, index) => (
        <div
          key={index}
          className={`skeleton skeleton--${variant}`}
          aria-hidden="true"
        />
      ))}
    </>
  );
}

/**
 * 加载占位卡片，复用 .candidate-card 的视觉密度，避免首屏抖动。
 * count 默认 6 张，与单页结果显示数量一致。
 */
export function SkeletonCardList({ count = 6 }: { count?: number }) {
  return (
    <div className="discovery-results">
      {Array.from({ length: count }, (_, index) => (
        <div key={index} className="skeleton-card" aria-hidden="true">
          <div className="skeleton-card__head">
            <Skeleton variant="poster" />
            <div className="skeleton-card__body">
              <Skeleton variant="title" />
              <Skeleton variant="meta" />
              <Skeleton variant="row" />
              <Skeleton variant="chip" />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}