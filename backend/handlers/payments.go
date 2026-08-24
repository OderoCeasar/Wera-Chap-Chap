package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wera-chap-chap/backend/config"
	"wera-chap-chap/backend/models"
)

type PaymentHandler struct {
	db  *gorm.DB
	cfg config.Config
}

func NewPaymentHandler(db *gorm.DB, cfg config.Config) *PaymentHandler {
	return &PaymentHandler{db: db, cfg: cfg}
}

// Initiate creates (or refreshes) the pending payment intent for a booking.
// The provider call is stubbed: we mint a checkout id the callback can settle.
func (h *PaymentHandler) Initiate(c *gin.Context) {
	booking, ok := h.clientBooking(c)
	if !ok {
		return
	}
	var input struct {
		Amount      float64 `json:"amount"`
		PhoneNumber string  `json:"phone_number"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.Amount <= 0 {
		input.Amount = booking.AgreedRate
	}

	var payment models.Payment
	err := h.db.Where("booking_id = ?", booking.ID).First(&payment).Error
	if err == nil && payment.Status == models.PaymentCompleted {
		c.JSON(http.StatusConflict, gin.H{"error": "booking is already paid"})
		return
	}

	checkoutID := fmt.Sprintf("mpesa_intent_%d_%d", booking.ID, time.Now().UnixNano())
	payment.BookingID = booking.ID
	payment.ClientID = booking.ClientID
	payment.TaskerID = booking.TaskerID
	payment.Amount = input.Amount
	payment.StripePaymentIntentID = checkoutID
	payment.Status = models.PaymentPending

	if err := h.db.Where(models.Payment{BookingID: booking.ID}).
		Assign(map[string]interface{}{
			"client_id":                booking.ClientID,
			"tasker_id":                booking.TaskerID,
			"amount":                   payment.Amount,
			"stripe_payment_intent_id": checkoutID,
			"status":                   models.PaymentPending,
		}).FirstOrCreate(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create payment"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"payment":             payment,
		"provider":            "mpesa",
		"checkout_request_id": checkoutID,
		"customer_message":    "Enter your M-Pesa PIN on the STK prompt to complete payment.",
	})
}

// Confirm settles a pending payment. It stands in for the provider callback so
// the flow is usable without live M-Pesa credentials.
func (h *PaymentHandler) Confirm(c *gin.Context) {
	booking, ok := h.clientBooking(c)
	if !ok {
		return
	}
	var payment models.Payment
	if err := h.db.Where("booking_id = ?", booking.ID).First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "start a payment first"})
		return
	}
	if payment.Status == models.PaymentCompleted {
		c.JSON(http.StatusOK, payment)
		return
	}
	h.db.Model(&payment).Update("status", models.PaymentCompleted)
	c.JSON(http.StatusOK, payment)
}

// Tip adds a gratuity on top of a settled payment.
func (h *PaymentHandler) Tip(c *gin.Context) {
	booking, ok := h.clientBooking(c)
	if !ok {
		return
	}
	var input struct {
		TipAmount float64 `json:"tip_amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var payment models.Payment
	if err := h.db.Where("booking_id = ?", booking.ID).First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	h.db.Model(&payment).Update("tip_amount", payment.TipAmount+input.TipAmount)
	c.JSON(http.StatusOK, payment)
}

// Get returns the payment for a booking. Both participants may read it.
func (h *PaymentHandler) Get(c *gin.Context) {
	scope, ok := loadBooking(h.db, c, "booking_id", false)
	if !ok {
		return
	}
	var payment models.Payment
	if err := h.db.Where("booking_id = ?", scope.Booking.ID).First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	c.JSON(http.StatusOK, payment)
}

// MpesaCallback is the unauthenticated provider webhook. It settles the payment
// matching the checkout id Safaricom echoes back.
func (h *PaymentHandler) MpesaCallback(c *gin.Context) {
	var payload struct {
		Body struct {
			STKCallback struct {
				CheckoutRequestID string `json:"CheckoutRequestID"`
				ResultCode        int    `json:"ResultCode"`
				ResultDesc        string `json:"ResultDesc"`
			} `json:"stkCallback"`
		} `json:"Body"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ResultCode": 1, "ResultDesc": "invalid payload"})
		return
	}

	callback := payload.Body.STKCallback
	if callback.CheckoutRequestID != "" {
		status := models.PaymentCompleted
		if callback.ResultCode != 0 {
			status = models.PaymentPending
		}
		h.db.Model(&models.Payment{}).
			Where("stripe_payment_intent_id = ?", callback.CheckoutRequestID).
			Update("status", status)
	}
	// Safaricom expects a 200 acknowledgement regardless of our own outcome.
	c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
}

// clientBooking restricts an action to the booking's client, who is the payer.
func (h *PaymentHandler) clientBooking(c *gin.Context) (models.Booking, bool) {
	scope, ok := loadBooking(h.db, c, "booking_id", false)
	if !ok {
		return scope.Booking, false
	}
	if !scope.IsClient {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the client can pay for a booking"})
		return scope.Booking, false
	}
	return scope.Booking, true
}
