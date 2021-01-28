FROM golang:alpine as builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o /bin/gameagent cmd/gameagent/main.go

FROM alpine
WORKDIR /app
COPY --from=builder /bin/gameagent /app/gameagent
ENTRYPOINT ["/app/gameagent"]
