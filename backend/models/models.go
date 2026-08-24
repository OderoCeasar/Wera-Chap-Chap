package models

import "time"

type Role string
type TaskStatus string
type BookingStatus string
type ApplicationStatus string
type BudgetType string
type PaymentStatus string

const (
	RoleClient Role = "client"
	RoleTasker Role = "tasker"

	TaskOpen       TaskStatus = "open"
	TaskMatched    TaskStatus = "matched"
	TaskInProgress TaskStatus = "in_progress"
	TaskCompleted  TaskStatus = "completed"
	TaskCancelled  TaskStatus = "cancelled"

	BookingConfirmed BookingStatus = "confirmed"
	BookingStarted   BookingStatus = "started"
	BookingCompleted BookingStatus = "completed"
	BookingCancelled BookingStatus = "cancelled"

	ApplicationPending  ApplicationStatus = "pending"
	ApplicationAccepted ApplicationStatus = "accepted"
	ApplicationRejected ApplicationStatus = "rejected"

	BudgetFixed  BudgetType = "fixed"
	BudgetHourly BudgetType = "hourly"

	PaymentPending   PaymentStatus = "pending"
	PaymentCompleted PaymentStatus = "completed"
	PaymentRefunded  PaymentStatus = "refunded"
)

type User struct {
	ID                    uint           `json:"id" gorm:"primaryKey"`
	Email                 string         `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash          string         `json:"-" gorm:"not null"`
	FullName              string         `json:"full_name" gorm:"not null"`
	Phone                 string         `json:"phone"`
	Role                  Role           `json:"role" gorm:"type:varchar(20);not null"`
	AvatarURL             string         `json:"avatar_url"`
	IsVerified            bool           `json:"is_verified"`
	BackgroundCheckPassed bool           `json:"background_check_passed"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	TaskerProfile         *TaskerProfile `json:"tasker_profile,omitempty"`
}

type TaskerProfile struct {
	ID              uint                 `json:"id" gorm:"primaryKey"`
	UserID          uint                 `json:"user_id" gorm:"uniqueIndex;not null"`
	User            User                 `json:"user"`
	Bio             string               `json:"bio"`
	HourlyRate      float64              `json:"hourly_rate"`
	YearsExperience int                  `json:"years_experience"`
	ServiceRadiusKM float64              `json:"service_radius_km"`
	IsAvailable     bool                 `json:"is_available" gorm:"default:true"`
	AvgRating       float64              `json:"avg_rating"`
	TotalReviews    int                  `json:"total_reviews"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	Skills          []TaskerSkill        `json:"skills,omitempty" gorm:"foreignKey:TaskerID"`
	Availability    []TaskerAvailability `json:"availability,omitempty" gorm:"foreignKey:TaskerID"`
}

type Category struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"uniqueIndex;not null"`
	IconURL     string `json:"icon_url"`
	Description string `json:"description"`
}

type TaskerSkill struct {
	ID         uint     `json:"id" gorm:"primaryKey"`
	TaskerID   uint     `json:"tasker_id" gorm:"index;not null;uniqueIndex:idx_tasker_skill"`
	CategoryID uint     `json:"category_id" gorm:"index;not null;uniqueIndex:idx_tasker_skill"`
	Category   Category `json:"category"`
}

type TaskerAvailability struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	TaskerID  uint   `json:"tasker_id" gorm:"index;not null"`
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type Task struct {
	ID              uint              `json:"id" gorm:"primaryKey"`
	ClientID        uint              `json:"client_id" gorm:"index;not null"`
	Client          User              `json:"client"`
	CategoryID      uint              `json:"category_id" gorm:"index;not null"`
	Category        Category          `json:"category"`
	Title           string            `json:"title" gorm:"not null"`
	Description     string            `json:"description"`
	LocationAddress string            `json:"location_address"`
	LocationLat     float64           `json:"location_lat"`
	LocationLng     float64           `json:"location_lng"`
	BudgetType      BudgetType        `json:"budget_type" gorm:"type:varchar(20);not null"`
	BudgetAmount    float64           `json:"budget_amount"`
	Status          TaskStatus        `json:"status" gorm:"type:varchar(30);default:open"`
	ScheduledAt     time.Time         `json:"scheduled_at"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Applications    []TaskApplication `json:"applications,omitempty"`
}

type TaskApplication struct {
	ID           uint              `json:"id" gorm:"primaryKey"`
	TaskID       uint              `json:"task_id" gorm:"index;not null;uniqueIndex:idx_task_applicant"`
	Task         Task              `json:"task"`
	TaskerID     uint              `json:"tasker_id" gorm:"index;not null;uniqueIndex:idx_task_applicant"`
	Tasker       TaskerProfile     `json:"tasker"`
	ProposedRate float64           `json:"proposed_rate"`
	CoverNote    string            `json:"cover_note"`
	Status       ApplicationStatus `json:"status" gorm:"type:varchar(20);default:pending"`
	CreatedAt    time.Time         `json:"created_at"`
}

type Booking struct {
	ID          uint          `json:"id" gorm:"primaryKey"`
	TaskID      uint          `json:"task_id" gorm:"index;not null"`
	Task        Task          `json:"task"`
	ClientID    uint          `json:"client_id" gorm:"index;not null"`
	Client      User          `json:"client"`
	TaskerID    uint          `json:"tasker_id" gorm:"index;not null"`
	Tasker      TaskerProfile `json:"tasker"`
	AgreedRate  float64       `json:"agreed_rate"`
	Status      BookingStatus `json:"status" gorm:"type:varchar(20);default:confirmed"`
	StartedAt   *time.Time    `json:"started_at"`
	CompletedAt *time.Time    `json:"completed_at"`
	CreatedAt   time.Time     `json:"created_at"`
}

type Message struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BookingID uint      `json:"booking_id" gorm:"index;not null"`
	SenderID  uint      `json:"sender_id" gorm:"index;not null"`
	Content   string    `json:"content" gorm:"not null"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
	Sender    User      `json:"sender"`
}

type Review struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	BookingID  uint      `json:"booking_id" gorm:"index;not null;uniqueIndex:idx_booking_reviewer"`
	ReviewerID uint      `json:"reviewer_id" gorm:"index;not null;uniqueIndex:idx_booking_reviewer"`
	RevieweeID uint      `json:"reviewee_id" gorm:"index;not null"`
	Rating     int       `json:"rating"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
	Reviewer   User      `json:"reviewer" gorm:"foreignKey:ReviewerID"`
}

type Payment struct {
	ID                    uint          `json:"id" gorm:"primaryKey"`
	BookingID             uint          `json:"booking_id" gorm:"uniqueIndex;not null"`
	ClientID              uint          `json:"client_id" gorm:"index;not null"`
	TaskerID              uint          `json:"tasker_id" gorm:"index;not null"`
	Amount                float64       `json:"amount"`
	TipAmount             float64       `json:"tip_amount"`
	StripePaymentIntentID string        `json:"stripe_payment_intent_id" gorm:"column:stripe_payment_intent_id"`
	Status                PaymentStatus `json:"status" gorm:"type:varchar(20);default:pending"`
	CreatedAt             time.Time     `json:"created_at"`
}
