import { useEffect, useState } from "react";
import { endpoints, errorMessage } from "../api/client";
import { useAuth } from "../context/AuthContext.jsx";
import { useToast } from "../context/ToastContext.jsx";
import StarRating from "./StarRating.jsx";

/** Lets each side of a completed booking review the other, exactly once. */
export default function ReviewForm({ bookingId, counterpartName }) {
  const { user } = useAuth();
  const toast = useToast();
  const [reviews, setReviews] = useState([]);
  const [rating, setRating] = useState(5);
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let active = true;
    endpoints
      .bookingReviews(bookingId)
      .then(({ data }) => active && setReviews(data || []))
      .catch(() => active && setReviews([]));
    return () => { active = false; };
  }, [bookingId]);

  const mine = reviews.find((review) => review.reviewer_id === user?.id);
  const received = reviews.find((review) => review.reviewee_id === user?.id);

  async function submit(event) {
    event.preventDefault();
    setBusy(true);
    try {
      const { data } = await endpoints.createReview(bookingId, { rating, comment });
      setReviews((current) => [...current, data]);
      toast.success("Thanks for your review!");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="card space-y-4">
      <h3 className="font-display text-xl font-bold text-brand-dark">Reviews</h3>

      {mine ? (
        <div className="rounded-2xl bg-brand-cream p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-brand-muted">Your review</p>
          <div className="mt-2"><StarRating value={mine.rating} /></div>
          {mine.comment && <p className="mt-2 text-sm text-brand-charcoal">{mine.comment}</p>}
        </div>
      ) : (
        <form onSubmit={submit} className="space-y-3">
          <p className="text-sm text-brand-muted">
            How did it go with {counterpartName || "the other party"}?
          </p>
          <StarRating value={rating} onChange={setRating} />
          <textarea
            className="input min-h-24"
            value={comment}
            onChange={(event) => setComment(event.target.value)}
            placeholder="Share a little detail (optional)"
            aria-label="Review comment"
          />
          <button disabled={busy} className="btn-primary w-full py-3 text-base disabled:opacity-50">
            Submit review
          </button>
        </form>
      )}

      {received && (
        <div className="border-t border-brand-border pt-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-brand-muted">
            Review you received
          </p>
          <div className="mt-2"><StarRating value={received.rating} /></div>
          {received.comment && <p className="mt-2 text-sm text-brand-charcoal">{received.comment}</p>}
        </div>
      )}
    </div>
  );
}
