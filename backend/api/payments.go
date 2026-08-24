package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	db "wera-chap-chap/backend/db/sqlc"
)

var (
	errPaymentNotFound = errors.New("payment not found")
	errNoPaymentYet    = errors.New("start a payment first")
	errAlreadyPaid     = errors.New("booking is already paid")
	errOnlyClientPays  = errors.New("only the client can pay for a booking")
)

type initiatePaymentRequest struct {
	Amount      float64 `json:"amount"`
	PhoneNumber string  `json:"phone_number"`
}

// initiatePayment creates or refreshes the pending payment intent for a
// booking. The provider call is stubbed: we mint a checkout id that the
// callback can later settle, which keeps the whole flow exercisable without
// live M-Pesa credentials.
func (server *Server) initiatePayment(ctx *gin.Context) {
	booking, ok := server.clientBooking(ctx)
	if !ok {
		return
	}

	var req initiatePaymentRequest
	// Bound loosely on purpose: every field has a sensible fallback, so an
	// empty body is a valid "charge the agreed rate".
	_ = ctx.ShouldBindJSON(&req)
	if req.Amount <= 0 {
		req.Amount = booking.AgreedRate
	}

	existing, err := server.store.GetPaymentByBooking(ctx, booking.ID)
	if err == nil && existing.Status == db.PaymentCompleted {
		ctx.JSON(http.StatusConflict, errorResponse(errAlreadyPaid))
		return
	} else if err != nil && !db.IsNotFound(err) {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	checkoutID := fmt.Sprintf("mpesa_intent_%d_%d", booking.ID, time.Now().UnixNano())

	payment, err := server.store.UpsertPayment(ctx, db.UpsertPaymentParams{
		BookingID:             booking.ID,
		ClientID:              booking.ClientID,
		TaskerID:              booking.TaskerID,
		Amount:                req.Amount,
		StripePaymentIntentID: checkoutID,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"payment":             payment,
		"provider":            "mpesa",
		"checkout_request_id": checkoutID,
		"customer_message":    "Enter your M-Pesa PIN on the STK prompt to complete payment.",
	})
}

// confirmPayment settles a pending payment. It stands in for the provider
// callback so the flow is usable without live credentials.
func (server *Server) confirmPayment(ctx *gin.Context) {
	booking, ok := server.clientBooking(ctx)
	if !ok {
		return
	}

	payment, err := server.store.GetPaymentByBooking(ctx, booking.ID)
	if err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errNoPaymentYet))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if payment.Status == db.PaymentCompleted {
		ctx.JSON(http.StatusOK, payment)
		return
	}

	settled, err := server.store.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
		BookingID: booking.ID,
		Status:    db.PaymentCompleted,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, settled)
}

type tipRequest struct {
	TipAmount float64 `json:"tip_amount" binding:"required,gt=0"`
}

// tipPayment adds a gratuity on top of a settled payment.
func (server *Server) tipPayment(ctx *gin.Context) {
	booking, ok := server.clientBooking(ctx)
	if !ok {
		return
	}

	var req tipRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// The increment happens in SQL rather than read-modify-write in Go, so two
	// tips landing at once cannot lose one another.
	payment, err := server.store.AddPaymentTip(ctx, db.AddPaymentTipParams{
		BookingID: booking.ID,
		TipAmount: req.TipAmount,
	})
	if err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errPaymentNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, payment)
}

// getPayment returns the payment for a booking. Both participants may read it.
func (server *Server) getPayment(ctx *gin.Context) {
	scope, ok := server.loadBooking(ctx, "booking_id")
	if !ok {
		return
	}

	payment, err := server.store.GetPaymentByBooking(ctx, scope.Booking.ID)
	if err != nil {
		if db.IsNotFound(err) {
			ctx.JSON(http.StatusNotFound, errorResponse(errPaymentNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, payment)
}

// mpesaCallback is the unauthenticated provider webhook. It settles the payment
// matching the checkout id Safaricom echoes back.
func (server *Server) mpesaCallback(ctx *gin.Context) {
	var payload struct {
		Body struct {
			STKCallback struct {
				CheckoutRequestID string `json:"CheckoutRequestID"`
				ResultCode        int    `json:"ResultCode"`
				ResultDesc        string `json:"ResultDesc"`
			} `json:"stkCallback"`
		} `json:"Body"`
	}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"ResultCode": 1, "ResultDesc": "invalid payload"})
		return
	}

	callback := payload.Body.STKCallback
	if callback.CheckoutRequestID != "" {
		status := db.PaymentCompleted
		if callback.ResultCode != 0 {
			status = db.PaymentPending
		}
		if err := server.store.SettlePaymentByCheckoutID(ctx, db.SettlePaymentByCheckoutIDParams{
			StripePaymentIntentID: callback.CheckoutRequestID,
			Status:                status,
		}); err != nil {
			ctx.Error(err) //nolint:errcheck // acknowledged below regardless
		}
	}

	// Safaricom expects a 200 acknowledgement regardless of our own outcome;
	// anything else makes it retry the callback.
	ctx.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
}

// clientBooking restricts an action to the booking's client, who is the payer.
func (server *Server) clientBooking(ctx *gin.Context) (db.Booking, bool) {
	scope, ok := server.loadBooking(ctx, "booking_id")
	if !ok {
		return scope.Booking, false
	}
	if !scope.IsClient {
		ctx.JSON(http.StatusForbidden, errorResponse(errOnlyClientPays))
		return scope.Booking, false
	}
	return scope.Booking, true
}
