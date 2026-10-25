import { faBinoculars, faClipboardQuestion, faListCheck, faRotate, faXmark } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import type { ReactNode } from "react";
import { PerformerToolbar } from "./PerformerToolbar";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { SUBSCRIPTION_PAGE_SIZE_OPTIONS } from "../../constants";
import type { PerformersPageProps } from "./types";

export function PerformerListFilters({ page, children }: { page: PerformersPageProps; children: ReactNode; }) {
  const { t } = useTranslation();
  const { reviewPendingCount = 0, subscriptionSearch, subscriptionPageSize, listSort = "default", performerSelectionMode, refreshingList, refreshingSubscriptionNow, subscribedPerformers, onSearchChange, onListSortChange, onPageSizeChange, onTogglePerformerSelectionMode, onReload, onRefreshAll } = page;
  return <div className="performer-list-controls">
    <PerformerToolbar expanded={performerSelectionMode} renderControls={panelId => <>
      <div className="toolbar-inline toolbar-inline--performer-list">
        <input type="search" aria-label={t("performerUi.search")} placeholder={t("performerUi.search")} value={subscriptionSearch} onChange={e => onSearchChange(e.target.value)} />
        <label>{t("performerWorkspace.sort")}<select value={listSort} onChange={e => onListSortChange?.(e.target.value as typeof listSort)}>{(["default", "name", "scene_count", "last_checked"] as const).map(value => <option key={value} value={value}>{t(`performerWorkspace.sorts.${value}`)}</option>)}</select></label>
        <select aria-label={t("performerWorkspace.pageSize")} value={subscriptionPageSize} onChange={e => onPageSizeChange(Number(e.target.value))}>{SUBSCRIPTION_PAGE_SIZE_OPTIONS.map(size => <option key={size} value={size}>{t("performerUi.pageSizeRows", { size })}</option>)}</select>
        <button type="button" className="ghost-button task-icon-button task-icon-button--bordered performer-toolbar__toggle" onClick={onTogglePerformerSelectionMode} disabled={page.performerBatchPending} aria-pressed={performerSelectionMode} aria-expanded={performerSelectionMode} aria-controls={panelId} aria-label={t(performerSelectionMode ? "performerBatch.exitMultiSelect" : "performerBatch.multiSelect")} title={t(performerSelectionMode ? "performerBatch.exitMultiSelect" : "performerBatch.multiSelect")}><FontAwesomeIcon icon={performerSelectionMode ? faXmark : faListCheck} /></button>
        <button type="button" className="ghost-button task-icon-button task-icon-button--bordered" onClick={onReload} disabled={refreshingList} aria-label={t("performerUi.reloadFromStash")} title={t("performerUi.reloadFromStash")}><FontAwesomeIcon icon={faRotate} className={refreshingList ? "is-spinning" : undefined} /></button>
        <button type="button" className="ghost-button task-icon-button task-icon-button--bordered" onClick={onRefreshAll} disabled={refreshingSubscriptionNow || subscribedPerformers.length === 0} aria-label={t(refreshingSubscriptionNow ? "performerUi.checking" : "performerUi.checkAllSubscribed")} aria-busy={refreshingSubscriptionNow} title={t(refreshingSubscriptionNow ? "performerUi.checking" : "performerUi.checkAllHelp")}><FontAwesomeIcon icon={faBinoculars} /></button>
        <Link className="ghost-button release-review-link" to="/performers/review"><FontAwesomeIcon icon={faClipboardQuestion} /> {t("releaseReview.entry", { count: reviewPendingCount })}</Link>
      </div>
    </>}>{children}</PerformerToolbar>
  </div>;
}
