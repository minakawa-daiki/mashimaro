MINIKUBE_PROFILE := mashimaro

build:
	docker-compose build

up:
	minikube start -p $(MINIKUBE_PROFILE) --cpus=3 --memory=2500mb --driver=virtualbox
	minikube profile $(MINIKUBE_PROFILE)
	kubectl apply -f k8s/namespace.yaml # You need to create your namespaces before installing Agones.
	helm repo add agones https://agones.dev/chart/stable
	helm repo update
	helm install agones --set "gameservers.namespaces={mashimaro}" --namespace agones-system --create-namespace agones/agones

down:
	docker-compose down
	docker volume rm mashimaro_x11socket

bash:
	docker-compose exec streamer bash

generate:
	docker run --rm -v $(shell pwd):/app -w /app znly/protoc -I. --go_out=plugins=grpc:./pkg ./proto/*.proto

test:
	go test -race -count=1 ./...
