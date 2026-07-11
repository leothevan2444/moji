import type { RouteObject } from "react-router";
import { Navigate } from "react-router";
import { NotFoundPage } from "./NotFoundPage";
import { AppLayout } from "./AppLayout";
import { RouteErrorPage } from "./RouteErrorPage";

export const appRoutes: RouteObject[] = [
  { path: "/settings", element: <Navigate replace to="/settings/connections" /> },
  {
    element: <AppLayout />,
    errorElement: <RouteErrorPage />,
    children: [
      { index: true, lazy: () => import("./routes/HomeRoute") },
      { path: "/tasks", lazy: () => import("./routes/TasksRoute") },
      { path: "/tasks/:taskId", lazy: () => import("./routes/TasksRoute") },
      { path: "/tasks/:taskId/resolve", lazy: () => import("./routes/TasksRoute") },
      { path: "/discover", lazy: () => import("./routes/DiscoverRoute") },
      { path: "/performers", lazy: () => import("./routes/PerformersRoute") },
      { path: "/performers/:performerId", lazy: () => import("./routes/PerformersRoute") },
      { path: "/stats", lazy: () => import("./routes/StatsRoute") },
      { path: "/settings/:section", lazy: () => import("./routes/SettingsRoute") }
    ]
  },
  { path: "*", element: <NotFoundPage /> }
];
