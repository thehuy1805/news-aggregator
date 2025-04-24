package kafka

import (
	"context"
	"encoding/json"

	"scraper-service/internal/scraper"

	"github.com/IBM/sarama"
)

func NewProducer(brokers []string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(brokers, config)
	return producer, err
}

func SendArticle(ctx context.Context, producer sarama.SyncProducer, topic string, article scraper.Article) error {
	data, err := json.Marshal(article)
	if err != nil {
		return err
	}
	_, _, err = producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(data),
	})
	return err
}
