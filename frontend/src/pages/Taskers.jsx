import { useEffect, useState } from "react";
import { endpoints, errorMessage } from "../api/client";
import TaskerCard from "../components/TaskerCard.jsx";
import { useToast } from "../context/ToastContext.jsx";

export default function Taskers() {
  const toast = useToast();
  const [taskers, setTaskers] = useState([]);
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [filters, setFilters] = useState({ q: "", category_id: "", max_rate: "", min_rating: "" });

  useEffect(() => {
    endpoints.categories().then(({ data }) => setCategories(data || [])).catch(() => {});
  }, []);

  // Debounced so typing in the search box does not fire a request per keystroke.
  useEffect(() => {
    let active = true;
    setLoading(true);
    const timer = setTimeout(() => {
      const params = Object.fromEntries(Object.entries(filters).filter(([, value]) => value !== ""));
      endpoints
        .taskers(params)
        .then(({ data }) => active && setTaskers(data || []))
        .catch((error) => {
          if (!active) return;
          setTaskers([]);
          toast.error(errorMessage(error, "Could not load taskers."));
        })
        .finally(() => active && setLoading(false));
    }, 300);
    return () => { active = false; clearTimeout(timer); };
  }, [filters, toast]);

  return (
    <main className="mx-auto max-w-7xl px-4 py-12">
      <div className="mb-8 space-y-2">
        <p className="section-label">Verified help</p>
        <h1 className="font-display text-5xl font-black text-brand-dark md:text-6xl">Browse Taskers</h1>
      </div>

      <div className="mb-10 grid gap-3 rounded-3xl border border-brand-border bg-brand-surface p-4 sm:grid-cols-2 lg:grid-cols-4">
        <input
          className="input" placeholder="Search name or bio" value={filters.q} aria-label="Search"
          onChange={(event) => setFilters({ ...filters, q: event.target.value })}
        />
        <select
          className="input" value={filters.category_id} aria-label="Category"
          onChange={(event) => setFilters({ ...filters, category_id: event.target.value })}
        >
          <option value="">All categories</option>
          {categories.map((category) => (
            <option key={category.id} value={category.id}>{category.name}</option>
          ))}
        </select>
        <input
          className="input" type="number" min="0" placeholder="Max rate (KES)" value={filters.max_rate} aria-label="Maximum hourly rate"
          onChange={(event) => setFilters({ ...filters, max_rate: event.target.value })}
        />
        <select
          className="input" value={filters.min_rating} aria-label="Minimum rating"
          onChange={(event) => setFilters({ ...filters, min_rating: event.target.value })}
        >
          <option value="">Any rating</option>
          <option value="3">3+ stars</option>
          <option value="4">4+ stars</option>
          <option value="4.5">4.5+ stars</option>
        </select>
      </div>

      {loading ? (
        <p className="py-16 text-center text-brand-muted">Finding taskers...</p>
      ) : taskers.length === 0 ? (
        <div className="card text-center">
          <p className="text-brand-muted">No taskers match those filters yet. Try widening your search.</p>
        </div>
      ) : (
        <>
          <p className="mb-5 text-sm font-semibold text-brand-muted">
            {taskers.length} tasker{taskers.length === 1 ? "" : "s"} available
          </p>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
            {taskers.map((tasker) => <TaskerCard key={tasker.id} tasker={tasker} />)}
          </div>
        </>
      )}
    </main>
  );
}
