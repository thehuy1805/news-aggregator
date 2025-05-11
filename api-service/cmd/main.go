package main

import (
	"context"
	"net/http"
	"os"

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
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		logger.Fatal("DATABASE_URL không được thiết lập")
	}
	dbConn, err := db.Connect(connStr)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer dbConn.Close()

	// Khởi tạo Kafka consumer
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "kafka:9092"
	}
	go db.ConsumeArticles(context.Background(), dbConn, []string{kafkaBrokers})

	// Khởi tạo router
	r := mux.NewRouter()
	r.Use(middleware.RateLimiter(100, 60)) // Giới hạn 100 yêu cầu/phút

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
