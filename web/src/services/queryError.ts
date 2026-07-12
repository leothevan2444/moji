/**
 * Describe a GraphQL/urql error in a human-readable format.
 */
export function describeQueryError(error: unknown) {
  if (!error || typeof error !== "object") return i18n.t("errors.unknown");

  const combined = error as {
    message?: string;
    graphQLErrors?: Array<{ message?: string; extensions?: Record<string, unknown> }>;
    networkError?: { message?: string };
  };

  const coded = combined.graphQLErrors?.map(describeCodedBackendError).filter(Boolean) ?? [];
  if (coded.length) return coded.join(" · ");

  const pieces: Array<string | undefined> = [];
  if (combined.networkError?.message) {
    pieces.push(i18n.t("errors.network", { message: combined.networkError.message }));
  }
  if (combined.graphQLErrors?.length) {
    pieces.push(
      `${combined.graphQLErrors
        .map((item) => describeKnownBackendError(item.message))
        .filter(Boolean)
        .join(" | ")}`
    );
  }

  return pieces
    .filter(Boolean)
    .map((item) => describeKnownBackendError(item))
    .join(" · ") || describeKnownBackendError(combined.message) || i18n.t("errors.unknown");
}

function describeCodedBackendError(error: { extensions?: Record<string, unknown> }) {
  const code = typeof error.extensions?.code === "string" ? error.extensions.code : "";
  if (!code) return "";
  const correlationId = typeof error.extensions?.correlationId === "string" ? error.extensions.correlationId : "";
  const key = i18n.exists(`errors.backend.${code}`) ? `errors.backend.${code}` : i18n.exists(`errorExtra.${code}`) ? `errorExtra.${code}` : i18n.exists(`errorSpecial.${code}`) ? `errorSpecial.${code}` : "errors.unknown";
  const translated = i18n.t(key, (error.extensions?.params as Record<string, unknown>) ?? {});
  return correlationId ? `${translated} (${i18n.t("errors.withId", { id: correlationId })})` : translated;
}

function describeKnownBackendError(message?: string) {
  if (!message) return message;
  if (message.includes("duplicate torrent task")) {
    return i18n.t("errors.backend.DUPLICATE_TORRENT_TASK");
  }
  if (message.includes("duplicate code task")) {
    return i18n.t("errors.backend.DUPLICATE_CODE_TASK");
  }
  if (message.includes("duplicate library code")) {
    return i18n.t("errors.backend.DUPLICATE_LIBRARY_CODE");
  }
  if (message.includes("task code is required")) {
    return i18n.t("errors.backend.TASK_CODE_REQUIRED");
  }
  return message;
}
import i18n from "../i18n/i18n";
