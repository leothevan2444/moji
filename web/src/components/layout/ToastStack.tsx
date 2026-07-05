import type { ToastItem } from "../../types";
import { TOAST_LIFETIME_MS } from "../../constants";

interface ToastStackProps {
  toasts: ToastItem[];
  onDismiss: (id: number) => void;
}

export function ToastStack({ toasts, onDismiss }: ToastStackProps) {
  return (
    <div className="toast-stack" aria-live="polite" aria-atomic="false">
      {toasts.map((toast) => {
        const lifetime = toast.lifetimeMs ?? TOAST_LIFETIME_MS;
        return (
          <div
            key={toast.id}
            className={`toast-card ${toast.tone} toast-card--${toast.phase}`}
            role="status"
            style={{ "--toast-progress-duration": `${lifetime}ms` } as React.CSSProperties}
          >
            <div className="toast-card__body">
              <span className="toast-card__label">
                {toast.tone === "tone-success" ? "成功" : toast.tone === "tone-danger" ? "错误" : "提示"}
              </span>
              <p>{toast.message}</p>
            </div>
            <button type="button" className="toast-card__close" onClick={() => onDismiss(toast.id)} aria-label="关闭消息">
              ×
            </button>
            <div className="toast-card__progress" aria-hidden="true" />
          </div>
        );
      })}
    </div>
  );
}
