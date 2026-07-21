// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"git.happydns.org/happyDomain/model"
)

// rateLimiterEntry holds the token bucket for a single client along with the
// last time it was seen, so idle entries can be evicted.
type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter throttles requests per client IP using a token bucket. It is
// meant to guard sensitive unauthenticated endpoints (account recovery, email
// validation) against brute-force and timing attacks.
type IPRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*rateLimiterEntry
	rate    rate.Limit
	burst   int
}

// NewIPRateLimiter creates a rate limiter allowing r requests per second per
// client IP, with a burst of b. A background goroutine periodically evicts
// entries that have been idle to keep memory bounded.
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	l := &IPRateLimiter{
		clients: make(map[string]*rateLimiterEntry),
		rate:    r,
		burst:   b,
	}

	go l.cleanupLoop()

	return l
}

// cleanupLoop evicts entries that have not been seen recently.
func (l *IPRateLimiter) cleanupLoop() {
	for range time.Tick(time.Minute) {
		l.mu.Lock()
		for ip, entry := range l.clients {
			if time.Since(entry.lastSeen) > 3*time.Minute {
				delete(l.clients, ip)
			}
		}
		l.mu.Unlock()
	}
}

// getLimiter returns the token bucket associated with the given IP, creating
// one if needed.
func (l *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.clients[ip]
	if !ok {
		entry = &rateLimiterEntry{limiter: rate.NewLimiter(l.rate, l.burst)}
		l.clients[ip] = entry
	}
	entry.lastSeen = time.Now()

	return entry.limiter
}

// Middleware returns a gin handler that rejects requests exceeding the
// configured rate with HTTP 429.
func (l *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.getLimiter(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, happydns.ErrorResponse{Message: "Too many requests, please try again later."})
			return
		}

		c.Next()
	}
}
