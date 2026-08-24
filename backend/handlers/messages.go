package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"wera-chap-chap/backend/middleware"
	"wera-chap-chap/backend/models"
	chat "wera-chap-chap/backend/websocket"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
	writeWait  = 10 * time.Second
	maxMessage = 4096
)

type MessageHandler struct {
	db  *gorm.DB
	hub *chat.Hub
}

func NewMessageHandler(db *gorm.DB, hub *chat.Hub) *MessageHandler {
	return &MessageHandler{db: db, hub: hub}
}

func (h *MessageHandler) History(c *gin.Context) {
	scope, ok := loadBooking(h.db, c, "booking_id", false)
	if !ok {
		return
	}
	messages := []models.Message{}
	h.db.Preload("Sender").Where("booking_id = ?", scope.Booking.ID).
		Order("created_at asc").Find(&messages)

	// Anything the caller did not write is now seen.
	h.db.Model(&models.Message{}).
		Where("booking_id = ? AND sender_id <> ? AND is_read = ?", scope.Booking.ID, middleware.CurrentUserID(c), false).
		Update("is_read", true)

	c.JSON(http.StatusOK, messages)
}

func (h *MessageHandler) Send(c *gin.Context) {
	scope, ok := loadBooking(h.db, c, "booking_id", false)
	if !ok {
		return
	}
	var input struct {
		Content string `json:"content" binding:"required,max=4000"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	message, err := h.persist(scope.Booking.ID, middleware.CurrentUserID(c), input.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not send message"})
		return
	}
	h.hub.Broadcast(scope.Booking.ID, message)
	c.JSON(http.StatusCreated, message)
}

func (h *MessageHandler) WebSocket(c *gin.Context) {
	scope, ok := loadBooking(h.db, c, "booking_id", false)
	if !ok {
		return
	}
	bookingID := scope.Booking.ID
	userID := middleware.CurrentUserID(c)

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &chat.Client{Hub: h.hub, BookingID: bookingID, Conn: conn, Send: make(chan []byte, 256)}
	h.hub.Register(client)

	// Write pump: drains the fan-out channel and keeps the link alive.
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			conn.Close()
		}()
		for {
			select {
			case data, open := <-client.Send:
				if !open {
					conn.SetWriteDeadline(time.Now().Add(writeWait))
					conn.WriteMessage(websocket.CloseMessage, nil)
					return
				}
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					return
				}
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// Read pump runs on this goroutine and owns the unregister.
	defer h.hub.Unregister(client)
	conn.SetReadLimit(maxMessage)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		var input struct {
			Content string `json:"content"`
		}
		if err := conn.ReadJSON(&input); err != nil {
			return
		}
		if input.Content == "" {
			continue
		}
		message, err := h.persist(bookingID, userID, input.Content)
		if err != nil {
			continue
		}
		h.hub.Broadcast(bookingID, message)
	}
}

func (h *MessageHandler) persist(bookingID, senderID uint, content string) (models.Message, error) {
	message := models.Message{BookingID: bookingID, SenderID: senderID, Content: content}
	if err := h.db.Create(&message).Error; err != nil {
		return message, err
	}
	h.db.Preload("Sender").First(&message, message.ID)
	return message, nil
}
