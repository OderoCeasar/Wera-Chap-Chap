import { Star } from "lucide-react";

export default function StarRating({ value = 0, onChange }) {
  return (
    <div className="flex gap-1">
      {[1, 2, 3, 4, 5].map((star) => (
        <button key={star} type="button" onClick={() => onChange?.(star)} className={onChange ? "cursor-pointer" : "cursor-default"}>
          <Star size={18} className={star <= Math.round(value) ? "fill-brand-secondary text-brand-secondary" : "text-brand-border"} />
        </button>
      ))}
    </div>
  );
}
