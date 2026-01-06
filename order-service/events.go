package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

// Global Writer (Producer)
var kafkaWriter *kafka.Writer

// Initialize the Kafka Writer
func InitKafka(kafkaAddr string, topic string) {
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaAddr),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{}, // Distributes messages evenly
	}
	log.Printf("✅ Kafka Producer initialized for topic: %s", topic)
}

// Close the connection
func CloseKafka() {
	if kafkaWriter != nil {
		kafkaWriter.Close()
	}
}

// The Event Structure (What we send)
type OrderCreatedEvent struct {
	OrderID   string `json:"order_id"`
	UserID    string `json:"user_id"`
	Timestamp string `json:"timestamp"`
}

// Publish the event
func PublishOrderCreated(ctx context.Context, event OrderCreatedEvent) error {
	// 1. Convert struct to JSON
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// 2. Write to Kafka
	// We use the Key (UserID) to ensure orders from the same user
	// always go to the same Kafka partition (order guarantee).
	err = kafkaWriter.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(event.UserID),
			Value: payload,
		},
	)
	if err != nil {
		return err
	}

	return nil
}
