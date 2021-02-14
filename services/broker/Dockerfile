FROM golang:alpine as builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/broker services/broker/main.go

FROM alpine
WORKDIR /app
COPY --from=builder /bin/broker /app/broker
ENTRYPOINT ["/app/broker"]
