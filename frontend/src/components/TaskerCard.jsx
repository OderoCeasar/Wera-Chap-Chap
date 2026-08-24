import { Link } from "react-router-dom";
import { BadgeCheck } from "lucide-react";
import StarRating from "./StarRating.jsx";
import { kes } from "../utils/money";

export default function TaskerCard({ tasker }) {
  return (
    <article className="flex flex-col rounded-3xl border border-brand-border bg-brand-surface p-5 shadow-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-lg hover:shadow-brand-dark/10">
      <div className="flex items-center gap-4">
        <img
          className="h-14 w-14 rounded-2xl object-cover ring-2 ring-brand-border"
          src={tasker.user?.avatar_url || `https://api.dicebear.com/8.x/initials/svg?seed=${encodeURIComponent(tasker.user?.full_name || "Tasker")}`}
          alt=""
        />
        <div>
          <h3 className="flex items-center gap-1.5 font-body font-bold text-brand-dark">
            {tasker.user?.full_name || "Verified Tasker"}
            {tasker.user?.background_check_passed && (
              <BadgeCheck size={16} className="text-green-600" aria-label="Background checked" />
            )}
          </h3>
          <div className="flex items-center gap-2 text-sm">
            <StarRating value={tasker.avg_rating} />
            <span className="text-brand-muted">({tasker.total_reviews || 0})</span>
          </div>
        </div>
      </div>

      <p className="mt-4 line-clamp-3 flex-1 text-brand-charcoal">
        {tasker.bio || "Friendly local pro ready to help with tasks across Nairobi and beyond."}
      </p>

      <div className="mt-4 flex flex-wrap gap-2">
        {tasker.skills?.slice(0, 3).map((skill) => (
          <span key={skill.id} className="rounded-full bg-brand-cream px-2 py-1 text-xs font-medium text-brand-charcoal">
            {skill.category?.name}
          </span>
        ))}
      </div>

      <div className="mt-4 flex items-center justify-between">
        <p className="font-display text-xl font-bold text-brand-primary">
          {kes(tasker.hourly_rate)}<span className="font-body text-sm text-brand-muted">/hr</span>
        </p>
        <Link className="btn-dark" to={`/taskers/${tasker.id}`}>View Profile</Link>
      </div>
    </article>
  );
}
