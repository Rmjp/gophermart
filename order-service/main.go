package main

import (
	"context"
	"log"
	"net"

	pb "gophermart/proto" // Importing the generated code
	"google.golang.org/grpc"
)

// Server struct implements the generated OrderServiceServer interface
type server struct {
	pb.UnimplementedOrderServiceServer
}

// CreateOrder is the actual business logic
func (s *server) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	log.Printf("📦 Received Order from User: %s with %d items", req.UserId, len(req.Items))

	// TODO: Save to Postgres (We will add this later)
	
	// Mock Response
	return &pb.CreateOrderResponse{
		OrderId: "ord-12345-mock",
		Status:  "PENDING",
	}, nil
}

func main() {
	// 1. Listen on TCP port 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 2. Create gRPC Server
	s := grpc.NewServer()

	// 3. Register our implementation
	pb.RegisterOrderServiceServer(s, &server{})

	log.Printf("🚀 Order Service (gRPC) listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}