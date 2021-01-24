MINIKUBE_PROFILE := mashimaro

up:
	minikube start -p $(MINIKUBE_PROFILE) --kubernetes-version v1.17.13
	minikube profile $(MINIKUBE_PROFILE)
	minikube kubectl -- apply -f k8s/namespace.yaml # You need to create your namespaces before installing Agones.
	helm repo add agones https://agones.dev/chart/stable
	helm repo update
	helm upgrade --install agones  --namespace agones-system --create-namespace \
		--set "gameservers.namespaces={mashimaro}" \
		--set "agones.allocator.generateTLS=false" \
		--set "agones.allocator.disableMTLS=true" \
		--set "agones.allocator.disableTLS=true" \
		agones/agones

run:
	skaffold run --minikube-profile=$(MINIKUBE_PROFILE) --port-forward --tail

down:
	minikube delete -p $(MINIKUBE_PROFILE)

generate:
	docker run --rm -v $(shell pwd):/app -w /app znly/protoc -I. --go_out=plugins=grpc:./pkg ./proto/*.proto

test:
	go test -race -count=1 ./...
