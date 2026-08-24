-- name: GetPaymentByBooking :one
SELECT * FROM payments
WHERE booking_id = $1 LIMIT 1;

-- name: UpsertPayment :one
-- One payment per booking (booking_id is UNIQUE), so re-initiating replaces the
-- pending intent rather than accumulating rows. The tip is left alone: it is
-- added separately and must survive a retried checkout.
INSERT INTO payments (booking_id, client_id, tasker_id, amount, stripe_payment_intent_id, status)
VALUES ($1, $2, $3, $4, $5, 'pending')
ON CONFLICT (booking_id) DO UPDATE
SET
  client_id = EXCLUDED.client_id,
  tasker_id = EXCLUDED.tasker_id,
  amount = EXCLUDED.amount,
  stripe_payment_intent_id = EXCLUDED.stripe_payment_intent_id,
  status = 'pending'
RETURNING *;

-- name: UpdatePaymentStatus :one
UPDATE payments
SET status = $2
WHERE booking_id = $1
RETURNING *;

-- name: AddPaymentTip :one
UPDATE payments
SET tip_amount = tip_amount + $2
WHERE booking_id = $1
RETURNING *;

-- name: SettlePaymentByCheckoutID :exec
-- The provider callback knows only the checkout id it echoes back.
UPDATE payments
SET status = $2
WHERE stripe_payment_intent_id = $1;
