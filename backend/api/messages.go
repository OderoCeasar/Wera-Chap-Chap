package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	db "wera-chap-chap/backend/db/sqlc"
	chat "wera-chap-chap/backend/websocket"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
	writeWait  = 10 * time.Second
	maxMessage = 4096
)

func (server *Server) listMessages(ctx *gin.Context) {
	scope, ok := server.loadBooking(ctx, "booking_id")
	if !ok {
		return
	}

	rows, err := server.store.ListMessagesByBooking(ctx, scope.Booking.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	messages := make([]messageResponse, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, newMessageResponse(row.Message, row.User))
	}

	// Anything the caller did not write is now seen. Best-effort: failing to
	// mark the thread read is not worth failing the read itself.
	if err := server.store.MarkMessagesRead(ctx, db.MarkMessagesReadParams{
		BookingID: scope.Booking.ID,
		SenderID:  currentUserID(ctx),
	}); err != nil {
		ctx.Error(err) //nolint:errcheck // recorded on the request, not fatal
	}

	ctx.JSON(http.StatusOK, messages)
}

type sendMessageRequest struct {
	Content string `json:"content" binding:"required,max=4000"`
}

func (server *Server) sendMessage(ctx *gin.Context) {
	scope, ok := server.loadBooking(ctx, "booking_id")
	if !ok {
		return
	}

	var req sendMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	message, err := server.persistMessage(ctx, scope.Booking.ID, currentUserID(ctx), req.Content)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	server.hub.Broadcast(scope.Booking.ID, message)
	ctx.JSON(http.StatusCreated, message)
}

// streamMessages upgrades the connection and joins the booking's chat room.
func (server *Server) streamMessages(ctx *gin.Context) {
	scope, ok := server.loadBooking(ctx, "booking_id")
	if !ok {
		return
	}
	bookingID := scope.Booking.ID
	userID := currentUserID(ctx)

	upgrader := websocket.Upgrader{
		// The room is already authorised: WSAuthMiddleware verified the token
		// and loadBooking confirmed this caller is a participant. Origin is not
		// what is guarding this endpoint.
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}

	client := &chat.Client{Hub: server.hub, BookingID: bookingID, Conn: conn, Send: make(chan []byte, 256)}
	server.hub.Register(client)

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
	defer server.hub.Unregister(client)
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
		message, err := server.persistMessage(ctx, bookingID, userID, input.Content)
		if err != nil {
			continue
		}
		server.hub.Broadcast(bookingID, message)
	}
}

// persistMessage stores the message and returns it with its sender nested, the
// shape both the REST response and the WebSocket broadcast carry.
func (server *Server) persistMessage(ctx *gin.Context, bookingID, senderID int64, content string) (messageResponse, error) {
	message, err := server.store.CreateMessage(ctx, db.CreateMessageParams{
		BookingID: bookingID,
		SenderID:  senderID,
		Content:   content,
	})
	if err != nil {
		return messageResponse{}, err
	}

	row, err := server.store.GetMessageWithSender(ctx, message.ID)
	if err != nil {
		return messageResponse{}, err
	}
	return newMessageResponse(row.Message, row.User), nil
}
