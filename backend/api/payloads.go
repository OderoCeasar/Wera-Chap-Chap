package api

import (
	"time"

	db "wera-chap-chap/backend/db/sqlc"
	"wera-chap-chap/backend/matching"
)

// The JSON the API returns.
//
// These exist for two reasons. The first is safety: sqlc generates
// db.User with a `password_hash` json tag, so returning a db.User anywhere
// would put every user's bcrypt hash on the wire. Nothing in this package
// serialises a db.User directly -- newUserResponse is the only way out.
//
// The second is shape. The GORM version produced nested objects through
// Preload; the queries that replaced it return flat joined rows, so the nesting
// is rebuilt here. The field names and nesting match what the client already
// consumes -- task.category, tasker.user, booking.task/client/tasker,
// message.sender, review.reviewer.

type userResponse struct {
	ID                    int64                  `json:"id"`
	Email                 string                 `json:"email"`
	FullName              string                 `json:"full_name"`
	Phone                 string                 `json:"phone"`
	Role                  string                 `json:"role"`
	AvatarURL             string                 `json:"avatar_url"`
	IsVerified            bool                   `json:"is_verified"`
	BackgroundCheckPassed bool                   `json:"background_check_passed"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
	TaskerProfile         *taskerProfileResponse `json:"tasker_profile,omitempty"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		ID:                    user.ID,
		Email:                 user.Email,
		FullName:              user.FullName,
		Phone:                 user.Phone,
		Role:                  user.Role,
		AvatarURL:             user.AvatarUrl,
		IsVerified:            user.IsVerified,
		BackgroundCheckPassed: user.BackgroundCheckPassed,
		CreatedAt:             user.CreatedAt,
		UpdatedAt:             user.UpdatedAt,
	}
}

func newUserRef(user db.User) *userResponse {
	response := newUserResponse(user)
	return &response
}

type skillResponse struct {
	ID         int64       `json:"id"`
	TaskerID   int64       `json:"tasker_id"`
	CategoryID int64       `json:"category_id"`
	Category   db.Category `json:"category"`
}

