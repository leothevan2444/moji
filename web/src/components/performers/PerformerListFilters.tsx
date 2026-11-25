import { workspaceToolbarIcons } from "../common/workspace-toolbar/icons";
import { useTranslation } from "react-i18next";
import { WorkspaceToolbar } from "../common/workspace-toolbar/WorkspaceToolbar";
import type { ToolbarContent } from "../common/workspace-toolbar/types";
import type { PerformersPageProps } from "./types";

export function PerformerListFilters({ page, selection }: { page: PerformersPageProps; selection: ToolbarContent }) {
  const { t } = useTranslation();
  const checkingAll = Boolean(page.refreshingSubscriptionNow || page.checkActivity?.allRunId || page.checkStarting);
  return <WorkspaceToolbar selecting={page.performerSelectionMode} toggleDisabled={page.performerBatchPending}
    toggleLabel={t(page.performerSelectionMode ? "performerBatch.exitMultiSelect" : "performerBatch.multiSelect")}
    onToggleSelection={page.onTogglePerformerSelectionMode} selectionLabel={t("performerBatch.selectionActions")} selection={selection}
    normal={{
      search: { label: t("performerUi.search"), value: page.subscriptionSearch, onChange: page.onSearchChange }, filters: [
        { id: "sort", label: t("performerWorkspace.sort"), value: page.listSort ?? "default", onChange: value => page.onListSortChange?.(value as NonNullable<PerformersPageProps["listSort"]>), options: (["default", "name", "scene_count", "last_checked"] as const).map(value => ({ value, label: t(`performerWorkspace.sorts.${value}`) })) }
      ], actions: [
        { id: "reload", label: t("performerUi.reloadFromStash"), icon: workspaceToolbarIcons.refresh, onClick: page.onReload, disabled: page.refreshingList, busy: page.refreshingList, spinning: page.refreshingList },
        { id: "check", label: t(checkingAll ? "performerUi.checking" : "performerUi.checkAllSubscribed"), title: t(checkingAll ? "performerUi.checking" : "performerUi.checkAllHelp"), icon: workspaceToolbarIcons.check, onClick: page.onRefreshAll, disabled: page.refreshingSubscriptionNow || !page.subscribedPerformers.length || Boolean(page.checkActivity?.activeCount || page.checkStarting || page.checksLoading || page.checksError), busy: checkingAll },
        { id: "review", label: t("releaseReview.entry", { count: page.reviewPendingCount ?? 0 }), icon: workspaceToolbarIcons.review, href: "/performers/review", showLabel: true },
      ]
    }} />;
}
