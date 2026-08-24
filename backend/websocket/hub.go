package websocket

import (
	"encoding/json"
	"strconv"

	"github.com/gorilla/websocket"
)

type Event struct {
	BookingID uint        `json:"booking_id"`
	Payload   interface{} `json:"payload"`
}

type Client struct {
	Hub       *Hub
	BookingID uint
	Conn      *websocket.Conn
	Send      chan []byte
}

type Hub struct {
	rooms      map[uint]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Event
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[uint]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Event),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if h.rooms[client.BookingID] == nil {
				h.rooms[client.BookingID] = make(map[*Client]bool)
			}
			h.rooms[client.BookingID][client] = true
		case client := <-h.unregister:
			if _, ok := h.rooms[client.BookingID][client]; ok {
				delete(h.rooms[client.BookingID], client)
				close(client.Send)
			}
		case event := <-h.broadcast:
			data, err := json.Marshal(event.Payload)
			if err != nil {
				continue
			}
			for client := range h.rooms[event.BookingID] {
				// Never block the hub on one slow reader: drop the client
				// instead, its read pump will notice the closed channel.
				select {
				case client.Send <- data:
				default:
					delete(h.rooms[event.BookingID], client)
					close(client.Send)
				}
			}
			if len(h.rooms[event.BookingID]) == 0 {
				delete(h.rooms, event.BookingID)
			}
		}
	}
}

func (h *Hub) Register(client *Client)   { h.register <- client }
func (h *Hub) Unregister(client *Client) { h.unregister <- client }
func (h *Hub) Broadcast(bookingID uint, payload interface{}) {
	h.broadcast <- Event{BookingID: bookingID, Payload: payload}
}

func ParseUint(value string) uint {
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return uint(parsed)
}
