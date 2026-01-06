package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The structure of data we want to save
type OrderEvent struct {
	OrderID   string `json:"order_id" bson:"order_id"`
	UserID    string `json:"user_id" bson:"user_id"`
	Timestamp string `json:"timestamp" bson:"timestamp"`
	// Mongo will add its own _id
}

func main() {
	// 1. Connect to MongoDB
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://admin:password@mongodb:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	collection := client.Database("analytics").Collection("order_events")
	log.Println("✅ Connected to MongoDB")

	// 2. Connect to Kafka (Consumer)
	kafkaAddr := os.Getenv("KAFKA_ADDR")
	if kafkaAddr == "" {
		kafkaAddr = "kafka:9092"
	}
	topic := "gophermart.orders.created.v1"

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaAddr},
		Topic:    topic,
		GroupID:  "analytics-service", // Important for load balancing!
		MinBytes: 10e3,                // 10KB
		MaxBytes: 10e6,                // 10MB
	})
	defer reader.Close()

	log.Println("🎧 Analytics Service listening for events...")

	// 3. The Consumer Loop
	for {
		// ReadMessage blocks until a new message arrives
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("❌ Error reading message: %v", err)
			continue
		}

		// Parse JSON
		var event OrderEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("❌ Failed to parse JSON: %v", err)
			continue
		}

		// Save to MongoDB
		// We create a new context for the DB write
		writeCtx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = collection.InsertOne(writeCtx, event)
		if err != nil {
			log.Printf("❌ Failed to save to Mongo: %v", err)
		} else {
			log.Printf("💾 Event saved to Mongo: %s (Partition: %d)", event.OrderID, m.Partition)
		}
	}
}
