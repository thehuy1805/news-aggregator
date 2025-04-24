package main

import (
	"context"
	"sync"
	"time"

	"scraper-service/internal/kafka"
	"scraper-service/internal/scraper"

	"go.uber.org/zap"
)

func main() {
	// Khởi tạo logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Danh sách nguồn RSS
	feeds := []string{
		"https://vnexpress.net/rss/tin-moi-nhat.rss",
		"https://www.bbc.com/news/rss.xml",
	}

	// Khởi tạo Kafka producer
	producer, err := kafka.NewProducer([]string{"kafka:9092"})
	if err != nil {
		logger.Fatal("Failed to initialize Kafka producer", zap.Error(err))
	}
	defer producer.Close()

	// Tạo context
	ctx := context.Background()

	// Thu thập tin tức định kỳ
	ticker := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-ticker.C:
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
					for _, item := range items {
						if err := kafka.SendArticle(ctx, producer, "articles", item); err != nil {
							logger.Error("Failed to send article to Kafka", zap.Error(err))
						}
					}
				}(feedURL)
			}
			wg.Wait()
		}
	}
}
