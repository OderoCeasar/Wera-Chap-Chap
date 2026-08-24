import { useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { errorMessage } from "../api/client";
import { useAuth } from "../context/AuthContext.jsx";

export default function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [form, setForm] = useState({ email: "", password: "" });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const user = await login(form);
      const fallback = user.role === "tasker" ? "/dashboard/tasker" : "/dashboard/client";
      navigate(location.state?.from || fallback, { replace: true });
    } catch (err) {
      setError(errorMessage(err, "Check your email and password, then try again."));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto grid max-w-5xl gap-8 px-4 py-16 md:grid-cols-2 md:items-center">
      <section className="card bg-brand-dark text-white">
        <h1 className="font-display text-5xl font-black leading-tight">Karibu back.</h1>
        <p className="mt-6 font-body text-lg text-white/70">
          Your bookings, tasks and messages are waiting.
        </p>
      </section>

      <form onSubmit={submit} className="card space-y-4">
        <h2 className="font-display text-3xl font-black text-brand-dark">Login</h2>
        {error && <p className="rounded-2xl bg-red-50 p-4 text-sm font-semibold text-red-600">{error}</p>}
        <input
          className="input" placeholder="Email" type="email" required autoComplete="email"
          value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })}
        />
        <input
          className="input" placeholder="Password" type="password" required autoComplete="current-password"
          value={form.password} onChange={(event) => setForm({ ...form, password: event.target.value })}
        />
        <button disabled={busy} className="btn-primary w-full disabled:opacity-50">
          {busy ? "Signing in..." : "Login"}
        </button>
        <p className="text-center text-sm text-brand-muted">
          New here?{" "}
          <Link className="font-body font-semibold text-brand-primary hover:text-brand-primary/80" to="/register">
            Create an account
          </Link>
        </p>
      </form>
    </main>
  );
}
