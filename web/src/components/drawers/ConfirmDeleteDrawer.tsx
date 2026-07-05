interface ConfirmDeleteDrawerProps {
  taskLabel: string;
  destructive: boolean;
  pending: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDeleteDrawer({
  taskLabel,
  destructive,
  pending,
  onConfirm,
  onCancel
}: ConfirmDeleteDrawerProps) {
  return (
    <div className="drawer-stack">
      <article className="drawer-card">
        <div className="drawer-card__head">
          <div>
            <h3>确认删除任务</h3>
            <p>{taskLabel}</p>
          </div>
        </div>
        <div className="settings-actions">
          <button
            type="button"
            className="ghost-button"
            onClick={onCancel}
            disabled={pending}
          >
            取消
          </button>
          <button
            type="button"
            className="task-ops__button task-ops__button--danger"
            onClick={onConfirm}
            disabled={pending}
          >
            {pending ? "删除中..." : "确认删除"}
          </button>
        </div>
      </article>
    </div>
  );
}
