# apply kube deployment
Build image before

If use minikube load it with `podman save <image_name> | minikube image load -`

```sh
kubectl apply -f ./k8s/
```

# rollout (restart)
```sh
kubectl rollout -n <namespace> deployment <name>
```

## postgres check order
```
kubectl exec -it -n gophermart-dev deployment/postgres -- psql -U gopher -d gophermart -c "SELECT * FROM orders;"
```