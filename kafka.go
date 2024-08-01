package main

import (
	"audio-extraction-job/env"
	"context"
	"crypto/tls"
	"log"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

func PublishProd(topic string, message []byte) error {
	eventHubNamespace := env.GetInstance().KafkaEventHubNamespace
	connectionString := env.GetInstance().KafkaEventHubConnectionString
	username := "$ConnectionString"

	// Set up SASL configuration
	mechanism := plain.Mechanism{
		Username: username,
		Password: connectionString,
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(eventHubNamespace + ".servicebus.windows.net:9093"),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
		Transport: &kafka.Transport{
			SASL: mechanism,
			TLS:  &tls.Config{},
		},
	}
	err := writer.WriteMessages(context.Background(),
		kafka.Message{
			Value: message,
		},
	)

	if err != nil {
		log.Println(err.Error())
	}

	return err
}
