import { useEffect, useState } from "react";
import { Link, NavLink, useLocation, useNavigate } from "react-router-dom";
import { Menu, Sparkles, X } from "lucide-react";
import { useAuth } from "../context/AuthContext.jsx";

export default function Navbar() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [open, setOpen] = useState(false);

  // Close the mobile drawer whenever the route changes.
  useEffect(() => setOpen(false), [location.pathname]);

  const dashboard = user?.role === "tasker" ? "/dashboard/tasker" : "/dashboard/client";

  const links = [
    { to: "/taskers", label: "Find Taskers" },
    ...(user ? [{ to: dashboard, label: "Dashboard" }, { to: "/bookings", label: "Bookings" }] : [])
  ];

  function signOut() {
    logout();
    navigate("/");
  }

  return (
    <header className="sticky top-0 z-50 border-b border-white/10 bg-brand-dark">
      <nav className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4">
        <Link to="/" className="flex items-center gap-2 font-display text-xl font-black text-white">
          <span className="rounded-2xl bg-brand-primary p-2 text-white"><Sparkles size={18} /></span>
          Wera Chap Chap
        </Link>

        <div className="hidden items-center gap-6 md:flex">
          {links.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              className={({ isActive }) =>
                `font-body font-medium transition-colors hover:text-white ${isActive ? "text-white" : "text-white/70"}`
              }
            >
              {link.label}
            </NavLink>
          ))}
          {user ? (
            <>
              <span className="font-body text-sm text-white/60">{user.full_name}</span>
              <button
                onClick={signOut}
                className="rounded-full border border-white/20 px-4 py-2 font-body font-semibold text-white transition-colors hover:bg-white/10"
              >
                Logout
              </button>
            </>
          ) : (
            <>
              <Link to="/login" className="font-body font-medium text-white/70 transition-colors hover:text-white">
                Login
              </Link>
              <Link
                to="/register"
                className="rounded-full bg-brand-primary px-5 py-2 font-body font-semibold text-white transition-all hover:bg-brand-primary/90 active:scale-95"
              >
                Get started
              </Link>
            </>
          )}
        </div>

        <button
          onClick={() => setOpen((current) => !current)}
          aria-label={open ? "Close menu" : "Open menu"}
          aria-expanded={open}
          className="text-white md:hidden"
        >
          {open ? <X /> : <Menu />}
        </button>
      </nav>

      {open && (
        <div className="border-t border-white/10 bg-brand-dark px-4 pb-6 md:hidden">
          <div className="flex flex-col gap-1 pt-2">
            {links.map((link) => (
              <NavLink key={link.to} to={link.to} className="rounded-xl px-3 py-3 font-body font-medium text-white/80 hover:bg-white/10">
                {link.label}
              </NavLink>
            ))}
            {user ? (
              <button onClick={signOut} className="mt-2 rounded-xl border border-white/20 px-3 py-3 text-left font-body font-semibold text-white">
                Logout
              </button>
            ) : (
              <>
                <Link to="/login" className="rounded-xl px-3 py-3 font-body font-medium text-white/80 hover:bg-white/10">Login</Link>
                <Link to="/register" className="mt-2 rounded-xl bg-brand-primary px-3 py-3 text-center font-body font-semibold text-white">
                  Get started
                </Link>
              </>
            )}
          </div>
        </div>
      )}
    </header>
  );
}
