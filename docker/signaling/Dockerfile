FROM golang:alpine as builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o /bin/signaling cmd/signaling/main.go

FROM alpine
WORKDIR /app
COPY --from=builder /bin/signaling /app/signaling
COPY --from=builder /app/static /app/static
ENTRYPOINT ["/app/signaling"]
