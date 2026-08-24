import { Link } from "react-router-dom";
import StarRating from "./StarRating.jsx";
import { kes } from "../utils/money";

const FACTORS = [
  ["skill_match", "Skill"],
  ["availability_match", "Available"],
  ["distance_score", "Distance"],
  ["rating_score", "Rating"],
  ["price_match", "Price"]
];

export default function SmartMatchList({ matches = [] }) {
  if (matches.length === 0) {
    return <p className="text-sm text-brand-muted">No taskers to rank yet.</p>;
  }

  return (
    <div className="space-y-3">
      {matches.map((match) => (
        <div key={match.tasker.id} className="rounded-2xl border border-brand-border bg-brand-cream p-4">
          <div className="flex items-start justify-between gap-4">
            <div>
              <Link
                to={`/taskers/${match.tasker.id}`}
                className="font-body font-bold text-brand-dark hover:text-brand-primary"
              >
                {match.tasker.user?.full_name || "Tasker"}
              </Link>
              <div className="mt-1 flex items-center gap-2">
                <StarRating value={match.tasker.avg_rating} />
                <span className="text-xs text-brand-muted">
                  {kes(match.tasker.hourly_rate)}/hr
                </span>
              </div>
            </div>
            <div className="text-right">
              <p className="font-display text-2xl font-black text-brand-primary">
                {Math.round(match.score * 100)}%
              </p>
              <p className="text-xs font-semibold uppercase text-brand-muted">match score</p>
            </div>
          </div>

          {/* Why this tasker ranked where they did. */}
          <ul className="mt-3 flex flex-wrap gap-1.5">
            {FACTORS.map(([key, label]) => (
              <li
                key={key}
                className={`rounded-full px-2 py-1 text-xs font-semibold ${
                  match[key] >= 0.75
                    ? "bg-green-50 text-green-700"
                    : match[key] >= 0.4
                      ? "bg-amber-50 text-amber-700"
                      : "bg-brand-border/50 text-brand-muted"
                }`}
              >
                {label} {Math.round((match[key] || 0) * 100)}%
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}
