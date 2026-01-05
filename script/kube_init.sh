minikube start --driver=podman

# 1. Create the namespace first
kubectl apply -f k8s/00-namespace.yaml

# 2. Apply the database and redis
kubectl apply -f k8s/01-postgres.yaml
kubectl apply -f k8s/02-redis.yaml

# 3. Apply the API Gateway (It will fail to pull image, which is expected!)
kubectl apply -f k8s/10-api-gateway.yaml