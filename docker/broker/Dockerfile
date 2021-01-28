FROM golang:alpine as builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o /bin/broker cmd/broker/main.go

FROM alpine
WORKDIR /app
COPY --from=builder /bin/broker /app/broker
ENTRYPOINT ["/app/broker"]
