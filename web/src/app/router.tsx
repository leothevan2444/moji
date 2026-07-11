import { Navigate, createBrowserRouter } from "react-router";
import { AppShell } from "./AppShell";
import { NotFoundPage } from "./NotFoundPage";
import { AppLayout } from "./AppLayout";

export const router = createBrowserRouter([
  { path: "/", element: <AppShell /> },
  { path: "/tasks", element: <AppShell /> },
  { path: "/tasks/:taskId", element: <AppShell /> },
  { path: "/tasks/:taskId/resolve", element: <AppShell /> },
  { path: "/performers", element: <AppShell /> },
  { path: "/performers/:performerId", element: <AppShell /> },
  { path: "/discover", element: <AppShell /> },
  { path: "/settings", element: <Navigate replace to="/settings/connections" /> },
  { path: "/settings/:section", element: <AppShell /> },
  {
    element: <AppLayout />,
    children: [
      { path: "/stats", lazy: () => import("./routes/StatsRoute") }
    ]
  },
  { path: "*", element: <NotFoundPage /> }
]);