type taskerProfileResponse struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	Bio             string    `json:"bio"`
	HourlyRate      float64   `json:"hourly_rate"`
	YearsExperience int32     `json:"years_experience"`
	ServiceRadiusKM float64   `json:"service_radius_km"`
	IsAvailable     bool      `json:"is_available"`
	AvgRating       float64   `json:"avg_rating"`
	TotalReviews    int32     `json:"total_reviews"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	User *userResponse `json:"user,omitempty"`
	// Skills and Availability carry no omitempty and are never nil: the client
	// calls .map and .find on them without guarding, so an absent key would be
	// a TypeError where an empty array is simply an empty list.
	Skills       []skillResponse         `json:"skills"`
	Availability []db.TaskerAvailability `json:"availability"`
}

func newTaskerProfileResponse(profile db.TaskerProfile) taskerProfileResponse {
	return taskerProfileResponse{
		ID:              profile.ID,
		UserID:          profile.UserID,
		Bio:             profile.Bio,
		HourlyRate:      profile.HourlyRate,
		YearsExperience: profile.YearsExperience,
		ServiceRadiusKM: profile.ServiceRadiusKm,
		IsAvailable:     profile.IsAvailable,
		AvgRating:       profile.AvgRating,
		TotalReviews:    profile.TotalReviews,
		CreatedAt:       profile.CreatedAt,
		UpdatedAt:       profile.UpdatedAt,
		Skills:          []skillResponse{},
		Availability:    []db.TaskerAvailability{},
	}
}

func newTaskerProfileRef(profile db.TaskerProfile, user db.User) *taskerProfileResponse {
	response := newTaskerProfileResponse(profile)
	response.User = newUserRef(user)
	return &response
}

type taskResponse struct {
	ID              int64      `json:"id"`
	ClientID        int64      `json:"client_id"`
	CategoryID      int64      `json:"category_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	LocationAddress string     `json:"location_address"`
	LocationLat     float64    `json:"location_lat"`
	LocationLng     float64    `json:"location_lng"`
	BudgetType      string     `json:"budget_type"`
	BudgetAmount    float64    `json:"budget_amount"`
	Status          string     `json:"status"`
	ScheduledAt     *time.Time `json:"scheduled_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	Client   *userResponse `json:"client,omitempty"`
	Category *db.Category  `json:"category,omitempty"`
	// Applications is omitted rather than empty when it was not loaded: the
	// client reads `task.applications || []`, and an always-empty array would
	// claim a task has no applicants on endpoints that never looked.
	Applications []applicationResponse `json:"applications,omitempty"`
}

func newTaskResponse(task db.Task) taskResponse {
	return taskResponse{
		ID:              task.ID,
		ClientID:        task.ClientID,
		CategoryID:      task.CategoryID,
		Title:           task.Title,
		Description:     task.Description,
		LocationAddress: task.LocationAddress,
		LocationLat:     task.LocationLat,
		LocationLng:     task.LocationLng,
		BudgetType:      task.BudgetType,
		BudgetAmount:    task.BudgetAmount,
		Status:          task.Status,
		ScheduledAt:     task.ScheduledAt,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
	}
}

func newTaskWithRelations(task db.Task, category db.Category, client db.User) taskResponse {
	response := newTaskResponse(task)
	response.Category = &category
	response.Client = newUserRef(client)
	return response
}

func newTaskRef(task db.Task, category db.Category, client db.User) *taskResponse {
	response := newTaskWithRelations(task, category, client)
	return &response
}

type applicationResponse struct {
	ID           int64     `json:"id"`
	TaskID       int64     `json:"task_id"`
	TaskerID     int64     `json:"tasker_id"`
	ProposedRate float64   `json:"proposed_rate"`
	CoverNote    string    `json:"cover_note"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`

	Tasker *taskerProfileResponse `json:"tasker,omitempty"`
	Task   *taskResponse          `json:"task,omitempty"`
}

func newApplicationResponse(application db.TaskApplication) applicationResponse {
	return applicationResponse{
		ID:           application.ID,
		TaskID:       application.TaskID,
		TaskerID:     application.TaskerID,
		ProposedRate: application.ProposedRate,
		CoverNote:    application.CoverNote,
		Status:       application.Status,
		CreatedAt:    application.CreatedAt,
	}
}

type bookingResponse struct {
	ID          int64      `json:"id"`
	TaskID      int64      `json:"task_id"`
	ClientID    int64      `json:"client_id"`
	TaskerID    int64      `json:"tasker_id"`
	AgreedRate  float64    `json:"agreed_rate"`
	Status      string     `json:"status"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`

	Task   *taskResponse          `json:"task,omitempty"`
	Client *userResponse          `json:"client,omitempty"`
	Tasker *taskerProfileResponse `json:"tasker,omitempty"`
}

func newBookingResponse(booking db.Booking) bookingResponse {
	return bookingResponse{
		ID:          booking.ID,
		TaskID:      booking.TaskID,
		ClientID:    booking.ClientID,
		TaskerID:    booking.TaskerID,
		AgreedRate:  booking.AgreedRate,
		Status:      booking.Status,
		StartedAt:   booking.StartedAt,
		CompletedAt: booking.CompletedAt,
		CreatedAt:   booking.CreatedAt,
	}
}

// newBookingWithRelations assembles the nested booking the client renders.
//
// The two user arguments are the reason GetBookingWithRelations joins users
// twice: `client` is who posted the task, `taskerUser` is the account behind
// the tasker profile. sqlc names those columns User and User_2 in join order,
// which is easy to swap by accident -- hence naming them here.
func newBookingWithRelations(
	booking db.Booking,
	task db.Task,
	category db.Category,
	profile db.TaskerProfile,
	client db.User,
	taskerUser db.User,
) bookingResponse {
	response := newBookingResponse(booking)
	response.Task = newTaskRef(task, category, client)
	response.Client = newUserRef(client)
	response.Tasker = newTaskerProfileRef(profile, taskerUser)
	return response
}

type messageResponse struct {
	ID        int64     `json:"id"`
	BookingID int64     `json:"booking_id"`
	SenderID  int64     `json:"sender_id"`
	Content   string    `json:"content"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`

	Sender *userResponse `json:"sender,omitempty"`
}

func newMessageResponse(message db.Message, sender db.User) messageResponse {
	return messageResponse{
		ID:        message.ID,
		BookingID: message.BookingID,
		SenderID:  message.SenderID,
		Content:   message.Content,
		IsRead:    message.IsRead,
		CreatedAt: message.CreatedAt,
		Sender:    newUserRef(sender),
	}
}

type reviewResponse struct {
	ID         int64     `json:"id"`
	BookingID  int64     `json:"booking_id"`
	ReviewerID int64     `json:"reviewer_id"`
	RevieweeID int64     `json:"reviewee_id"`
	Rating     int32     `json:"rating"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`

	Reviewer *userResponse `json:"reviewer,omitempty"`
}

func newReviewResponse(review db.Review, reviewer db.User) reviewResponse {
	return reviewResponse{
		ID:         review.ID,
		BookingID:  review.BookingID,
		ReviewerID: review.ReviewerID,
		RevieweeID: review.RevieweeID,
		Rating:     review.Rating,
		Comment:    review.Comment,
		CreatedAt:  review.CreatedAt,
		Reviewer:   newUserRef(reviewer),
	}
}

type matchResponse struct {
	Tasker            *taskerProfileResponse `json:"tasker"`
	Score             float64                `json:"score"`
	SkillMatch        float64                `json:"skill_match"`
	AvailabilityMatch float64                `json:"availability_match"`
	DistanceScore     float64                `json:"distance_score"`
	RatingScore       float64                `json:"rating_score"`
	PriceMatch        float64                `json:"price_match"`
	DistanceKM        float64                `json:"distance_km"`
}

func newMatchResponses(results []matching.Result) []matchResponse {
	responses := make([]matchResponse, 0, len(results))
	for _, result := range results {
		tasker := newTaskerProfileRef(result.Candidate.Profile, result.Candidate.User)
		tasker.Skills = newSkillResponses(result.Candidate.Skills)
		tasker.Availability = result.Candidate.Availability
		if tasker.Availability == nil {
			tasker.Availability = []db.TaskerAvailability{}
		}

		responses = append(responses, matchResponse{
			Tasker:            tasker,
			Score:             result.Score,
			SkillMatch:        result.SkillMatch,
			AvailabilityMatch: result.AvailabilityMatch,
			DistanceScore:     result.DistanceScore,
			RatingScore:       result.RatingScore,
			PriceMatch:        result.PriceMatch,
			DistanceKM:        result.DistanceKM,
		})
	}
	return responses
}

// newSkillResponses flattens the joined skill rows. The two row types are
// distinct generated structs with identical shapes, so the batch form gets its
// own converter below rather than a shared generic one.
func newSkillResponses(rows []db.ListTaskerSkillsByTaskerIDsRow) []skillResponse {
	skills := make([]skillResponse, 0, len(rows))
	for _, row := range rows {
		skills = append(skills, skillResponse{
			ID:         row.TaskerSkill.ID,
			TaskerID:   row.TaskerSkill.TaskerID,
			CategoryID: row.TaskerSkill.CategoryID,
			Category:   row.Category,
		})
	}
	return skills
}

func newSkillResponsesFromSingle(rows []db.ListTaskerSkillsRow) []skillResponse {
	skills := make([]skillResponse, 0, len(rows))
	for _, row := range rows {
		skills = append(skills, skillResponse{
			ID:         row.TaskerSkill.ID,
			TaskerID:   row.TaskerSkill.TaskerID,
			CategoryID: row.TaskerSkill.CategoryID,
			Category:   row.Category,
		})
	}
	return skills
}
