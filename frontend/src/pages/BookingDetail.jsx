import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ArrowLeft, CalendarClock, MapPin } from "lucide-react";
import { endpoints, errorMessage } from "../api/client";
import ChatWindow from "../components/ChatWindow.jsx";
import PaymentPanel from "../components/PaymentPanel.jsx";
import ReviewForm from "../components/ReviewForm.jsx";
import StatusBadge from "../components/StatusBadge.jsx";
import { useAuth } from "../context/AuthContext.jsx";
import { useToast } from "../context/ToastContext.jsx";
import { kes } from "../utils/money";

const STAGES = ["confirmed", "started", "completed"];

export default function BookingDetail() {
  const { id } = useParams();
  const { user } = useAuth();
  const toast = useToast();
  const [booking, setBooking] = useState(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const { data } = await endpoints.booking(id);
      setBooking(data);
      setError("");
    } catch (err) {
      setError(errorMessage(err, "This booking is not available."));
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  if (loading) return <main className="p-16 text-center text-brand-muted">Loading booking...</main>;
  if (error || !booking) {
    return (
      <main className="mx-auto max-w-2xl px-4 py-20 text-center">
        <h1 className="font-display text-3xl font-black text-brand-dark">Booking unavailable</h1>
        <p className="mt-3 text-brand-muted">{error}</p>
        <Link to="/bookings" className="btn-primary mt-8 inline-block">Back to bookings</Link>
      </main>
    );
  }

  const isClient = booking.client_id === user?.id;
  const task = booking.task || {};
  const counterpart = isClient ? booking.tasker?.user?.full_name : booking.client?.full_name;
  const stageIndex = STAGES.indexOf(booking.status);
  const cancelled = booking.status === "cancelled";

  async function act(action, message) {
    setBusy(true);
    try {
      await action();
      toast.success(message);
      await load();
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto max-w-6xl px-4 py-10">
      <Link to="/bookings" className="mb-6 inline-flex items-center gap-2 text-sm font-semibold text-brand-muted hover:text-brand-dark">
        <ArrowLeft size={16} /> All bookings
      </Link>

      <header className="mb-8 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="section-label">{task.category?.icon_url} {task.category?.name}</p>
          <h1 className="mt-2 font-display text-4xl font-black text-brand-dark">
            {task.title || `Booking #${booking.id}`}
          </h1>
          <div className="mt-3 flex flex-wrap gap-4 text-sm text-brand-muted">
            <span className="flex items-center gap-1.5"><MapPin size={14} /> {task.location_address || "Kenya"}</span>
            {task.scheduled_at && (
              <span className="flex items-center gap-1.5">
                <CalendarClock size={14} /> {new Date(task.scheduled_at).toLocaleString()}
              </span>
            )}
          </div>
        </div>
        <div className="text-right">
          <StatusBadge status={booking.status} />
          <p className="mt-2 font-display text-3xl font-black text-brand-primary">{kes(booking.agreed_rate)}</p>
          <p className="text-xs font-semibold uppercase tracking-wide text-brand-muted">
            with {counterpart || "—"}
          </p>
        </div>
      </header>

      {/* Lifecycle progress */}
      <section className="card mb-8">
        {cancelled ? (
          <p className="text-center font-semibold text-red-600">This booking was cancelled.</p>
        ) : (
          <ol className="flex items-center gap-2">
            {STAGES.map((stage, index) => (
              <li key={stage} className="flex flex-1 items-center gap-2">
                <div className="flex-1">
                  <div className={`h-2 rounded-full ${index <= stageIndex ? "bg-brand-primary" : "bg-brand-border"}`} />
                  <p className={`mt-2 text-xs font-semibold uppercase tracking-wide ${index <= stageIndex ? "text-brand-dark" : "text-brand-muted"}`}>
                    {stage}
                  </p>
                </div>
              </li>
            ))}
          </ol>
        )}

        <div className="mt-6 flex flex-wrap gap-3">
          {!isClient && booking.status === "confirmed" && (
            <button disabled={busy} onClick={() => act(() => endpoints.startBooking(booking.id), "Booking started.")} className="btn-primary py-3 text-base disabled:opacity-50">
              Start work
            </button>
          )}
          {!isClient && booking.status === "started" && (
            <button disabled={busy} onClick={() => act(() => endpoints.completeBooking(booking.id), "Booking completed.")} className="btn-primary py-3 text-base disabled:opacity-50">
              Mark complete
            </button>
          )}
          {["confirmed", "started"].includes(booking.status) && (
            <button
              disabled={busy}
              onClick={() => act(() => endpoints.cancelBooking(booking.id), "Booking cancelled.")}
              className="rounded-2xl border-2 border-brand-border px-6 py-3 font-semibold text-brand-muted transition-colors hover:border-red-300 hover:text-red-600 disabled:opacity-50"
            >
              Cancel booking
            </button>
          )}
          {isClient && booking.status === "confirmed" && (
            <p className="self-center text-sm text-brand-muted">
              Waiting for {counterpart || "your tasker"} to start the job.
            </p>
          )}
        </div>
      </section>

      <div className="grid gap-8 lg:grid-cols-[1.2fr_.8fr]">
        <ChatWindow bookingId={booking.id} />
        <div className="space-y-6">
          <PaymentPanel booking={booking} isClient={isClient} />
          {booking.status === "completed" && (
            <ReviewForm bookingId={booking.id} counterpartName={counterpart} />
          )}
        </div>
      </div>
    </main>
  );
}
