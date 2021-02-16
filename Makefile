MINIKUBE_PROFILE := mashimaro
KUBERNETES_VERSION := 1.17.13
AGONES_VERSION := 1.12.0

up:
	minikube start -p $(MINIKUBE_PROFILE) --kubernetes-version $(KUBERNETES_VERSION) --mount-string "$(realpath .)/games:/games" --mount
	minikube profile $(MINIKUBE_PROFILE)
	minikube kubectl -- apply -f services/namespace.yaml
	helm repo add agones https://agones.dev/chart/stable
	helm repo update
	helm upgrade --install agones --version $(AGONES_VERSION)  --namespace agones-system --create-namespace \
		--set "gameservers.namespaces={mashimaro}" \
		--set "agones.allocator.generateTLS=false" \
		--set "agones.allocator.disableMTLS=true" \
		--set "agones.allocator.disableTLS=true" \
		agones/agones

web:
	npx live-server ./client

run:
	skaffold run --minikube-profile=$(MINIKUBE_PROFILE) --port-forward

down:
	minikube stop -p $(MINIKUBE_PROFILE)

delete:
	minikube delete -p $(MINIKUBE_PROFILE)

generate:
	docker run --rm -v $(shell pwd):/app -w /app znly/protoc -I. --go_out=plugins=grpc:./pkg ./proto/*.proto

test:
	docker-compose up -d ayame
	go test -v -race -count=1 ./...
