import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { endpoints, errorMessage } from "../api/client";
import BookingCard from "../components/BookingCard.jsx";
import TaskCard from "../components/TaskCard.jsx";
import { useToast } from "../context/ToastContext.jsx";
import { kes } from "../utils/money";

const DAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const DEFAULT_SLOTS = DAYS.map((_, day_of_week) => ({
  day_of_week,
  start_time: "08:00",
  end_time: "17:00",
  enabled: day_of_week >= 1 && day_of_week <= 5
}));

export default function TaskerDashboard() {
  const toast = useToast();
  const [categories, setCategories] = useState([]);
  const [feed, setFeed] = useState([]);
  const [applications, setApplications] = useState([]);
  const [bookings, setBookings] = useState([]);
  const [profile, setProfile] = useState(null);
  const [form, setForm] = useState({
    bio: "", hourly_rate: "", years_experience: "", service_radius_km: 10,
    is_available: true, skill_ids: []
  });
  const [slots, setSlots] = useState(DEFAULT_SLOTS);
  const [busy, setBusy] = useState(false);
  const [applyingTo, setApplyingTo] = useState(null);
  const [applyForm, setApplyForm] = useState({ proposed_rate: "", cover_note: "" });

  // Hydrate the form from the saved profile so a save never wipes existing data.
  const loadProfile = useCallback(async () => {
    const { data } = await endpoints.myTaskerProfile();
    setProfile(data);
    setForm({
      bio: data.bio || "",
      hourly_rate: data.hourly_rate ?? "",
      years_experience: data.years_experience ?? "",
      service_radius_km: data.service_radius_km ?? 10,
      is_available: data.is_available ?? true,
      skill_ids: (data.skills || []).map((skill) => skill.category_id)
    });
    if (data.availability?.length) {
      setSlots(
        DEFAULT_SLOTS.map((slot) => {
          const saved = data.availability.find((item) => item.day_of_week === slot.day_of_week);
          return saved
            ? { ...slot, start_time: saved.start_time.slice(0, 5), end_time: saved.end_time.slice(0, 5), enabled: true }
            : { ...slot, enabled: false };
        })
      );
    }
  }, []);

  const loadApplications = useCallback(async () => {
    const { data } = await endpoints.myApplications();
    setApplications(data || []);
  }, []);

  const loadFeed = useCallback(async () => {
    const { data } = await endpoints.tasks();
    setFeed(data || []);
  }, []);

  useEffect(() => {
    endpoints.categories().then(({ data }) => setCategories(data || [])).catch(() => {});
    loadProfile().catch((error) => toast.error(errorMessage(error, "Could not load your profile.")));
    loadFeed().catch(() => {});
    loadApplications().catch(() => {});
    endpoints.myTaskerBookings().then(({ data }) => setBookings(data || [])).catch(() => {});
  }, [loadProfile, loadFeed, loadApplications, toast]);

  const stats = useMemo(() => {
    const earned = bookings
      .filter((booking) => booking.status === "completed")
      .reduce((total, booking) => total + Number(booking.agreed_rate || 0), 0);
    const upcoming = bookings.filter((booking) => ["confirmed", "started"].includes(booking.status)).length;
    const pending = applications.filter((application) => application.status === "pending").length;
    return [
      { label: "Completed earnings", value: kes(earned) },
      { label: "Upcoming bookings", value: upcoming },
      { label: "Pending applications", value: pending }
    ];
  }, [bookings, applications]);

  const appliedTaskIds = useMemo(
    () => new Set(applications.map((application) => application.task_id)),
    [applications]
  );

  async function saveProfile(event) {
    event.preventDefault();
    setBusy(true);
    try {
      await endpoints.updateTaskerProfile({
        bio: form.bio,
        hourly_rate: Number(form.hourly_rate || 0),
        years_experience: Number(form.years_experience || 0),
        service_radius_km: Number(form.service_radius_km || 0),
        is_available: form.is_available,
        skill_ids: form.skill_ids
      });
      toast.success("Profile saved.");
      await loadProfile();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function saveAvailability() {
    setBusy(true);
    try {
      const payload = slots
        .filter((slot) => slot.enabled)
        .map(({ day_of_week, start_time, end_time }) => ({ day_of_week, start_time, end_time }));
      await endpoints.setAvailability(payload);
      toast.success("Availability updated.");
      await loadProfile();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function submitApplication(event) {
    event.preventDefault();
    setBusy(true);
    try {
      await endpoints.applyToTask(applyingTo.id, {
        proposed_rate: Number(applyForm.proposed_rate),
        cover_note: applyForm.cover_note
      });
      toast.success("Application sent.");
      setApplyingTo(null);
      setApplyForm({ proposed_rate: "", cover_note: "" });
      await loadApplications();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  function toggleSkill(categoryId) {
    setForm((current) => ({
      ...current,
      skill_ids: current.skill_ids.includes(categoryId)
        ? current.skill_ids.filter((id) => id !== categoryId)
        : [...current.skill_ids, categoryId]
    }));
  }

  return (
    <main className="mx-auto max-w-7xl px-4 py-10">
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div className="space-y-2">
          <p className="section-label">Tasker dashboard</p>
          <h1 className="font-display text-5xl font-black text-brand-dark">Earn with your skills</h1>
        </div>
        <label className="flex cursor-pointer items-center gap-3 rounded-2xl border border-brand-border bg-brand-surface px-5 py-3">
          <input
            type="checkbox"
            checked={form.is_available}
            onChange={(event) => setForm({ ...form, is_available: event.target.checked })}
            className="h-4 w-4 accent-[#D4500A]"
          />
          <span className="font-semibold text-brand-dark">
            {form.is_available ? "Available for work" : "Not accepting work"}
          </span>
        </label>
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
        <div className="space-y-6">
          <form onSubmit={saveProfile} className="card space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="font-display text-3xl font-black text-brand-dark">My Profile</h2>
              {profile && (
                <Link to={`/taskers/${profile.id}`} className="text-sm font-semibold text-brand-primary hover:underline">
                  Public view →
                </Link>
              )}
            </div>
            <textarea className="input min-h-28" placeholder="Tell clients what you do best" value={form.bio}
              onChange={(event) => setForm({ ...form, bio: event.target.value })} aria-label="Bio" />
            <div className="grid grid-cols-2 gap-3">
              <input className="input" type="number" min="0" placeholder="Hourly rate (KES)" value={form.hourly_rate}
                onChange={(event) => setForm({ ...form, hourly_rate: event.target.value })} aria-label="Hourly rate" />
              <input className="input" type="number" min="0" placeholder="Years experience" value={form.years_experience}
                onChange={(event) => setForm({ ...form, years_experience: event.target.value })} aria-label="Years of experience" />
            </div>
            <div>
              <label className="text-sm font-semibold text-brand-muted">
                Service radius: {form.service_radius_km} km
              </label>
              <input type="range" min="1" max="50" value={form.service_radius_km} className="mt-2 w-full accent-[#D4500A]"
                onChange={(event) => setForm({ ...form, service_radius_km: event.target.value })} />
            </div>
            <div>
              <p className="mb-2 text-sm font-semibold text-brand-muted">Skills</p>
              <div className="flex flex-wrap gap-2">
                {categories.map((category) => (
                  <button type="button" key={category.id} onClick={() => toggleSkill(category.id)}
                    className={`rounded-full px-3 py-2 text-sm font-semibold transition-colors ${
                      form.skill_ids.includes(category.id)
                        ? "bg-brand-primary text-white"
                        : "bg-brand-cream text-brand-charcoal hover:bg-brand-border"
                    }`}>
                    {category.name}
                  </button>
                ))}
              </div>
            </div>
            <button disabled={busy} className="btn-primary w-full disabled:opacity-50">Save profile</button>
          </form>

          <div className="card space-y-3">
            <h2 className="font-display text-3xl font-black text-brand-dark">Availability</h2>
            {slots.map((slot, index) => (
              <div className="grid grid-cols-[1.5rem_3rem_1fr_1fr] items-center gap-2" key={slot.day_of_week}>
                <input type="checkbox" checked={slot.enabled} className="h-4 w-4 accent-[#D4500A]"
                  aria-label={`Available on ${DAYS[slot.day_of_week]}`}
                  onChange={(event) =>
                    setSlots(slots.map((item, i) => (i === index ? { ...item, enabled: event.target.checked } : item)))} />
                <b className="text-brand-dark">{DAYS[slot.day_of_week]}</b>
                <input className="input py-2" type="time" value={slot.start_time} disabled={!slot.enabled}
                  aria-label={`${DAYS[slot.day_of_week]} start`}
                  onChange={(event) =>
                    setSlots(slots.map((item, i) => (i === index ? { ...item, start_time: event.target.value } : item)))} />
                <input className="input py-2" type="time" value={slot.end_time} disabled={!slot.enabled}
                  aria-label={`${DAYS[slot.day_of_week]} end`}
                  onChange={(event) =>
                    setSlots(slots.map((item, i) => (i === index ? { ...item, end_time: event.target.value } : item)))} />
              </div>
            ))}
            <button onClick={saveAvailability} disabled={busy} className="btn-secondary w-full disabled:opacity-50">
              Save availability
            </button>
          </div>
        </div>

        <div className="space-y-5">
          {bookings.length > 0 && (
            <>
              <h2 className="font-display text-3xl font-black text-brand-dark">My bookings</h2>
              {bookings.slice(0, 3).map((booking) => (
                <BookingCard key={booking.id} booking={booking} perspective="tasker" />
              ))}
            </>
          )}

          <h2 className="font-display text-3xl font-black text-brand-dark">Open tasks</h2>
          {feed.length === 0 && (
            <div className="card text-center text-brand-muted">No open tasks right now. Check back soon.</div>
          )}
          {feed.map((task) => {
            const applied = appliedTaskIds.has(task.id);
            return (
              <TaskCard
                key={task.id}
                task={task}
                extraMeta={task.scheduled_at ? [`🗓 ${new Date(task.scheduled_at).toLocaleDateString()}`] : []}
                action={
                  applied ? (
                    <p className="rounded-2xl bg-brand-cream p-3 text-center text-sm font-semibold text-brand-muted">
                      Application sent
                    </p>
                  ) : applyingTo?.id === task.id ? (
                    <form onSubmit={submitApplication} className="space-y-3 border-t border-brand-border pt-4">
                      <input className="input" type="number" min="1" required placeholder="Your rate (KES)"
                        value={applyForm.proposed_rate} aria-label="Proposed rate"
                        onChange={(event) => setApplyForm({ ...applyForm, proposed_rate: event.target.value })} />
                      <textarea className="input min-h-20" placeholder="Why you're a good fit"
                        value={applyForm.cover_note} aria-label="Cover note"
                        onChange={(event) => setApplyForm({ ...applyForm, cover_note: event.target.value })} />
                      <div className="flex gap-2">
                        <button disabled={busy} className="btn-primary flex-1 py-2 text-base disabled:opacity-50">Send</button>
                        <button type="button" onClick={() => setApplyingTo(null)}
                          className="rounded-2xl border-2 border-brand-border px-5 py-2 font-semibold text-brand-muted">
                          Cancel
                        </button>
                      </div>
                    </form>
                  ) : (
                    <button
                      onClick={() => {
                        setApplyingTo(task);
                        setApplyForm({ proposed_rate: task.budget_amount || "", cover_note: "" });
                      }}
                      className="btn-primary w-full py-2 text-base"
                    >
                      Apply
                    </button>
                  )
                }
              />
            );
          })}
        </div>
      </section>
    </main>
  );
}
