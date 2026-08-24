import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { BadgeCheck, CalendarDays } from "lucide-react";
import { endpoints, errorMessage } from "../api/client";
import StarRating from "../components/StarRating.jsx";
import { useAuth } from "../context/AuthContext.jsx";
import { kes } from "../utils/money";

const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

export default function TaskerProfile() {
  const { id } = useParams();
  const { user } = useAuth();
  const navigate = useNavigate();
  const [tasker, setTasker] = useState(null);
  const [reviews, setReviews] = useState([]);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    endpoints
      .tasker(id)
      .then(({ data }) => active && setTasker(data))
      .catch((err) => active && setError(errorMessage(err, "Tasker not found.")));
    endpoints
      .taskerReviews(id)
      .then(({ data }) => active && setReviews(data || []))
      .catch(() => active && setReviews([]));
    return () => { active = false; };
  }, [id]);

  if (error) {
    return (
      <main className="mx-auto max-w-2xl px-4 py-24 text-center">
        <h1 className="font-display text-3xl font-black text-brand-dark">{error}</h1>
        <Link to="/taskers" className="btn-primary mt-8 inline-block">Browse other Taskers</Link>
      </main>
    );
  }
  if (!tasker) return <main className="p-12 text-center text-brand-muted">Loading...</main>;

  // "Hire me" posts a task pre-filled with this tasker's main skill.
  function hire() {
    if (!user) return navigate("/login", { state: { from: `/taskers/${id}` } });
    if (user.role !== "client") return navigate("/dashboard/tasker");
    navigate("/dashboard/client", {
      state: {
        prefill: {
          category_id: tasker.skills?.[0]?.category_id ? String(tasker.skills[0].category_id) : "",
          budget_amount: tasker.hourly_rate ? String(tasker.hourly_rate) : "",
          budget_type: "hourly"
        }
      }
    });
  }

  return (
    <main className="mx-auto grid max-w-6xl gap-8 px-4 py-12 lg:grid-cols-[.8fr_1.2fr]">
      <aside className="card h-fit text-center">
        <img
          className="mx-auto h-32 w-32 rounded-3xl object-cover ring-4 ring-brand-border"
          src={tasker.user?.avatar_url || `https://api.dicebear.com/8.x/initials/svg?seed=${encodeURIComponent(tasker.user?.full_name || "Tasker")}`}
          alt=""
        />
        <h1 className="mt-5 font-display text-3xl font-black text-brand-dark">{tasker.user?.full_name}</h1>

        {tasker.user?.background_check_passed && (
          <p className="mt-2 inline-flex items-center gap-1.5 rounded-full bg-green-50 px-3 py-1 text-xs font-semibold text-green-700">
            <BadgeCheck size={14} /> Background checked
          </p>
        )}

        <div className="mt-3 flex justify-center"><StarRating value={tasker.avg_rating} /></div>
        <p className="mt-1 text-sm text-brand-muted">
          {Number(tasker.avg_rating || 0).toFixed(1)} from {tasker.total_reviews || 0} review
          {tasker.total_reviews === 1 ? "" : "s"}
        </p>

        <p className="mt-4 font-display text-2xl font-black text-brand-primary">
          {kes(tasker.hourly_rate)}<span className="font-body text-sm text-brand-muted">/hr</span>
        </p>
        <p className="mt-1 text-sm text-brand-muted">
          {tasker.years_experience || 0} yrs experience · {tasker.service_radius_km || 0} km radius
        </p>

        <button onClick={hire} className="btn-primary mt-6 w-full">
          {user?.role === "tasker" ? "Back to dashboard" : "Hire Me"}
        </button>
        {!tasker.is_available && (
          <p className="mt-3 text-sm font-semibold text-brand-muted">Currently not accepting work.</p>
        )}
      </aside>

      <section className="space-y-6">
        <div className="card">
          <h2 className="font-display text-2xl font-black text-brand-dark">About</h2>
          <p className="mt-4 leading-relaxed text-brand-charcoal">
            {tasker.bio || "Experienced, punctual and ready to help."}
          </p>
        </div>

        <div className="card">
          <h2 className="font-display text-2xl font-black text-brand-dark">Skills</h2>
          <div className="mt-4 flex flex-wrap gap-2">
            {(tasker.skills || []).length === 0 && (
              <p className="text-sm text-brand-muted">No skills listed yet.</p>
            )}
            {tasker.skills?.map((skill) => (
              <span key={skill.id} className="rounded-full bg-brand-cream px-3 py-1 font-body text-xs font-medium text-brand-charcoal">
                {skill.category?.name}
              </span>
            ))}
          </div>
        </div>

        {tasker.availability?.length > 0 && (
          <div className="card">
            <h2 className="flex items-center gap-2 font-display text-2xl font-black text-brand-dark">
              <CalendarDays size={20} className="text-brand-primary" /> Availability
            </h2>
            <ul className="mt-4 grid gap-2 sm:grid-cols-2">
              {tasker.availability.map((slot) => (
                <li key={slot.id} className="flex justify-between rounded-2xl bg-brand-cream px-4 py-2 text-sm">
                  <span className="font-semibold text-brand-dark">{DAYS[slot.day_of_week]}</span>
                  <span className="text-brand-muted">
                    {slot.start_time.slice(0, 5)} – {slot.end_time.slice(0, 5)}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}

        <div className="card">
          <h2 className="font-display text-2xl font-black text-brand-dark">Reviews</h2>
          {reviews.length === 0 ? (
            <p className="mt-4 text-sm text-brand-muted">No reviews yet.</p>
          ) : (
            <div className="mt-4 space-y-4">
              {reviews.map((review) => (
                <div className="border-t border-brand-border pt-4" key={review.id}>
                  <div className="flex items-start justify-between">
                    <div>
                      <StarRating value={review.rating} />
                      <p className="mt-1 text-sm font-semibold text-brand-dark">
                        {review.reviewer?.full_name || "Client"}
                      </p>
                    </div>
                    <p className="text-xs text-brand-muted">
                      {new Date(review.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  {review.comment && <p className="mt-2 text-brand-charcoal">{review.comment}</p>}
                </div>
              ))}
            </div>
          )}
        </div>
      </section>
    </main>
  );
}
