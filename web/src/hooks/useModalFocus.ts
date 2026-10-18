import { useEffect, useRef, type RefObject } from "react";

const modalStack: HTMLElement[] = [];
let previousOverflow = "";
const focusSelector = 'button:not(:disabled), a[href], input:not(:disabled):not([type="hidden"]), select:not(:disabled), textarea:not(:disabled), summary, [tabindex]:not([tabindex="-1"])';

function focusableElements(panel: HTMLElement) {
  return Array.from(panel.querySelectorAll<HTMLElement>(focusSelector)).filter(element => {
    if (element.closest("[hidden], [inert]") || (element.hasAttribute("tabindex") && element.tabIndex < 0)) return false;
    let ancestor: HTMLElement | null = element;
    while (ancestor && ancestor !== panel) {
      const style = getComputedStyle(ancestor);
      if (style.display === "none" || style.visibility === "hidden") return false;
      if (ancestor instanceof HTMLDetailsElement && !ancestor.open) {
        const summary = ancestor.querySelector(":scope > summary");
        if (!summary?.contains(element)) return false;
      }
      ancestor = ancestor.parentElement;
    }
    return true;
  }).sort((a, b) => {
    const aOrder = a.tabIndex > 0 ? a.tabIndex : Infinity;
    const bOrder = b.tabIndex > 0 ? b.tabIndex : Infinity;
    if (aOrder !== bOrder) return aOrder - bOrder;
    return a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING ? -1 : 1;
  });
}

/** Keep keyboard focus inside the topmost modal and restore its opener on close. */
export function useModalFocus(ref: RefObject<HTMLElement | null>, onClose: () => void) {
  const closeRef = useRef(onClose);
  closeRef.current = onClose;
  useEffect(() => {
    const panel = ref.current;
    if (!panel) return;
    const opener = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const taskId = opener?.dataset.taskId;
    if (!modalStack.length) {
      previousOverflow = document.body.style.overflow;
      document.body.style.overflow = "hidden";
    }
    const parent = modalStack.at(-1);
    if (parent) parent.inert = true;
    modalStack.push(panel);
    const focusFirst = () => (focusableElements(panel)[0] ?? panel).focus();
    focusFirst();
    const onKeyDown = (event: KeyboardEvent) => {
      if (modalStack.at(-1) !== panel || event.defaultPrevented) return;
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopImmediatePropagation();
        closeRef.current();
      } else if (event.key === "Tab") {
        const elements = focusableElements(panel);
        event.preventDefault();
        if (!elements.length) panel.focus();
        else {
          const current = elements.indexOf(document.activeElement as HTMLElement);
          const next = current < 0 ? (event.shiftKey ? elements.length - 1 : 0)
            : (current + (event.shiftKey ? -1 : 1) + elements.length) % elements.length;
          elements[next].focus();
        }
      }
    };
    const onFocus = (event: FocusEvent) => {
      if (modalStack.at(-1) === panel && event.target instanceof Node && !panel.contains(event.target)) focusFirst();
    };
    document.addEventListener("keydown", onKeyDown);
    document.addEventListener("focusin", onFocus);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.removeEventListener("focusin", onFocus);
      const wasTop = modalStack.at(-1) === panel;
      const index = modalStack.indexOf(panel);
      if (index >= 0) modalStack.splice(index, 1);
      const remaining = modalStack.at(-1);
      if (remaining) remaining.inert = false;
      if (!modalStack.length) document.body.style.overflow = previousOverflow;
      if (wasTop) {
        const restored = opener?.isConnected ? opener : taskId
          ? Array.from(document.querySelectorAll<HTMLElement>("[data-task-id]")).find(element => element.dataset.taskId === taskId) : null;
        if (restored && (!remaining || remaining.contains(restored))) restored.focus();
        else if (remaining) (focusableElements(remaining)[0] ?? remaining).focus();
      }
    };
  }, [ref]);
}
