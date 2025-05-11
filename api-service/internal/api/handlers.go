package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"api-service/internal/models"

	"github.com/dgrijalva/jwt-go"
)

var jwtKey = []byte("your-secret-key")

func GetArticles(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, title, description, link, published FROM articles ORDER BY id DESC LIMIT 50")
		if err != nil {
			http.Error(w, "Failed to fetch articles", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var articles []models.Article
		for rows.Next() {
			var a models.Article
			if err := rows.Scan(&a.ID, &a.Title, &a.Description, &a.Link, &a.Published); err != nil {
				http.Error(w, "Failed to scan article", http.StatusInternalServerError)
				return
			}
			articles = append(articles, a)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(articles)
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Kiểm tra thông tin đăng nhập
	if creds.Username != "admin" || creds.Password != "admin" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Tạo JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": creds.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}
