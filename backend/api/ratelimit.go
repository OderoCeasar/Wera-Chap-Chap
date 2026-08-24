package api

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var errTooManyRequests = errors.New("too many auth requests, please slow down")

const (
	// authRate is the sustained allowance per client: one attempt every two
	// seconds, with authBurst available up front so a person logging in
	// normally never meets the limiter.
	authRate  = rate.Limit(0.5)
	authBurst = 10
	// idleEviction is how long an unused client entry is kept before the
	// sweeper drops it, so the map does not grow without bound.
	idleEviction = 10 * time.Minute
)

// rateLimiter throttles the unauthenticated auth endpoints per client IP.
//
// The previous implementation used a single process-wide limiter, which meant
// one noisy client could exhaust the allowance for everybody -- a denial of
// service anyone could trigger. Keying on the IP confines the effect to the
// client causing it.
type rateLimiter struct {
	mutex   sync.Mutex
	clients map[string]*client
}

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newRateLimiter() *rateLimiter {
	limiter := &rateLimiter{clients: make(map[string]*client)}
	go limiter.sweep()
	return limiter
}

// AuthRateLimit rejects a client that is attempting authentication too often.
func (server *Server) AuthRateLimit() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !server.limiter.allow(ctx.ClientIP()) {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, errorResponse(errTooManyRequests))
			return
		}
		ctx.Next()
	}
}

func (r *rateLimiter) allow(ip string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	existing, found := r.clients[ip]
	if !found {
		existing = &client{limiter: rate.NewLimiter(authRate, authBurst)}
		r.clients[ip] = existing
	}
	existing.lastSeen = time.Now()

	return existing.limiter.Allow()
}

func (r *rateLimiter) sweep() {
	ticker := time.NewTicker(idleEviction)
	defer ticker.Stop()

	for range ticker.C {
		r.mutex.Lock()
		for ip, existing := range r.clients {
			if time.Since(existing.lastSeen) > idleEviction {
				delete(r.clients, ip)
			}
		}
		r.mutex.Unlock()
	}
}
