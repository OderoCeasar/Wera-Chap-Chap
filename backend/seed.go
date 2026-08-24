package main

import (
	"log"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"wera-chap-chap/backend/models"
)

// seedDemoData populates a browsable marketplace for local development. It is
// opt-in via SEED_DEMO and never touches accounts that already exist.
func seedDemoData(db *gorm.DB) {
	if os.Getenv("SEED_DEMO") != "true" {
		return
	}

	var categories []models.Category
	db.Order("name asc").Find(&categories)
	if len(categories) == 0 {
		return
	}
	categoryID := func(name string) uint {
		for _, category := range categories {
			if category.Name == name {
				return category.ID
			}
		}
		return categories[0].ID
	}

	demoTaskers := []struct {
		Name, Email, Bio string
		Rate             float64
		Years            int
		Radius           float64
		Skills           []string
	}{
		{"Amina Wanjiru", "amina@example.com", "Detail-obsessed cleaner serving Westlands and Parklands for six years.", 850, 6, 15, []string{"Cleaning", "Yard Work"}},
		{"Brian Otieno", "brian@example.com", "Handyman and furniture assembler. No job too small, chap chap delivery.", 1200, 8, 25, []string{"Handyman", "Furniture Assembly", "Home Repairs"}},
		{"Faith Njeri", "faith@example.com", "Reliable errand runner and personal assistant across Nairobi CBD.", 600, 3, 20, []string{"Delivery & Errands", "Personal Assistant"}},
		{"Kevin Mutua", "kevin@example.com", "Moving crew lead with a covered pickup. Careful with fragile items.", 1500, 5, 40, []string{"Moving", "Delivery & Errands"}},
		{"Grace Achieng", "grace@example.com", "Plumbing, electrical and general repairs. Certified and insured.", 1800, 10, 30, []string{"Home Repairs", "Handyman"}},
	}

	password, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	created := 0

	for _, demo := range demoTaskers {
		var count int64
		db.Model(&models.User{}).Where("email = ?", demo.Email).Count(&count)
		if count > 0 {
			continue
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			user := models.User{
				Email: demo.Email, PasswordHash: string(password), FullName: demo.Name,
				Phone: "+254700000000", Role: models.RoleTasker,
				IsVerified: true, BackgroundCheckPassed: true,
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
			profile := models.TaskerProfile{
				UserID: user.ID, Bio: demo.Bio, HourlyRate: demo.Rate,
				YearsExperience: demo.Years, ServiceRadiusKM: demo.Radius, IsAvailable: true,
			}
			if err := tx.Create(&profile).Error; err != nil {
				return err
			}
			for _, skill := range demo.Skills {
				if err := tx.Create(&models.TaskerSkill{TaskerID: profile.ID, CategoryID: categoryID(skill)}).Error; err != nil {
					return err
				}
			}
			// Available every weekday, 8am to 6pm.
			for day := 1; day <= 5; day++ {
				if err := tx.Create(&models.TaskerAvailability{
					TaskerID: profile.ID, DayOfWeek: day, StartTime: "08:00", EndTime: "18:00",
				}).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Printf("seed tasker %s: %v", demo.Email, err)
			continue
		}
		created++
	}

	// A demo client with a couple of open tasks so the tasker feed is populated.
	var client models.User
	if err := db.Where("email = ?", "client@example.com").First(&client).Error; err != nil {
		client = models.User{
			Email: "client@example.com", PasswordHash: string(password), FullName: "Daniel Kimani",
			Phone: "+254711000000", Role: models.RoleClient, IsVerified: true,
		}
		if err := db.Create(&client).Error; err != nil {
			log.Printf("seed client: %v", err)
			return
		}
		demoTasks := []struct {
			Title, Description, Category, Location string
			Budget                                 float64
		}{
			{"Assemble a queen bed and wardrobe", "Flat-pack delivery arriving Saturday morning, needs assembly.", "Furniture Assembly", "Kilimani, Nairobi", 3500},
			{"Deep clean a 2-bedroom apartment", "Post-tenancy clean including kitchen and balcony.", "Cleaning", "Westlands, Nairobi", 5000},
			{"Fix a leaking kitchen sink", "Slow leak under the sink, likely a worn seal.", "Home Repairs", "Lavington, Nairobi", 2500},
		}
		for _, demo := range demoTasks {
			db.Create(&models.Task{
				ClientID: client.ID, CategoryID: categoryID(demo.Category), Title: demo.Title,
				Description: demo.Description, LocationAddress: demo.Location,
				BudgetType: models.BudgetFixed, BudgetAmount: demo.Budget,
				Status: models.TaskOpen, ScheduledAt: time.Now().Add(48 * time.Hour),
			})
		}
	}

	if created > 0 {
		log.Printf("seeded %d demo taskers (login with any listed email / password123)", created)
	}
}
