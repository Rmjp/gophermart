package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	pb "gophermart/proto" // Importing the generated code

	"google.golang.org/grpc"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/google/uuid" // You need to go get this
	_ "github.com/lib/pq"    // Postgres driver
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not create driver: %w", err)
	}

	// Load files from the embedded filesystem
	d, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("could not create source: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs", d, "postgres", driver)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}

	// Run "Up" to apply all migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// ValidateOrder checks if the request is valid BEFORE we touch the DB.
// This is a "Pure Function" -> Input in, Error out. Perfect for Unit Testing.
func ValidateOrder(req *pb.CreateOrderRequest) error {
	if req.UserId == "" {
		return errors.New("user_id cannot be empty")
	}
	if len(req.Items) == 0 {
		return errors.New("order must have at least one item")
	}
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return errors.New("quantity must be positive")
		}
	}
	return nil
}

// Global DB connection
var db *sql.DB

type server struct {
	pb.UnimplementedOrderServiceServer
}

func (s *server) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	log.Printf("📦 Processing Order for User: %s", req.UserId)

	if err := ValidateOrder(req); err != nil {
		return nil, err // Return error to gRPC client
	}

	// 1. Generate ID
	orderID := uuid.New().String()

	// 2. Start Transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %v", err)
	}
	// Defer rollback in case of panic/error
	defer tx.Rollback()

	// 3. Insert Order
	_, err = tx.ExecContext(ctx,
		"INSERT INTO orders (order_id, user_id, status) VALUES ($1, $2, 'PENDING')",
		orderID, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to insert order: %v", err)
	}

	// 4. Insert Items
	for _, item := range req.Items {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO order_items (order_id, product_id, quantity) VALUES ($1, $2, $3)",
			orderID, item.ProductId, item.Quantity)
		if err != nil {
			return nil, fmt.Errorf("failed to insert item: %v", err)
		}
	}

	// 5. Commit
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %v", err)
	}

	log.Printf("✅ Order Saved: %s", orderID)

	// --- NEW: Publish to Kafka ---
	// We do this AFTER commit. If DB fails, we don't send the event.
	// Ideally, you use the "Outbox Pattern" for 100% safety, but this is fine for now.

	event := OrderCreatedEvent{
		OrderID:   orderID,
		UserID:    req.UserId,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Publish!
	go func() {
		// We use a separate context with its own timeout (e.g., 10s)
		// so a slow Kafka doesn't block the goroutine forever.
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := PublishOrderCreated(bgCtx, event)
		if err != nil {
			// CRITICAL LOG: We can't tell the user anymore (they are gone),
			// so we must log this so the SysAdmin sees it.
			log.Printf("❌ [Background] Failed to publish event for order %s: %v", orderID, err)
		} else {
			log.Printf("📢 [Background] Event published: %s", orderID)
		}
	}()

	return &pb.CreateOrderResponse{OrderId: orderID, Status: "PENDING"}, nil
}

func main() {

	// --- 1. INIT TRACER ---
	ctx := context.Background()
	// Service Name: "order-service"
	shutdown, err := InitTracer(ctx, "order-service", "jaeger:4317")
	if err != nil {
		log.Fatalf("Failed to init tracer: %v", err)
	}
	defer shutdown(ctx)

	// --- Init Database ---
	pgURL := os.Getenv("POSTGRES_URL")
	if pgURL == "" {
		pgURL = "postgres://gopher:gopherpass@postgres:5432/gophermart?sslmode=disable"
	}

	db, err = sql.Open("postgres", pgURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	// --- NEW: Run Migrations ---
	log.Println("🔄 Running Database Migrations...")
	if err := runMigrations(db); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}
	log.Println("✅ Migrations applied successfully!")

	// --- NEW: Init Kafka ---
	kafkaAddr := os.Getenv("KAFKA_ADDR")
	if kafkaAddr == "" {
		kafkaAddr = "kafka:9092" // K8s Service Name
	}
	InitKafka(kafkaAddr, "gophermart.orders.created.v1")
	defer CloseKafka()

	// 1. Listen on TCP port 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 2. Create gRPC Server
	s := grpc.NewServer(
		// This "Interceptor" automatically starts a span for every gRPC call
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	// 3. Register our implementation
	pb.RegisterOrderServiceServer(s, &server{})

	log.Printf("🚀 Order Service (gRPC + OTel) listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
