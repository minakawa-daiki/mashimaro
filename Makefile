up:
	docker-compose up -d

down:
	docker-compose down
	docker volume rm mashimaro_x11socket

bash:
	docker-compose exec streamer bash

generate:
	docker run --rm -v $(shell pwd):/app -w /app znly/protoc -I. --go_out=plugins=grpc:./pkg ./proto/*.proto

