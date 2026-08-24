export default function CategoryGrid({ categories = [] }) {
  return (
    <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
      {categories.map((category) => (
        <div key={category.id || category.name} className="group rounded-3xl border border-brand-border bg-brand-surface p-6 transition-all duration-300 hover:-translate-y-1 hover:border-brand-primary hover:shadow-lg hover:shadow-brand-primary/10">
          <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-brand-primary/10 text-2xl transition-colors duration-300 group-hover:bg-brand-primary">
            {category.icon_url}
          </div>
          <h3 className="mt-4 font-body font-semibold text-brand-dark text-sm">{category.name}</h3>
          <p className="mt-2 text-sm text-brand-muted">{category.description}</p>
        </div>
      ))}
    </div>
  );
}
