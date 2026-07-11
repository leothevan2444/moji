import { Navigate, createBrowserRouter } from "react-router";
import { AppShell } from "./AppShell";
import { NotFoundPage } from "./NotFoundPage";
import { AppLayout } from "./AppLayout";

export const router = createBrowserRouter([
  { path: "/tasks", element: <AppShell /> },
  { path: "/tasks/:taskId", element: <AppShell /> },
  { path: "/tasks/:taskId/resolve", element: <AppShell /> },
  { path: "/performers", element: <AppShell /> },
  { path: "/performers/:performerId", element: <AppShell /> },
  { path: "/discover", element: <AppShell /> },
  { path: "/settings", element: <Navigate replace to="/settings/connections" /> },
  {
    element: <AppLayout />,
    children: [
      { index: true, lazy: () => import("./routes/HomeRoute") },
      { path: "/stats", lazy: () => import("./routes/StatsRoute") },
      { path: "/settings/:section", lazy: () => import("./routes/SettingsRoute") }
    ]
  },
  { path: "*", element: <NotFoundPage /> }
]);
