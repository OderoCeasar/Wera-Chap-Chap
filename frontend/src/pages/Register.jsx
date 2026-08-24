import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { errorMessage } from "../api/client";
import { useAuth } from "../context/AuthContext.jsx";

export default function Register() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ role: "client", full_name: "", email: "", phone: "", password: "" });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const user = await register(form);
      navigate(user.role === "tasker" ? "/dashboard/tasker" : "/dashboard/client", { replace: true });
    } catch (err) {
      setError(errorMessage(err, "Could not create account. The email may already be used."));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto max-w-2xl px-4 py-16">
      <form onSubmit={submit} className="card space-y-4">
        <h1 className="font-display text-4xl font-black text-brand-dark">Join Wera Chap Chap</h1>

        <div className="grid grid-cols-2 gap-3 rounded-2xl bg-brand-cream p-2">
          {["client", "tasker"].map((role) => (
            <button
              type="button" key={role} onClick={() => setForm({ ...form, role })}
              className={`rounded-xl px-4 py-3 font-body font-semibold capitalize transition-all ${
                form.role === role ? "bg-brand-primary text-white" : "bg-brand-surface text-brand-dark hover:bg-brand-border"
              }`}
            >
              {role === "client" ? "I need help" : "I want to work"}
            </button>
          ))}
        </div>

        {error && <p className="rounded-2xl bg-red-50 p-4 text-sm font-semibold text-red-600">{error}</p>}

        <input className="input" placeholder="Full name" required autoComplete="name"
          value={form.full_name} onChange={(event) => setForm({ ...form, full_name: event.target.value })} />
        <input className="input" placeholder="Email" type="email" required autoComplete="email"
          value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} />
        <input className="input" placeholder="Phone e.g. +2547..." autoComplete="tel"
          value={form.phone} onChange={(event) => setForm({ ...form, phone: event.target.value })} />
        <input className="input" placeholder="Password (min 8 characters)" type="password" required minLength={8} autoComplete="new-password"
          value={form.password} onChange={(event) => setForm({ ...form, password: event.target.value })} />

        <button disabled={busy} className="btn-primary w-full disabled:opacity-50">
          {busy ? "Creating account..." : "Create account"}
        </button>

        {form.role === "tasker" && (
          <p className="text-sm text-brand-muted">
            Taskers complete profile verification and a background check during onboarding.
          </p>
        )}
        <p className="text-center text-sm text-brand-muted">
          Already have an account?{" "}
          <Link className="font-body font-semibold text-brand-primary hover:text-brand-primary/80" to="/login">
            Login
          </Link>
        </p>
      </form>
    </main>
  );
}
