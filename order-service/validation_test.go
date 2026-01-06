package main

import (
	"testing"

	pb "gophermart/proto"

	"github.com/stretchr/testify/assert"
)

func TestValidateOrder(t *testing.T) {
	// 1. Define the Test Cases (The Table)
	tests := []struct {
		name        string
		req         *pb.CreateOrderRequest
		expectError bool
		errMsg      string
	}{
		{
			name: "Happy Path - Valid Order",
			req: &pb.CreateOrderRequest{
				UserId: "user-123",
				Items: []*pb.OrderItem{
					{ProductId: "prod-1", Quantity: 1},
				},
			},
			expectError: false,
		},
		{
			name: "Fail - Missing User ID",
			req: &pb.CreateOrderRequest{
				UserId: "", // Empty!
				Items: []*pb.OrderItem{
					{ProductId: "prod-1", Quantity: 1},
				},
			},
			expectError: true,
			errMsg:      "user_id cannot be empty",
		},
		{
			name: "Fail - No Items",
			req: &pb.CreateOrderRequest{
				UserId: "user-123",
				Items:  []*pb.OrderItem{}, // Empty!
			},
			expectError: true,
			errMsg:      "order must have at least one item",
		},
		{
			name: "Fail - Negative Quantity",
			req: &pb.CreateOrderRequest{
				UserId: "user-123",
				Items: []*pb.OrderItem{
					{ProductId: "prod-1", Quantity: -5}, // Invalid!
				},
			},
			expectError: true,
			errMsg:      "quantity must be positive",
		},
	}

	// 2. Loop through the table
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function
			err := ValidateOrder(tt.req)

			// 3. Assertions using Testify
			if tt.expectError {
				assert.Error(t, err)                    // We expect an error
				assert.Equal(t, tt.errMsg, err.Error()) // Verify the message
			} else {
				assert.NoError(t, err) // We expect NO error
			}
		})
	}
}
