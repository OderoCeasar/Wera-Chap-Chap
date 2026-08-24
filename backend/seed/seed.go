package seed

import (
	"context"
	"log"
	"time"

	"wera-chap-chap/backend/config"
	db "wera-chap-chap/backend/db/sqlc"
	"wera-chap-chap/backend/utils"
)

// demoPassword is the login for every seeded account. Local only -- Run is a
// no-op unless SEED_DEMO is set.
const demoPassword = "password123"

type demoTasker struct {
	Name, Email, Bio string
	Rate             float64
	Years            int32
	Radius           float64
	Skills           []string
}

var demoTaskers = []demoTasker{
	{"Amina Wanjiru", "amina@example.com", "Detail-obsessed cleaner serving Westlands and Parklands for six years.", 850, 6, 15, []string{"Cleaning", "Yard Work"}},
	{"Brian Otieno", "brian@example.com", "Handyman and furniture assembler. No job too small, chap chap delivery.", 1200, 8, 25, []string{"Handyman", "Furniture Assembly", "Home Repairs"}},
	{"Faith Njeri", "faith@example.com", "Reliable errand runner and personal assistant across Nairobi CBD.", 600, 3, 20, []string{"Delivery & Errands", "Personal Assistant"}},
	{"Kevin Mutua", "kevin@example.com", "Moving crew lead with a covered pickup. Careful with fragile items.", 1500, 5, 40, []string{"Moving", "Delivery & Errands"}},
	{"Grace Achieng", "grace@example.com", "Plumbing, electrical and general repairs. Certified and insured.", 1800, 10, 30, []string{"Home Repairs", "Handyman"}},
}

var demoTasks = []struct {
	Title, Description, Category, Location string
	Budget                                 float64
}{
	{"Assemble a queen bed and wardrobe", "Flat-pack delivery arriving Saturday morning, needs assembly.", "Furniture Assembly", "Kilimani, Nairobi", 3500},
	{"Deep clean a 2-bedroom apartment", "Post-tenancy clean including kitchen and balcony.", "Cleaning", "Westlands, Nairobi", 5000},
	{"Fix a leaking kitchen sink", "Slow leak under the sink, likely a worn seal.", "Home Repairs", "Lavington, Nairobi", 2500},
}

// Run populates a browsable marketplace for local development. It is opt-in via
// SEED_DEMO and never touches an account that already exists, so it is safe to
// leave enabled across restarts.
//
// The category rows it references are reference data and arrive with migration
// 000002 rather than from here.
func Run(ctx context.Context, store db.Store, cfg config.Config) {
	if !cfg.SeedDemo {
		return
	}

	categories, err := store.ListCategories(ctx)
	if err != nil || len(categories) == 0 {
		if err != nil {
			log.Printf("seed: could not list categories: %v", err)
		}
		return
	}

	categoryID := func(name string) int64 {
		for _, category := range categories {
			if category.Name == name {
				return category.ID
			}
		}
		return categories[0].ID
	}

	password, err := utils.HashPassword(demoPassword)
	if err != nil {
		log.Printf("seed: could not hash demo password: %v", err)
		return
	}

	created := 0
	for _, demo := range demoTaskers {
		if _, err := store.GetUserByEmail(ctx, demo.Email); err == nil {
			continue
		} else if !db.IsNotFound(err) {
			log.Printf("seed: lookup %s: %v", demo.Email, err)
			continue
		}

		if err := seedTasker(ctx, store, demo, password, categoryID); err != nil {
			log.Printf("seed tasker %s: %v", demo.Email, err)
			continue
		}
		created++
	}

	seedClient(ctx, store, password, categoryID)

	if created > 0 {
		log.Printf("seeded %d demo taskers (login with any listed email / %s)", created, demoPassword)
	}
}

// seedTasker creates the account, its profile, its skills and a weekday
// schedule as one unit. RegisterUserTx already creates the empty profile for a
// tasker; AfterCreate fills it in and adds the rest inside the same
// transaction, so a half-built tasker never becomes visible to the directory.
func seedTasker(ctx context.Context, store db.Store, demo demoTasker, password string, categoryID func(string) int64) error {
	_, err := store.RegisterUserTx(ctx, db.RegisterUserTxParams{
		CreateUserParams: db.CreateUserParams{
			Email:                 demo.Email,
			PasswordHash:          password,
			FullName:              demo.Name,
			Phone:                 "+254700000000",
			Role:                  db.RoleTasker,
			IsVerified:            true,
			BackgroundCheckPassed: true,
		},
		AfterCreate: func(user db.User, q *db.Queries) error {
			profile, err := q.GetTaskerProfileByUserID(ctx, user.ID)
			if err != nil {
				return err
			}

			if _, err := q.UpdateTaskerProfile(ctx, db.UpdateTaskerProfileParams{
				ID:              profile.ID,
				Bio:             &demo.Bio,
				HourlyRate:      &demo.Rate,
				YearsExperience: &demo.Years,
				ServiceRadiusKm: &demo.Radius,
			}); err != nil {
				return err
			}

			for _, skill := range demo.Skills {
				if _, err := q.CreateTaskerSkill(ctx, db.CreateTaskerSkillParams{
					TaskerID:   profile.ID,
					CategoryID: categoryID(skill),
				}); err != nil && !db.IsNotFound(err) {
					return err
				}
			}

			// Available every weekday, 8am to 6pm.
			for day := int32(1); day <= 5; day++ {
				if _, err := q.CreateTaskerAvailability(ctx, db.CreateTaskerAvailabilityParams{
					TaskerID:  profile.ID,
					DayOfWeek: day,
					StartTime: "08:00",
					EndTime:   "18:00",
				}); err != nil {
					return err
				}
			}
			return nil
		},
	})
	return err
}

// seedClient adds a demo client with a few open tasks, so a tasker logging in
// for the first time sees a populated feed rather than an empty one.
func seedClient(ctx context.Context, store db.Store, password string, categoryID func(string) int64) {
	const email = "client@example.com"

	if _, err := store.GetUserByEmail(ctx, email); err == nil {
		return
	} else if !db.IsNotFound(err) {
		log.Printf("seed: lookup %s: %v", email, err)
		return
	}

	result, err := store.RegisterUserTx(ctx, db.RegisterUserTxParams{
		CreateUserParams: db.CreateUserParams{
			Email:        email,
			PasswordHash: password,
			FullName:     "Daniel Kimani",
			Phone:        "+254711000000",
			Role:         db.RoleClient,
			IsVerified:   true,
		},
	})
	if err != nil {
		log.Printf("seed client: %v", err)
		return
	}

	scheduledAt := time.Now().Add(48 * time.Hour)
	for _, demo := range demoTasks {
		if _, err := store.CreateTask(ctx, db.CreateTaskParams{
			ClientID:        result.User.ID,
			CategoryID:      categoryID(demo.Category),
			Title:           demo.Title,
			Description:     demo.Description,
			LocationAddress: demo.Location,
			BudgetType:      db.BudgetFixed,
			BudgetAmount:    demo.Budget,
			ScheduledAt:     &scheduledAt,
		}); err != nil {
			log.Printf("seed task %q: %v", demo.Title, err)
		}
	}
}
