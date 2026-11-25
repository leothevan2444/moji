import { faAngleLeft, faAngleRight, faAnglesLeft, faAnglesRight, faEllipsis } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useCallback, useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { PAGE_SIZE_OPTIONS } from "../../../constants";
import { PaginationOptions } from "./PaginationOptions";
import "../../../styles/pagination.scss";

type Layout = "full" | "size-hidden" | "compact";

interface PaginationProps {
  label: string;
  position: "top" | "bottom";
  page: number;
  pageSize: number;
  totalPages: number | null;
  /** Null while loading; with unknown totalPages this is the loaded matching count. */
  totalCount: number | null;
  hasNextPage?: boolean;
  loading?: boolean;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
}

export function Pagination({ label, position, page, pageSize, totalPages, totalCount, hasNextPage = false, loading = false, onPageChange, onPageSizeChange }: PaginationProps) {
  const { t, i18n } = useTranslation();
  const id = useId();
  const viewport = useRef<HTMLDivElement>(null);
  const measure = useRef<HTMLDivElement>(null);
  const sizeHiddenMeasure = useRef<HTMLDivElement>(null);
  const more = useRef<HTMLButtonElement>(null);
  const sizeControl = useRef<HTMLSelectElement>(null);
  const focusCollapsed = useRef(false);
  const focusExpanded = useRef(false);
  const [layout, setLayout] = useState<Layout>("full");
  const previousLayout = useRef<Layout>("full");
  const [optionsOpen, setOptionsOpen] = useState(false);
  const empty = totalCount === 0 && totalPages !== null;
  const displayPage = empty ? 0 : page;
  const maximumPage = totalPages ?? Math.max(page, Math.ceil((totalCount ?? 0) / pageSize));
  const disabled = loading || totalCount === null;
  const firstDisabled = disabled || empty || page <= 1;
  const lastDisabled = disabled || empty || totalPages === null || page >= totalPages;
  const nextDisabled = disabled || empty || (totalPages === null ? !hasNextPage : page >= totalPages);
  const [draft, setDraft] = useState(String(displayPage));
  const number = (value: number) => value.toLocaleString(i18n.resolvedLanguage);
  const pagesText = totalPages === null ? "—" : number(empty ? 0 : totalPages);
  const countLabel = totalCount === null ? t("pagination.unknownItems") :
    t(totalPages === null ? "pagination.loadedItems" : "pagination.items", { total: number(totalCount) });
  const inputWidth = `calc(${Math.max(2, String(maximumPage).length)}ch + 16px)`;

  useEffect(() => setDraft(String(displayPage)), [displayPage, pageSize]);
  useLayoutEffect(() => {
    const update = () => {
      if (!viewport.current || !measure.current || !sizeHiddenMeasure.current) return;
      const available = viewport.current.clientWidth - 8 - 2;
      const next: Layout = measure.current.scrollWidth <= available ? "full" : sizeHiddenMeasure.current.scrollWidth <= available ? "size-hidden" : "compact";
      const focused = document.activeElement;
      if (focused instanceof Element && viewport.current.contains(focused) &&
        ((next !== "full" && focused.closest(".pagination__size-slot")) || (next === "compact" && focused.closest(".pagination__edge")))) focusCollapsed.current = true;
      if (next === "full" && focused === more.current) focusExpanded.current = true;
      setLayout(next);
    };
    update();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(update);
    if (viewport.current) observer.observe(viewport.current);
    if (measure.current) observer.observe(measure.current);
    if (sizeHiddenMeasure.current) observer.observe(sizeHiddenMeasure.current);
    return () => observer.disconnect();
  }, []);
  useLayoutEffect(() => {
    if (layout !== "full" && focusCollapsed.current) { focusCollapsed.current = false; more.current?.focus(); }
    if (layout === "full" && (optionsOpen || focusExpanded.current)) {
      focusExpanded.current = false;
      setOptionsOpen(false);
      sizeControl.current?.focus();
    } else if (optionsOpen && previousLayout.current !== layout) {
      setOptionsOpen(false);
      more.current?.focus();
    }
    previousLayout.current = layout;
  }, [layout, optionsOpen]);

  const closeOptions = useCallback((restoreFocus: boolean) => {
    // A resize can move the anchor offscreen before the row restores its full layout.
    const fullFits = viewport.current && measure.current && measure.current.scrollWidth <= viewport.current.clientWidth - 10;
    const panelFocused = document.getElementById(id)?.contains(document.activeElement);
    if (fullFits && (restoreFocus || panelFocused)) focusExpanded.current = true;
    setOptionsOpen(false);
    if (restoreFocus && !fullFits) more.current?.focus();
  }, [id]);
  const navigate = (target: number) => {
    setDraft(String(displayPage));
    if (!disabled && !empty && Number.isSafeInteger(target) && target >= 1 && target !== page) onPageChange(target);
  };
  const renderSize = (measuring: boolean) => (
    <select ref={measuring ? undefined : sizeControl} className="pagination__size" aria-label={measuring ? undefined : t("pagination.pageSize")}
      value={pageSize} disabled={disabled || measuring} onChange={measuring ? undefined : event => onPageSizeChange(Number(event.target.value))}>
      {PAGE_SIZE_OPTIONS.map(size => <option key={size} value={size}>{t("pagination.sizeOption", { size })}</option>)}
    </select>
  );
  const renderRow = (measuring: boolean, sizeHidden = false) => {
    const arrow = (action: "first" | "previous" | "next" | "last", icon: typeof faAngleLeft, inactive: boolean, target: number) => {
      const className = `ghost-button pagination__button${action === "first" || action === "last" ? " pagination__edge" : ""}`;
      return measuring ? <span className={className}><FontAwesomeIcon icon={icon} /></span> :
        <button type="button" className={className} disabled={inactive} onClick={() => navigate(target)} aria-label={t(`pagination.${action}`)} title={t(`pagination.${action}`)}><FontAwesomeIcon icon={icon} aria-hidden="true" /></button>;
    };
    const pageContents = <>
      {measuring ? <span className="pagination__input" style={{ width: inputWidth }}>{displayPage}</span> :
        <input className="pagination__input" style={{ width: inputWidth }} type="text" inputMode="numeric" dir="ltr" aria-label={t("pagination.pageNumber")}
          title={t("pagination.jumpHint")} value={totalCount === null ? "" : draft} placeholder="—" disabled={disabled || empty}
          onChange={event => { event.target.setCustomValidity(""); setDraft(event.target.value); }}
          onBlur={event => { event.target.setCustomValidity(""); setDraft(String(displayPage)); }}
          onKeyDown={event => { if (event.key === "Escape") { event.currentTarget.setCustomValidity(""); setDraft(String(displayPage)); } }} />}
      <span className="pagination__pages" aria-hidden="true">/ {pagesText}</span>
    </>;
    return <>
      <div className="pagination__navigation">
        {arrow("first", faAnglesLeft, firstDisabled, 1)}
        {arrow("previous", faAngleLeft, firstDisabled, page - 1)}
        {measuring ? <div className="pagination__page">{pageContents}</div> :
          <form className="pagination__page" onSubmit={event => {
            event.preventDefault();
            const input = event.currentTarget.querySelector("input")!;
            const target = Number(draft);
            if (!/^\d+$/.test(draft) || !Number.isSafeInteger(target) || target < 1 || target > maximumPage) {
              input.setCustomValidity(t("pagination.invalidPage", { max: number(maximumPage) }));
              input.reportValidity();
              return;
            }
            navigate(target);
          }}>{pageContents}</form>}
        {arrow("next", faAngleRight, nextDisabled, page + 1)}
        {arrow("last", faAnglesRight, lastDisabled, totalPages ?? page)}
      </div>
      <span className="pagination__count" title={measuring ? undefined : countLabel}>
        <span className="pagination__full-count">{countLabel}</span>
        <span className="pagination__short-count">{totalCount === null ? "—" : t(totalPages === null ? "pagination.loadedItems" : "pagination.itemsCompact", { total: number(totalCount) })}</span>
      </span>
      <div className="pagination__controls">
        <span className="pagination__size-slot">{renderSize(measuring)}</span>
        {measuring && sizeHidden ? <span className="ghost-button pagination__button pagination__more"><FontAwesomeIcon icon={faEllipsis} /></span> : !measuring ? <button ref={more} type="button" className="ghost-button pagination__button pagination__more" aria-label={t("pagination.options")}
          title={t("pagination.options")} aria-haspopup="dialog" aria-expanded={optionsOpen} aria-controls={optionsOpen ? id : undefined}
          onClick={() => setOptionsOpen(open => !open)}><FontAwesomeIcon icon={faEllipsis} aria-hidden="true" /></button> : null}
      </div>
    </>;
  };

  return <nav className={`pagination${layout === "compact" ? " is-compact" : ""}${layout !== "full" ? " is-size-hidden" : ""}`} aria-label={t("pagination.navigation", { label, position: t(`pagination.${position}`) })} aria-busy={loading}>
    <div ref={viewport} className="pagination__viewport"><div className={`pagination__row${layout !== "full" ? " pagination__row--size-hidden" : ""}${layout === "compact" ? " pagination__row--compact" : ""}`}>{renderRow(false)}</div></div>
    <span className="pagination__announcement" role="status" aria-live={position === "top" ? "polite" : "off"}>{t("pagination.status", { page: totalCount === null ? "—" : number(displayPage), pages: pagesText, total: countLabel })}</span>
    <div className="pagination__measurement" aria-hidden="true" inert><div ref={measure} className="pagination__row pagination__row--measure">{renderRow(true)}</div><div ref={sizeHiddenMeasure} className="pagination__row pagination__row--measure pagination__row--size-hidden">{renderRow(true, true)}</div></div>
    {optionsOpen && layout !== "full" ? <PaginationOptions id={id} showEdges={layout === "compact"} anchor={more} pageSize={pageSize} disabled={disabled} firstDisabled={firstDisabled} lastDisabled={lastDisabled}
      onPageSizeChange={onPageSizeChange} onFirst={() => navigate(1)} onLast={() => navigate(totalPages ?? page)} onClose={closeOptions} /> : null}
  </nav>;
}
