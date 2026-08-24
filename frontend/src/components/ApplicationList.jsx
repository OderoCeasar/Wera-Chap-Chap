import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { endpoints, errorMessage } from "../api/client";
import { useToast } from "../context/ToastContext.jsx";
import StarRating from "./StarRating.jsx";
import StatusBadge from "./StatusBadge.jsx";
import { kes } from "../utils/money";

/** The applications a client received for one task, with accept/reject actions. */
export default function ApplicationList({ task, onChanged }) {
  const toast = useToast();
  const navigate = useNavigate();
  const [busyId, setBusyId] = useState(null);

  const applications = task.applications || [];
  if (applications.length === 0) {
    return (
      <p className="rounded-2xl bg-brand-cream p-4 text-sm text-brand-muted">
        No applications yet. Taskers matching this category will see it in their feed.
      </p>
    );
  }

  async function decide(application, accept) {
    setBusyId(application.id);
    try {
      const { data } = accept
        ? await endpoints.acceptApplication(task.id, application.id)
        : await endpoints.rejectApplication(task.id, application.id);
      toast.success(accept ? "Tasker booked!" : "Application rejected.");
      await onChanged?.();
      if (accept && data.booking?.id) navigate(`/bookings/${data.booking.id}`);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setBusyId(null);
    }
  }

  return (
    <ul className="space-y-3">
      {applications.map((application) => {
        const tasker = application.tasker || {};
        const pending = application.status === "pending";
        return (
          <li key={application.id} className="rounded-2xl border border-brand-border bg-brand-cream p-4">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <p className="font-body font-bold text-brand-dark">
                  {tasker.user?.full_name || "Tasker"}
                </p>
                <div className="mt-1 flex items-center gap-2">
                  <StarRating value={tasker.avg_rating} />
                  <span className="text-xs text-brand-muted">({tasker.total_reviews || 0})</span>
                </div>
              </div>
              <div className="text-right">
                <p className="font-display text-lg font-black text-brand-primary">
                  {kes(application.proposed_rate)}
                </p>
                <StatusBadge status={application.status} />
              </div>
            </div>

            {application.cover_note && (
              <p className="mt-3 text-sm text-brand-charcoal">{application.cover_note}</p>
            )}

            {pending && task.status === "open" && (
              <div className="mt-4 flex gap-2">
                <button
                  disabled={busyId === application.id}
                  onClick={() => decide(application, true)}
                  className="btn-dark flex-1 disabled:opacity-50"
                >
                  Accept &amp; book
                </button>
                <button
                  disabled={busyId === application.id}
                  onClick={() => decide(application, false)}
                  className="flex-1 rounded-xl border-2 border-brand-border px-4 py-2 text-sm font-semibold text-brand-muted transition-colors hover:border-red-300 hover:text-red-600 disabled:opacity-50"
                >
                  Reject
                </button>
              </div>
            )}
          </li>
        );
      })}
    </ul>
  );
}
