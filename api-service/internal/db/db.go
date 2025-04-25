package db

import (
	"context"
	"database/sql"
	"encoding/json"

	"api-service/internal/models"

	"github.com/IBM/sarama"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func Connect(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	// Tạo bảng nếu chưa tồn tại
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS articles (
			id SERIAL PRIMARY KEY,
			title TEXT,
			description TEXT,
			link TEXT,
			published TEXT
		)
	`)
	return db, err
}

func ConsumeArticles(ctx context.Context, db *sql.DB, brokers []string) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	config := sarama.NewConfig()
	consumer, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		logger.Fatal("Failed to create consumer", zap.Error(err))
	}
	defer consumer.Close()

	partitionConsumer, err := consumer.ConsumePartition("articles", 0, sarama.OffsetNewest)
	if err != nil {
		logger.Fatal("Failed to create partition consumer", zap.Error(err))
	}
	defer partitionConsumer.Close()

	logger.Info("Started consuming articles")

	for {
		select {
		case msg := <-partitionConsumer.Messages():
			var article models.Article
			if err := json.Unmarshal(msg.Value, &article); err != nil {
				logger.Error("Failed to unmarshal article", zap.Error(err))
				continue
			}
			_, err := db.Exec(
				"INSERT INTO articles (title, description, link, published) VALUES ($1, $2, $3, $4)",
				article.Title, article.Description, article.Link, article.Published,
			)
			if err != nil {
				logger.Error("Failed to insert article", zap.Error(err))
			} else {
				logger.Info("Inserted article",
					zap.String("title", article.Title),
					zap.String("link", article.Link))
			}
		case <-ctx.Done():
			return
		}
	}
}
