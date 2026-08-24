import { useEffect, useState } from "react";
import { endpoints, errorMessage } from "../api/client";
import BookingCard from "../components/BookingCard.jsx";
import { useAuth } from "../context/AuthContext.jsx";
import { useToast } from "../context/ToastContext.jsx";

const FILTERS = ["all", "confirmed", "started", "completed", "cancelled"];

export default function Bookings() {
  const { user } = useAuth();
  const toast = useToast();
  const [bookings, setBookings] = useState([]);
  const [filter, setFilter] = useState("all");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    endpoints
      .bookings()
      .then(({ data }) => active && setBookings(data || []))
      .catch((error) => active && toast.error(errorMessage(error, "Could not load bookings.")))
      .finally(() => active && setLoading(false));
    return () => { active = false; };
  }, [toast]);

  const visible = filter === "all" ? bookings : bookings.filter((booking) => booking.status === filter);

  return (
    <main className="mx-auto max-w-5xl px-4 py-12">
      <div className="mb-8 space-y-2">
        <p className="section-label">Your work</p>
        <h1 className="font-display text-5xl font-black text-brand-dark">Bookings</h1>
      </div>

      <div className="mb-8 flex flex-wrap gap-2">
        {FILTERS.map((option) => (
          <button
            key={option}
            onClick={() => setFilter(option)}
            className={`rounded-full px-4 py-2 text-sm font-semibold capitalize transition-colors ${
              filter === option ? "bg-brand-primary text-white" : "bg-brand-surface text-brand-charcoal border border-brand-border"
            }`}
          >
            {option}
          </button>
        ))}
      </div>

      {loading ? (
        <p className="py-16 text-center text-brand-muted">Loading bookings...</p>
      ) : visible.length === 0 ? (
        <div className="card text-center">
          <p className="text-brand-muted">
            {bookings.length === 0
              ? "No bookings yet. Accepted applications turn into bookings here."
              : `No ${filter} bookings.`}
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {visible.map((booking) => (
            <BookingCard key={booking.id} booking={booking} perspective={user?.role} />
          ))}
        </div>
      )}
    </main>
  );
}
