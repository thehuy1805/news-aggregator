package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"api-service/internal/api"
	"api-service/internal/db"
	"api-service/internal/middleware"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	// Khởi tạo logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Kết nối PostgreSQL
	dbConnStr := os.Getenv("POSTGRES_URL")
	if dbConnStr == "" {
		// Nếu không có POSTGRES_URL, build connection string từ các biến Render cung cấp
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		if host != "" && port != "" && user != "" && password != "" && dbname != "" {
			dbConnStr = "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + dbname + "?sslmode=require"
		} else {
			// Fallback giá trị mặc định cho môi trường local (Docker Compose)
			dbConnStr = "postgres://postgres:0937491454az@postgres:5432/news?sslmode=disable"
		}
	}
	dbConn, err := db.Connect(dbConnStr)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer dbConn.Close()

	// Khởi tạo Kafka consumer nếu được bật qua biến môi trường
	enableKafka := os.Getenv("ENABLE_KAFKA_CONSUMER")
	if enableKafka == "true" {
		kafkaBrokers := os.Getenv("KAFKA_BROKERS")
		if kafkaBrokers == "" {
			kafkaBrokers = "kafka:9092"
		}
		go db.ConsumeArticles(context.Background(), dbConn, []string{kafkaBrokers})
	}

	// Khởi tạo router
	r := mux.NewRouter()
	r.Use(middleware.RateLimiter(100, time.Minute)) // Giới hạn 100 yêu cầu/phút

	// Routes không cần authentication
	r.HandleFunc("/login", api.Login).Methods("POST")
	r.Handle("/metrics", promhttp.Handler())

	// Subrouter cho các routes cần authentication
	protected := r.PathPrefix("").Subrouter()
	protected.Use(middleware.JWTAuth)
	protected.HandleFunc("/articles", api.GetArticles(dbConn)).Methods("GET")

	// Khởi động server
	logger.Info("Starting API server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
