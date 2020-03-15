gen:
	protoc -I ../proto streamer.proto --go_out=plugins=grpc:.

up:
	go run cmd/main.go
	open http://localhost:8080