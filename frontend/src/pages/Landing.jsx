import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ShieldCheck, Smartphone, Users } from "lucide-react";
import { endpoints } from "../api/client";
import CategoryGrid from "../components/CategoryGrid.jsx";

const fallbackCategories = [
  { name: "Home Repairs", icon_url: "🛠️", description: "Quick fixes from verified local pros." },
  { name: "Cleaning", icon_url: "🧽", description: "Sparkling spaces without the stress." },
  { name: "Delivery & Errands", icon_url: "🛵", description: "Chap chap help across town." },
  { name: "Moving", icon_url: "🚚", description: "Lifting, packing and relocation support." }
];

export default function Landing() {
  const [categories, setCategories] = useState(fallbackCategories);
  useEffect(() => {
    endpoints.categories().then(({ data }) => setCategories(data)).catch(() => {});
  }, []);
  return (
    <main>
      <section className="relative overflow-hidden bg-brand-cream py-16 lg:py-24">
        <div className="absolute -right-32 -top-32 h-96 w-96 rounded-full bg-brand-primary/10 blur-3xl" />
        <div className="relative mx-auto max-w-7xl px-4">
          <div className="grid gap-12 lg:grid-cols-2 lg:items-center">
            <div className="space-y-8">
              <div className="inline-flex items-center gap-2 rounded-full bg-brand-primary/10 px-4 py-2">
                <span className="h-2 w-2 rounded-full bg-brand-primary animate-pulse" />
                <span className="font-body text-xs font-semibold uppercase tracking-wide text-brand-primary">Built for Kenya's everyday jobs</span>
              </div>
              <h1 className="font-display text-5xl font-black leading-[0.9] text-brand-dark md:text-7xl">Find <span className="italic text-brand-primary">trusted</span> help, book fast, pay safely.</h1>
              <p className="max-w-md font-body text-lg leading-relaxed text-brand-muted">Wera Chap Chap connects clients with vetted Taskers for repairs, cleaning, errands, moving, delivery and more — from Nairobi estates to county towns.</p>
              <div className="flex flex-wrap gap-4 pt-2">
                <Link to="/register" className="btn-primary">Post a task</Link>
                <Link to="/taskers" className="btn-secondary">Browse Taskers</Link>
              </div>
              <div className="flex gap-8 border-t border-brand-border pt-8">
                {[["12k+", "tasks completed"], ["4.8/5", "average rating"], ["47", "service areas"]].map(([value, label]) => (
                  <div key={label}>
                    <p className="font-display text-4xl font-black text-brand-dark">{value}</p>
                    <p className="text-xs font-semibold uppercase tracking-wide text-brand-muted">{label}</p>
                  </div>
                ))}
              </div>
            </div>
            <div className="relative rounded-3xl border border-brand-border bg-brand-dark p-8 text-white shadow-lg">
              <div className="absolute -right-20 -top-20 h-48 w-48 rounded-full bg-brand-primary/20 blur-3xl" />
              <h2 className="relative font-display text-2xl font-black">Today's top matches</h2>
              <div className="relative mt-8 space-y-4">
                {["Assemble queen bed in Kilimani", "Deep clean apartment in Westlands", "Deliver documents to Upper Hill"].map((task, index) => (
                  <div key={task} className="rounded-2xl bg-white/10 p-4 backdrop-blur-sm">
                    <p className="text-xs font-semibold uppercase text-white/60">#{index + 1} smart match</p>
                    <p className="mt-1 font-display font-bold text-base">{task}</p>
                    <p className="mt-2 text-xs text-white/70">✓ Verified Tasker · 📱 M-Pesa ready · 💬 Chat enabled</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </section>
      <section className="mx-auto max-w-7xl px-4 py-16">
        <div className="mb-12 space-y-2">
          <p className="font-body text-xs font-semibold uppercase tracking-[0.2em] text-brand-primary">Categories</p>
          <h2 className="font-display text-4xl font-black text-brand-dark md:text-5xl">What do you need done?</h2>
        </div>
        <CategoryGrid categories={categories} />
      </section>
      <section className="mx-auto grid max-w-7xl gap-6 px-4 py-16 md:grid-cols-3">
        {[["Post", "Tell us the task, location, schedule and budget.", Smartphone], ["Match", "Get ranked Taskers by skill, rating, availability and rate.", Users], ["Pay", "Book securely, chat in-app and pay with M-Pesa.", ShieldCheck]].map(([title, text, Icon]) => (
          <div key={title} className="rounded-3xl border border-brand-border bg-brand-surface p-6 shadow-sm"><Icon className="h-6 w-6 text-brand-primary" /><h3 className="mt-5 font-display text-2xl font-black text-brand-dark">{title}</h3><p className="mt-2 text-brand-charcoal">{text}</p></div>
        ))}
      </section>
    </main>
  );
}
