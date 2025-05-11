package main

import (
	"context"
	"os"
	"sync"
	"time"

	"scraper-service/internal/kafka"
	"scraper-service/internal/scraper"

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
