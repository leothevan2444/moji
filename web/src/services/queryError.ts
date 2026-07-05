/**
 * Describe a GraphQL/urql error in a human-readable format.
 */
export function describeQueryError(error: unknown) {
  if (!error || typeof error !== "object") return "unknown error";

  const combined = error as {
    message?: string;
    graphQLErrors?: Array<{ message?: string }>;
    networkError?: { message?: string };
  };

  const pieces = [combined.message];
  if (combined.networkError?.message) {
    pieces.push(`network: ${combined.networkError.message}`);
  }
  if (combined.graphQLErrors?.length) {
    pieces.push(
      `graphql: ${combined.graphQLErrors
        .map((item) => describeKnownBackendError(item.message))
        .filter(Boolean)
        .join(" | ")}`
    );
  }

  return pieces
    .filter(Boolean)
    .map((item) => describeKnownBackendError(item))
    .join(" · ") || "unknown error";
}

function describeKnownBackendError(message?: string) {
  if (!message) return message;
  if (message.includes("duplicate torrent task")) {
    return "同一个 torrent 或 magnet 已存在对应的 Moji 任务。";
  }
  if (message.includes("duplicate code task")) {
    return "同一个番号已经存在 Moji 任务，当前请求被严格去重拦截。";
  }
  if (message.includes("duplicate library code")) {
    return "Stash 库中已存在相同番号的影片，当前请求被拦截。";
  }
  if (message.includes("task code is required")) {
    return "任务创建前必须解析出影片番号，但当前输入无法稳定提取 code。";
  }
  return message;
}
