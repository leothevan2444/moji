import { isRouteErrorResponse, useRouteError } from "react-router";

export function RouteErrorPage() {
  const error = useRouteError();
  const message = isRouteErrorResponse(error) ? `${error.status} ${error.statusText}` : error instanceof Error ? error.message : "页面模块加载失败";
  return <main className="content"><section className="empty-card"><h2>页面加载失败</h2><p>{message}</p><button type="button" onClick={() => window.location.reload()}>重新加载</button></section></main>;
}
