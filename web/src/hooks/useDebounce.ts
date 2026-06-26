import { useEffect, useState } from "react";

/**
 * 返回延迟 `delay` 毫秒后的 `value`。在 value 连续变化期间只更新一次，
 * 避免把抖动值（如搜索框连续输入）推到下游——常用于：
 *
 *  - 网络请求（不过本项目由 GraphQL urql + useDeferredValue 接管，此处多用
 *    于"边输入边过滤本地列表"的预览场景）；
 *  - 过滤历史下拉；
 *  - 任何"连续触发会浪费算力"的副作用。
 */
export function useDebounce<T>(value: T, delay = 400): T {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const handle = window.setTimeout(() => setDebounced(value), delay);
    return () => window.clearTimeout(handle);
  }, [value, delay]);

  return debounced;
}