import { useQuery } from "urql";
import {
  JackettIndexersDocumentDocument,
  type JackettIndexersDocumentQuery,
  type JackettIndexersDocumentQueryVariables
} from "../graphql/generated/graphql";

/**
 * 拉取 Jackett 配置的索引器列表。仅在 `enabled` 为 true 时请求，避免 StashBox
 * 模式下做无用功。后端在 Jackett 未配置时返回空数组，前端据此渲染"未连接"提示。
 */
export function useJackettIndexers(enabled: boolean) {
  const [{ data, fetching, error }] = useQuery<
    JackettIndexersDocumentQuery,
    JackettIndexersDocumentQueryVariables
  >({
    query: JackettIndexersDocumentDocument,
    pause: !enabled
  });

  return {
    indexers: (data?.jackettIndexers ?? []).filter((indexer) => indexer.enabled),
    fetching,
    error
  };
}
