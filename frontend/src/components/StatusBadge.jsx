const tone = {
  // task statuses
  open: "bg-green-50 text-green-700",
  matched: "bg-amber-50 text-amber-700",
  in_progress: "bg-blue-50 text-blue-700",
  // booking statuses
  confirmed: "bg-amber-50 text-amber-700",
  started: "bg-blue-50 text-blue-700",
  // shared terminal states
  completed: "bg-gray-100 text-gray-600",
  cancelled: "bg-red-50 text-red-600",
  // application statuses
  pending: "bg-gray-100 text-gray-600",
  accepted: "bg-green-50 text-green-700",
  rejected: "bg-red-50 text-red-600"
};

export default function StatusBadge({ status }) {
  return (
    <span className={`whitespace-nowrap rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-wide ${tone[status] || tone.pending}`}>
      {String(status || "pending").replace(/_/g, " ")}
    </span>
  );
}
