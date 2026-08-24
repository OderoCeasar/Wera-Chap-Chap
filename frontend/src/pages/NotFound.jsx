import { Link } from "react-router-dom";

export default function NotFound() {
  return (
    <main className="mx-auto max-w-2xl px-4 py-24 text-center">
      <p className="section-label">404</p>
      <h1 className="mt-3 font-display text-5xl font-black text-brand-dark">Page not found</h1>
      <p className="mt-4 text-brand-muted">
        That page has moved or never existed. Let's get you back to work.
      </p>
      <div className="mt-8 flex flex-wrap justify-center gap-4">
        <Link to="/" className="btn-primary">Go home</Link>
        <Link to="/taskers" className="btn-secondary">Browse Taskers</Link>
      </div>
    </main>
  );
}
