export function kes(value) {
  return `KES ${Number(value || 0).toLocaleString("en-KE", { maximumFractionDigits: 0 })}`;
}
