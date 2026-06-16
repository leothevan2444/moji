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
        .map((item) => item.message)
        .filter(Boolean)
        .join(" | ")}`
    );
  }

  return pieces.filter(Boolean).join(" · ") || "unknown error";
}
