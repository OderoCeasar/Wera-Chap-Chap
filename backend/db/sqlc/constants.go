package db

// The status and role vocabularies. sqlc generates these columns as plain
// strings because they are VARCHAR + CHECK rather than Postgres enums, so
// these constants are the Go-side mirror of the CHECK constraints in
// 000001_init.up.sql. Changing one means changing both.

const (
	RoleClient = "client"
	RoleTasker = "tasker"
)

const (
	TaskOpen       = "open"
	TaskMatched    = "matched"
	TaskInProgress = "in_progress"
	TaskCompleted  = "completed"
	TaskCancelled  = "cancelled"
)

const (
	BookingConfirmed = "confirmed"
	BookingStarted   = "started"
	BookingCompleted = "completed"
	BookingCancelled = "cancelled"
)

const (
	ApplicationPending  = "pending"
	ApplicationAccepted = "accepted"
	ApplicationRejected = "rejected"
)

const (
	BudgetFixed  = "fixed"
	BudgetHourly = "hourly"
)

const (
	PaymentPending   = "pending"
	PaymentCompleted = "completed"
	PaymentRefunded  = "refunded"
)

// DefaultServiceRadiusKm matches the column default in the schema. It is
// repeated here because RegisterUserTx passes the column explicitly and would
// otherwise write 0, which reads as "serves nowhere".
const DefaultServiceRadiusKm = 10

// IsValidRole reports whether role is one the schema will accept, so a bad
// value is a 400 rather than a constraint violation surfacing as a 500.
func IsValidRole(role string) bool {
	return role == RoleClient || role == RoleTasker
}

// IsValidBudgetType reports whether budgetType is one the schema will accept.
func IsValidBudgetType(budgetType string) bool {
	return budgetType == BudgetFixed || budgetType == BudgetHourly
}
