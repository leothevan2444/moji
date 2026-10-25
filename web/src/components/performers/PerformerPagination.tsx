import { faAngleLeft } from "@fortawesome/free-solid-svg-icons/faAngleLeft";
import { faAngleRight } from "@fortawesome/free-solid-svg-icons/faAngleRight";
import { faAnglesLeft } from "@fortawesome/free-solid-svg-icons/faAnglesLeft";
import { faAnglesRight } from "@fortawesome/free-solid-svg-icons/faAnglesRight";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import type { PerformersPageProps } from "./types";

export function PerformerPagination({ page }: { page: Pick<PerformersPageProps, "stashPerformerPage" | "fetchingStashPerformers" | "onPrevPage" | "onNextPage" | "onPageChange">; }) {
  const { t } = useTranslation();
  const { stashPerformerPage, fetchingStashPerformers, onPrevPage, onNextPage, onPageChange } = page;
  const [pageInput, setPageInput] = useState("");
  const targetPage = Number(pageInput);
  const canJump = Number.isInteger(targetPage) && targetPage >= 1 && targetPage <= (stashPerformerPage?.totalPages ?? 1) && targetPage !== stashPerformerPage?.page;
  useEffect(() => setPageInput(""), [stashPerformerPage?.page]);
  return stashPerformerPage && stashPerformerPage.totalPages > 1 ? (
    <div className="pagination-bar pagination-bar--performers">
      <button type="button" className="ghost-button task-icon-button task-icon-button--bordered" disabled={!stashPerformerPage.hasPrevPage || fetchingStashPerformers} onClick={() => onPageChange(1)} aria-label={t("performerUi.firstPage")} title={t("performerUi.firstPage")}>
        <FontAwesomeIcon icon={faAnglesLeft} />
      </button>
      <button type="button" className="ghost-button task-icon-button task-icon-button--bordered" disabled={!stashPerformerPage.hasPrevPage || fetchingStashPerformers} onClick={onPrevPage} aria-label={t("performerUi.previous")} title={t("performerUi.previous")}>
        <FontAwesomeIcon icon={faAngleLeft} />
      </button>
      <span className="status-chip tone-neutral">
        {t("performerUi.page", { page: stashPerformerPage.page, pages: stashPerformerPage.totalPages })}
      </span>
      <button type="button" className="ghost-button task-icon-button task-icon-button--bordered" disabled={!stashPerformerPage.hasNextPage || fetchingStashPerformers} onClick={onNextPage} aria-label={t("performerUi.next")} title={t("performerUi.next")}>
        <FontAwesomeIcon icon={faAngleRight} />
      </button>
      <button type="button" className="ghost-button task-icon-button task-icon-button--bordered" disabled={!stashPerformerPage.hasNextPage || fetchingStashPerformers} onClick={() => onPageChange(stashPerformerPage.totalPages)} aria-label={t("performerUi.lastPage")} title={t("performerUi.lastPage")}>
        <FontAwesomeIcon icon={faAnglesRight} />
      </button>
      <form className="pagination-bar__jump" onSubmit={(event) => {
        event.preventDefault();
        if (canJump && !fetchingStashPerformers) onPageChange(targetPage);
      }}>
        <input
          type="number"
          min={1}
          max={stashPerformerPage.totalPages}
          step={1}
          required
          value={pageInput}
          placeholder={String(stashPerformerPage.page)}
          aria-label={t("performerUi.targetPage")}
          disabled={fetchingStashPerformers}
          onChange={(event) => setPageInput(event.target.value)}
        />
        <button type="submit" className="ghost-button" disabled={!canJump || fetchingStashPerformers}>
          {t("performerUi.jumpToPage")}
        </button>
      </form>
    </div>
  ) : null;
}
