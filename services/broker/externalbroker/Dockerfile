FROM golang as builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/broker services/broker/externalbroker/main.go

FROM scratch
WORKDIR /app
COPY --from=builder /bin/broker /app/broker
ENTRYPOINT ["/app/broker"]
