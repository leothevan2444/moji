import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import { Drawer } from "../layout/Drawer";
import "../../styles/confirmation-dialog.scss";

interface ConfirmationDialogProps {
  title: string;
  children: ReactNode;
  confirmLabel: string;
  cancelLabel: string;
  pending: boolean;
  error?: string | null;
  onConfirm: () => void;
  onCancel: () => void;
}

/** A shared confirmation surface; the caller owns the operation and its result. */
export function ConfirmationDialog({ title, children, confirmLabel, cancelLabel, pending, error, onConfirm, onCancel }: ConfirmationDialogProps) {
  const dismiss = () => { if (!pending) onCancel(); };
  return createPortal(<Drawer visibleDrawer="confirm" closing={false} title={title} onClose={dismiss} closeDisabled={pending}
    footer={<div className="confirmation-actions">
      <button type="button" className="ghost-button" disabled={pending} onClick={dismiss}>{cancelLabel}</button>
      <button type="button" className="confirmation-actions__confirm" disabled={pending} aria-busy={pending} onClick={onConfirm}>{confirmLabel}</button>
    </div>}>
    {children}
    {error ? <p className="inline-error" role="alert">{error}</p> : null}
  </Drawer>, document.body);
}
