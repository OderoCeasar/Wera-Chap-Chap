import StatusBadge from "./StatusBadge.jsx";
import { kes } from "../utils/money";

/**
 * Shared task card. `extraMeta` adds role-specific facts to the meta row,
 * `action` renders the primary control, and `children` holds expanded content.
 */
export default function TaskCard({ task, extraMeta = [], action, children }) {
  const meta = [
    `📍 ${task.location_address || "Kenya"}`,
    `💰 ${kes(task.budget_amount)} ${task.budget_type || ""}`.trim(),
    ...extraMeta
  ];

  return (
    <article className="rounded-3xl border border-brand-border border-l-4 border-l-brand-primary bg-brand-surface p-6 shadow-sm">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-sm font-semibold text-brand-primary">
            {task.category?.icon_url} {task.category?.name}
          </p>
          <h3 className="mt-1 font-display text-lg font-bold text-brand-dark">{task.title}</h3>
        </div>
        <StatusBadge status={task.status || "open"} />
      </div>

      {task.description && <p className="mt-3 line-clamp-2 text-brand-charcoal">{task.description}</p>}

      <div className="mt-4 flex flex-wrap gap-4 text-sm font-semibold text-brand-muted">
        {meta.map((item) => <span key={item}>{item}</span>)}
      </div>

      {action && <div className="mt-4">{action}</div>}
      {children}
    </article>
  );
}
