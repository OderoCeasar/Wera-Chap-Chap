package api

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"wera-chap-chap/backend/config"
	db "wera-chap-chap/backend/db/sqlc"
	"wera-chap-chap/backend/matching"
	"wera-chap-chap/backend/token"
	chat "wera-chap-chap/backend/websocket"
)

// Server serves HTTP requests for the marketplace.
//
// One struct holds every dependency and every handler hangs off it, rather
// than a constructor per resource. Adding a dependency is then a field and a
// line in NewServer instead of a change to several signatures, and a handler
// can reach anything it needs without one being threaded through.
type Server struct {
	config     config.Config
	store      db.Store
	tokenMaker token.Maker
	router     *gin.Engine
	matcher    *matching.Service
	hub        *chat.Hub
	// limiter throttles the unauthenticated auth endpoints, which are otherwise
	// open to credential stuffing.
	limiter *rateLimiter
}

// NewServer creates a new HTTP server and sets up routing.
func NewServer(config config.Config, store db.Store, hub *chat.Hub) (*Server, error) {
	tokenMaker, err := token.NewJWTMaker(config.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
		matcher:    matching.NewService(matching.NewRepository(store)),
		hub:        hub,
		limiter:    newRateLimiter(),
	}

	server.setupRouter()
	return server, nil
}

// Start runs the HTTP server on a specific address.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

// GetRouter exposes the router so tests can drive it without a listener.
func (server *Server) GetRouter() *gin.Engine {
	return server.router
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

func messageResponseBody(message string) gin.H {
	return gin.H{"error": message}
}
