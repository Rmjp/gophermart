package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// Import our generated code
	pb "gophermart/proto"
)

var (
	rdb *redis.Client
	db  *sql.DB
	// Global gRPC Client
	orderClient pb.OrderServiceClient
)

func main() {
	// --- CONFIGURATION ---
	redisAddr := os.Getenv("REDIS_ADDR")
	pgURL := os.Getenv("POSTGRES_URL")
	// Default to localhost:50051 if not set (for local testing)
	orderServiceAddr := os.Getenv("ORDER_SERVICE_ADDR")
	if orderServiceAddr == "" {
		orderServiceAddr = "localhost:50051"
	}

	// --- 1. CONNECT TO REDIS & POSTGRES (Existing code) ---
	// (Skipping detailed error checks here for brevity, assume they work as before)
	rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
	db, _ = sql.Open("postgres", pgURL)

	// --- 2. CONNECT TO GRPC SERVICE ---
	// We use "WithTransportCredentials(insecure)" because we haven't set up TLS certificates yet.
	conn, err := grpc.NewClient(orderServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect to Order Service: %v", err)
	}
	defer conn.Close()

	// Initialize the client stub
	orderClient = pb.NewOrderServiceClient(conn)
	log.Printf("✅ Connected to Order Service at %s", orderServiceAddr)

	// --- 3. SETUP GIN ROUTER ---
	r := gin.Default()

	// Existing Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "up"})
	})

	// NEW: Create Order Endpoint
	r.POST("/orders", createOrderHandler)

	log.Println("🚀 API Gateway starting on :8080")
	r.Run(":8080")
}

// Handler to convert HTTP JSON -> gRPC
func createOrderHandler(c *gin.Context) {
	// A. Define the JSON structure we expect from the user
	type RequestBody struct {
		UserID    string `json:"user_id"`
		ProductID string `json:"product_id"`
		Quantity  int32  `json:"quantity"`
	}
	var body RequestBody

	// B. Parse JSON
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// C. Call gRPC Service
	// We create a context with a 1-second timeout so requests don't hang forever
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Construct the gRPC Request Message
	grpcReq := &pb.CreateOrderRequest{
		UserId: body.UserID,
		Items: []*pb.OrderItem{
			{ProductId: body.ProductID, Quantity: body.Quantity},
		},
	}

	// The Actual RPC Call
	resp, err := orderClient.CreateOrder(ctx, grpcReq)
	if err != nil {
		log.Printf("Error calling Order Service: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to place order"})
		return
	}

	// D. Return JSON Response
	c.JSON(http.StatusOK, gin.H{
		"order_id": resp.OrderId,
		"status":   resp.Status,
	})
}
