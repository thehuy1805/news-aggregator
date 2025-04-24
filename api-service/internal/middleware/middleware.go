package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var jwtKey = []byte("your-secret-key")

func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RateLimiter(limit int, window time.Duration) func(http.Handler) http.Handler {
	type client struct {
		count     int
		lastReset time.Time
	}
	clients := make(map[string]*client)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			now := time.Now()

			c, ok := clients[ip]
			if !ok {
				c = &client{count: 0, lastReset: now}
				clients[ip] = c
			}

			if now.Sub(c.lastReset) > window {
				c.count = 0
				c.lastReset = now
			}

			if c.count >= limit {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			c.count++
			next.ServeHTTP(w, r)
		})
	}
}
