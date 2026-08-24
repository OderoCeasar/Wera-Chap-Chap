import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { Sparkles } from "lucide-react";
import { endpoints, errorMessage } from "../api/client";
import ApplicationList from "../components/ApplicationList.jsx";
import BookingCard from "../components/BookingCard.jsx";
import SmartMatchList from "../components/SmartMatchList.jsx";
import TaskCard from "../components/TaskCard.jsx";
import { useToast } from "../context/ToastContext.jsx";
import { kes } from "../utils/money";

const EMPTY_FORM = {
  category_id: "",
  title: "",
  description: "",
  location_address: "Nairobi",
  budget_type: "fixed",
  budget_amount: "",
  scheduled_at: ""
};

export default function ClientDashboard() {
  const toast = useToast();
  const location = useLocation();
  const [categories, setCategories] = useState([]);
  const [tasks, setTasks] = useState([]);
  const [bookings, setBookings] = useState([]);
  const [matches, setMatches] = useState({});
  const [openTaskId, setOpenTaskId] = useState(null);
  const [form, setForm] = useState(EMPTY_FORM);
  const [busy, setBusy] = useState(false);

  const loadTasks = useCallback(async () => {
    const { data } = await endpoints.myTasks();
    setTasks(data || []);
  }, []);

  const loadBookings = useCallback(async () => {
    const { data } = await endpoints.bookings();
    setBookings(data || []);
  }, []);

  useEffect(() => {
    endpoints.categories().then(({ data }) => setCategories(data || [])).catch(() => {});
    loadTasks().catch((error) => toast.error(errorMessage(error, "Could not load your tasks.")));
    loadBookings().catch(() => {});
  }, [loadTasks, loadBookings, toast]);

  // "Hire me" on a tasker profile lands here with the category preselected.
  useEffect(() => {
    const prefill = location.state?.prefill;
    if (prefill) setForm((current) => ({ ...current, ...prefill }));
  }, [location.state]);

  const stats = useMemo(() => {
    const active = tasks.filter((task) => ["open", "matched", "in_progress"].includes(task.status)).length;
    const live = bookings.filter((booking) => ["confirmed", "started"].includes(booking.status)).length;
    const spend = bookings
      .filter((booking) => booking.status === "completed")
      .reduce((total, booking) => total + Number(booking.agreed_rate || 0), 0);
    return [
      { label: "Active tasks", value: active },
      { label: "Live bookings", value: live },
      { label: "Completed spend", value: kes(spend) }
    ];
  }, [tasks, bookings]);

  async function postTask(event) {
    event.preventDefault();
    setBusy(true);
    try {
      const payload = {
        ...form,
        category_id: Number(form.category_id),
        budget_amount: Number(form.budget_amount || 0),
        scheduled_at: form.scheduled_at
          ? new Date(form.scheduled_at).toISOString()
          : new Date().toISOString()
      };
      const { data } = await endpoints.createTask(payload);
      toast.success("Task posted. Finding you taskers...");
      setForm(EMPTY_FORM);
      await loadTasks();
      setOpenTaskId(data.id);
      await showMatches(data.id);
    } catch (error) {
      toast.error(errorMessage(error, "Could not post the task."));
    } finally {
      setBusy(false);
    }
  }

  async function showMatches(taskId) {
    try {
      const { data } = await endpoints.matches(taskId);
      setMatches((current) => ({ ...current, [taskId]: data || [] }));
    } catch (error) {
      toast.error(errorMessage(error, "Could not rank taskers."));
    }
  }

  async function cancelTask(taskId) {
    try {
      await endpoints.cancelTask(taskId);
      toast.success("Task cancelled.");
      await loadTasks();
    } catch (error) {
      toast.error(errorMessage(error));
    }
  }

  return (
    <main className="mx-auto max-w-7xl px-4 py-10">
      <div className="mb-8 space-y-2">
        <p className="section-label">Client dashboard</p>
        <h1 className="font-display text-5xl font-black text-brand-dark">Manage tasks and bookings</h1>
      </div>

      <section className="grid gap-4 md:grid-cols-3">
        {stats.map((stat) => (
          <div className="dashboard-stat" key={stat.label}>
            <p className="text-sm font-semibold uppercase tracking-wide text-brand-muted">{stat.label}</p>
            <p className="mt-2 font-display text-4xl font-black text-brand-dark">{stat.value}</p>
          </div>
        ))}
      </section>

      <section className="mt-8 grid gap-8 lg:grid-cols-[.9fr_1.1fr]">
        <form onSubmit={postTask} className="card h-fit space-y-4">
          <h2 className="font-display text-3xl font-black text-brand-dark">Post a Task</h2>
          <select
            className="input"
            value={form.category_id}
            onChange={(event) => setForm({ ...form, category_id: event.target.value })}
            required
            aria-label="Category"
          >
            <option value="">Choose category</option>
            {categories.map((category) => (
              <option key={category.id} value={category.id}>{category.name}</option>
            ))}
          </select>
          <input className="input" placeholder="Task title" value={form.title} required
            onChange={(event) => setForm({ ...form, title: event.target.value })} />
          <textarea className="input min-h-28" placeholder="Describe the task" value={form.description}
            onChange={(event) => setForm({ ...form, description: event.target.value })} />
          <input className="input" placeholder="Location" value={form.location_address}
            onChange={(event) => setForm({ ...form, location_address: event.target.value })} />
          <input className="input" type="datetime-local" value={form.scheduled_at} aria-label="Scheduled for"
            onChange={(event) => setForm({ ...form, scheduled_at: event.target.value })} />
          <div className="grid grid-cols-2 gap-3">
            <select className="input" value={form.budget_type} aria-label="Budget type"
              onChange={(event) => setForm({ ...form, budget_type: event.target.value })}>
              <option value="fixed">Fixed</option>
              <option value="hourly">Hourly</option>
            </select>
            <input className="input" type="number" min="0" placeholder="Budget KES" value={form.budget_amount}
              onChange={(event) => setForm({ ...form, budget_amount: event.target.value })} />
          </div>
          <button disabled={busy} className="btn-primary w-full disabled:opacity-50">
            {busy ? "Posting..." : "Post task & find taskers"}
          </button>
        </form>

        <div className="space-y-5">
          <div className="flex items-center justify-between">
            <h2 className="font-display text-3xl font-black text-brand-dark">My Tasks</h2>
            <Link to="/bookings" className="text-sm font-semibold text-brand-primary hover:underline">
              View bookings →
            </Link>
          </div>

          {tasks.length === 0 && (
            <div className="card text-center text-brand-muted">
              You have not posted any tasks yet.
            </div>
          )}

          {tasks.map((task) => {
            const expanded = openTaskId === task.id;
            const pending = (task.applications || []).filter((a) => a.status === "pending").length;
            return (
              <TaskCard
                key={task.id}
                task={task}
                extraMeta={[`👥 ${pending} pending application${pending === 1 ? "" : "s"}`]}
                action={
                  <div className="flex flex-wrap gap-2">
                    <button onClick={() => setOpenTaskId(expanded ? null : task.id)} className="btn-dark">
                      {expanded ? "Hide details" : "View applications"}
                    </button>
                    <button
                      onClick={() => { setOpenTaskId(task.id); showMatches(task.id); }}
                      className="inline-flex items-center gap-2 rounded-xl border-2 border-brand-border px-4 py-2 text-sm font-semibold text-brand-dark transition-colors hover:border-brand-primary"
                    >
                      <Sparkles size={14} /> Smart match
                    </button>
                    {task.status === "open" && (
                      <button
                        onClick={() => cancelTask(task.id)}
                        className="rounded-xl border-2 border-brand-border px-4 py-2 text-sm font-semibold text-brand-muted transition-colors hover:border-red-300 hover:text-red-600"
                      >
                        Cancel
                      </button>
                    )}
                  </div>
                }
              >
                {expanded && (
                  <div className="mt-5 space-y-5 border-t border-brand-border pt-5">
                    {matches[task.id]?.length > 0 && (
                      <div>
                        <h4 className="mb-3 font-display text-lg font-bold text-brand-dark">Smart matches</h4>
                        <SmartMatchList matches={matches[task.id]} />
                      </div>
                    )}
                    <div>
                      <h4 className="mb-3 font-display text-lg font-bold text-brand-dark">Applications</h4>
                      <ApplicationList
                        task={task}
                        onChanged={async () => { await loadTasks(); await loadBookings(); }}
                      />
                    </div>
                  </div>
                )}
              </TaskCard>
            );
          })}

          {bookings.length > 0 && (
            <div className="space-y-4 pt-4">
              <h2 className="font-display text-3xl font-black text-brand-dark">Recent bookings</h2>
              {bookings.slice(0, 3).map((booking) => (
                <BookingCard key={booking.id} booking={booking} perspective="client" />
              ))}
            </div>
          )}
        </div>
      </section>
    </main>
  );
}
