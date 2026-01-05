package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq" // Postgres driver
)

var (
	rdb *redis.Client
	db  *sql.DB
)

func main() {
	// 1. Get Configuration from Env (Defined in K8s)
	redisAddr := os.Getenv("REDIS_ADDR")
	pgURL := os.Getenv("POSTGRES_URL")

	if redisAddr == "" || pgURL == "" {
		log.Fatal("Missing required environment variables: REDIS_ADDR or POSTGRES_URL")
	}

	// 2. Initialize Redis
	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
	} else {
		log.Println("✅ Connected to Redis!")
	}

	// 3. Initialize Postgres
	var err error
	db, err = sql.Open("postgres", pgURL+"?sslmode=disable")
	if err != nil {
		log.Fatalf("Error opening DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Printf("Warning: Failed to connect to Postgres: %v", err)
	} else {
		log.Println("✅ Connected to Postgres!")
	}

	// 4. Setup Gin Router
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "up",
			"redis":    rdb.Ping(c).Err() == nil,
			"postgres": db.Ping() == nil,
		})
	})

	log.Println("🚀 API Gateway starting on :8080")
	r.Run(":8080")
}