package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"sync"
	"time"

	"scraper-service/internal/kafka"
	"scraper-service/internal/scraper"

	_ "github.com/lib/pq"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

func main() {
	// Khởi tạo logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Danh sách nguồn RSS
	feeds := []string{
		"https://vnexpress.net/rss/tin-moi-nhat.rss",
		"https://thanhnien.vn/rss/home.rss",
		"https://tuoitre.vn/rss/tin-moi-nhat.rss",
	}

	// Kiểm soát việc sử dụng Kafka producer qua biến môi trường
	useKafka := false
	if v := os.Getenv("ENABLE_KAFKA_PRODUCER"); v == "true" {
		useKafka = true
	}

	var producer sarama.SyncProducer
	var err error
	if useKafka {
		producer, err = kafka.NewProducer([]string{"kafka:9092"})
		if err != nil {
			logger.Fatal("Failed to initialize Kafka producer", zap.Error(err))
		}
		defer producer.Close()
	}

	// Tạo context
	ctx := context.Background()

	// Khởi động một HTTP server đơn giản để Render nhận diện service là alive
	go func() {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		_ = http.ListenAndServe(":8080", nil)
	}()

	// Hàm thu thập tin tức
	scrapeNews := func() {
		var wg sync.WaitGroup
		for _, feedURL := range feeds {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				items, err := scraper.ScrapeFeed(url)
				if err != nil {
					logger.Error("Failed to scrape feed", zap.String("url", url), zap.Error(err))
					return
				}
				logger.Info("Scraped articles", zap.String("url", url), zap.Int("count", len(items)))
				if useKafka {
					for _, item := range items {
						if err := kafka.SendArticle(ctx, producer, "articles", item); err != nil {
							logger.Error("Failed to send article to Kafka", zap.Error(err))
						}
					}
				} else {
					// Lưu trực tiếp vào database nếu không dùng Kafka
					dbHost := os.Getenv("DB_HOST")
					dbPort := os.Getenv("DB_PORT")
					dbUser := os.Getenv("DB_USER")
					dbPassword := os.Getenv("DB_PASSWORD")
					dbName := os.Getenv("DB_NAME")
					if dbHost != "" && dbPort != "" && dbUser != "" && dbPassword != "" && dbName != "" {
						connStr := "postgres://" + dbUser + ":" + dbPassword + "@" + dbHost + ":" + dbPort + "/" + dbName + "?sslmode=require"
						db, err := sql.Open("postgres", connStr)
						if err != nil {
							logger.Error("Failed to connect to database", zap.Error(err))
							return
						}
						defer db.Close()
						for _, item := range items {
							_, err := db.Exec(
								"INSERT INTO articles (title, description, link, published) VALUES ($1, $2, $3, $4)",
								item.Title, item.Description, item.Link, item.Published,
							)
							if err != nil {
								logger.Error("Failed to insert article", zap.Error(err))
							}
						}
					}
				}
			}(feedURL)
		}
		wg.Wait()
	}

	// Chạy lần đầu ngay lập tức
	scrapeNews()

	// Thu thập tin tức định kỳ
	ticker := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-ticker.C:
			scrapeNews()
		}
	}
}
