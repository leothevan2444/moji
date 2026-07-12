import { isRouteErrorResponse, useRouteError } from "react-router";
import { useTranslation } from "react-i18next";

export function RouteErrorPage() {
  const error = useRouteError();
  const { t } = useTranslation();
  const message = isRouteErrorResponse(error) ? `${error.status} ${error.statusText}` : error instanceof Error ? error.message : t("errors.moduleLoad");
  return <main className="content"><section className="empty-card"><h2>{t("errors.routeTitle")}</h2><p>{message}</p><button type="button" onClick={() => window.location.reload()}>{t("common.reload")}</button></section></main>;
}
