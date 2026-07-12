import i18n from "../i18n/i18n";

export function deliveryModeLabel(mode: string) {
  switch (mode) {
    case "PATH_MAP":
      return i18n.t("home.ingest.pathMap");
    case "TRANSFER":
      return i18n.t("home.ingest.transfer");
    default:
      return mode || i18n.t("home.ingest.none");
  }
}

export function transferActionLabel(action: string) {
  switch (action) {
    case "COPY":
      return i18n.t("home.ingest.copy");
    case "MOVE":
      return i18n.t("home.ingest.move");
    case "SYMLINK":
      return i18n.t("home.ingest.symlink");
    default:
      return action || "—";
  }
}
