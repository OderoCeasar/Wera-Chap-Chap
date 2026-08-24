import { Link } from "react-router-dom";
import { CalendarClock, MapPin } from "lucide-react";
import StatusBadge from "./StatusBadge.jsx";
import { kes } from "../utils/money";

export default function BookingCard({ booking, perspective }) {
  const task = booking.task || {};
  const counterpart =
    perspective === "tasker" ? booking.client?.full_name : booking.tasker?.user?.full_name;

  return (
    <article className="rounded-3xl border border-brand-border bg-brand-surface p-6 shadow-sm transition-all hover:shadow-md">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-sm font-semibold text-brand-primary">
            {task.category?.icon_url} {task.category?.name}
          </p>
          <h3 className="mt-1 font-display text-lg font-bold text-brand-dark">
            {task.title || `Booking #${booking.id}`}
          </h3>
        </div>
        <StatusBadge status={booking.status} />
      </div>

      <div className="mt-4 flex flex-wrap gap-4 text-sm text-brand-muted">
        <span className="flex items-center gap-1.5">
          <MapPin size={14} /> {task.location_address || "Kenya"}
        </span>
        {task.scheduled_at && (
          <span className="flex items-center gap-1.5">
            <CalendarClock size={14} /> {new Date(task.scheduled_at).toLocaleString()}
          </span>
        )}
      </div>

      <div className="mt-4 flex items-center justify-between border-t border-brand-border pt-4">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-brand-muted">
            {perspective === "tasker" ? "Client" : "Tasker"}
          </p>
          <p className="font-body font-semibold text-brand-dark">{counterpart || "—"}</p>
        </div>
        <div className="text-right">
          <p className="font-display text-xl font-black text-brand-primary">{kes(booking.agreed_rate)}</p>
          <Link to={`/bookings/${booking.id}`} className="text-sm font-semibold text-brand-dark hover:text-brand-primary">
            Open booking →
          </Link>
        </div>
      </div>
    </article>
  );
}
