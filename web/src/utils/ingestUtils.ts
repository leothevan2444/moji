import i18n from "../i18n/i18n";

export function transferActionLabel(action: string) {
  switch (action) {
    case "COPY":
      return i18n.t("home.ingest.copy");
    case "MOVE":
      return i18n.t("home.ingest.move");
    default:
      return action || "—";
  }
}
