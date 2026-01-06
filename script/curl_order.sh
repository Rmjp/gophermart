curl -X POST http://localhost:50001/orders \
     -H "Content-Type: application/json" \
     -d '{"user_id": "user-99", "product_id": "laptop", "quantity": 1}'