import { useEffect, useState } from "react";
import { Smartphone } from "lucide-react";
import { endpoints, errorMessage } from "../api/client";
import { useToast } from "../context/ToastContext.jsx";
import { kes } from "../utils/money";

/**
 * M-Pesa style payment flow for a booking. The client initiates an intent, then
 * confirms it; a live deployment would settle via the STK callback instead.
 */
export default function PaymentPanel({ booking, isClient }) {
  const toast = useToast();
  const [payment, setPayment] = useState(null);
  const [amount, setAmount] = useState(booking.agreed_rate || 0);
  const [phone, setPhone] = useState("");
  const [tip, setTip] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let active = true;
    endpoints
      .payment(booking.id)
      .then(({ data }) => active && setPayment(data))
      .catch(() => active && setPayment(null));
    return () => { active = false; };
  }, [booking.id]);

  async function run(action, successMessage) {
    setBusy(true);
    try {
      const { data } = await action();
      setPayment(data.payment || data);
      toast.success(successMessage);
      return true;
    } catch (error) {
      toast.error(errorMessage(error));
      return false;
    } finally {
      setBusy(false);
    }
  }

  const paid = payment?.status === "completed";

  return (
    <div className="card space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="font-display text-xl font-bold text-brand-dark">Payment</h3>
        <span
          className={`rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-wide ${
            paid ? "bg-green-50 text-green-700" : "bg-amber-50 text-amber-700"
          }`}
        >
          {payment?.status || "not started"}
        </span>
      </div>

      <dl className="space-y-1 text-sm">
        <div className="flex justify-between">
          <dt className="text-brand-muted">Agreed rate</dt>
          <dd className="font-semibold text-brand-dark">{kes(booking.agreed_rate)}</dd>
        </div>
        {payment && (
          <>
            <div className="flex justify-between">
              <dt className="text-brand-muted">Amount</dt>
              <dd className="font-semibold text-brand-dark">{kes(payment.amount)}</dd>
            </div>
            {payment.tip_amount > 0 && (
              <div className="flex justify-between">
                <dt className="text-brand-muted">Tip</dt>
                <dd className="font-semibold text-brand-dark">{kes(payment.tip_amount)}</dd>
              </div>
            )}
          </>
        )}
      </dl>

      {!isClient && (
        <p className="rounded-2xl bg-brand-cream p-4 text-sm text-brand-muted">
          {paid
            ? "The client has settled this booking."
            : "Waiting for the client to pay for this booking."}
        </p>
      )}

      {isClient && !paid && (
        <div className="space-y-3">
          <input
            className="input"
            type="number"
            min="0"
            value={amount}
            onChange={(event) => setAmount(event.target.value)}
            placeholder="Amount (KES)"
            aria-label="Amount"
          />
          <input
            className="input"
            value={phone}
            onChange={(event) => setPhone(event.target.value)}
            placeholder="M-Pesa phone e.g. +2547..."
            aria-label="M-Pesa phone number"
          />
          <button
            disabled={busy}
            onClick={() =>
              run(
                () => endpoints.initiatePayment(booking.id, { amount: Number(amount), phone_number: phone }),
                "STK push sent. Approve it on your phone."
              )
            }
            className="btn-primary flex w-full items-center justify-center gap-2 py-3 text-base disabled:opacity-50"
          >
            <Smartphone size={18} /> {payment ? "Resend M-Pesa prompt" : "Pay with M-Pesa"}
          </button>
          {payment && (
            <button
              disabled={busy}
              onClick={() => run(() => endpoints.confirmPayment(booking.id), "Payment confirmed.")}
              className="btn-secondary w-full py-3 text-base disabled:opacity-50"
            >
              I have paid — confirm
            </button>
          )}
        </div>
      )}

      {isClient && paid && (
        <div className="space-y-3 border-t border-brand-border pt-4">
          <p className="text-sm font-semibold text-brand-dark">Add a tip</p>
          <div className="flex gap-2">
            <input
              className="input flex-1"
              type="number"
              min="1"
              value={tip}
              onChange={(event) => setTip(event.target.value)}
              placeholder="Tip amount"
              aria-label="Tip amount"
            />
            <button
              disabled={busy || !Number(tip)}
              onClick={async () => {
                if (await run(() => endpoints.tipPayment(booking.id, Number(tip)), "Thanks for tipping!")) {
                  setTip("");
                }
              }}
              className="btn-dark px-5 disabled:opacity-50"
            >
              Send
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
