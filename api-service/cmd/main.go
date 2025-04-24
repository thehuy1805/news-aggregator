package main

import (
	"context"
	"net/http"

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
	dbConn, err := db.Connect("postgres://user:password@postgres:5432/news?sslmode=disable")
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer dbConn.Close()

	// Khởi tạo Kafka consumer
	go db.ConsumeArticles(context.Background(), dbConn, []string{"kafka:9092"})

	// Khởi tạo router
	r := mux.NewRouter()
	r.Use(middleware.RateLimiter(100, 60)) // Giới hạn 100 yêu cầu/phút
	r.Use(middleware.JWTAuth)

	// Định nghĩa endpoint
	r.HandleFunc("/articles", api.GetArticles(dbConn)).Methods("GET")
	r.HandleFunc("/login", api.Login).Methods("POST")

	r.Handle("/metrics", promhttp.Handler())

	// Khởi động server
	logger.Info("Starting API server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
