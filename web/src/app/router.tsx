import { Navigate, createBrowserRouter } from "react-router";
import { AppShell } from "./AppShell";
import { NotFoundPage } from "./NotFoundPage";
import { AppLayout } from "./AppLayout";

export const router = createBrowserRouter([
  { path: "/performers", element: <AppShell /> },
  { path: "/performers/:performerId", element: <AppShell /> },
  { path: "/discover", element: <AppShell /> },
  { path: "/settings", element: <Navigate replace to="/settings/connections" /> },
  {
    element: <AppLayout />,
    children: [
      { index: true, lazy: () => import("./routes/HomeRoute") },
      { path: "/tasks", lazy: () => import("./routes/TasksRoute") },
      { path: "/tasks/:taskId", lazy: () => import("./routes/TasksRoute") },
      { path: "/tasks/:taskId/resolve", lazy: () => import("./routes/TasksRoute") },
      { path: "/stats", lazy: () => import("./routes/StatsRoute") },
      { path: "/settings/:section", lazy: () => import("./routes/SettingsRoute") }
    ]
  },
  { path: "*", element: <NotFoundPage /> }
]);
