FROM golang as builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/gameserver services/gameserver/gameserver/main.go

FROM alpine as certer
RUN apk update && apk add --no-cache ca-certificates && update-ca-certificates

FROM scratch
WORKDIR /app
COPY --from=certer /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/gameserver /app/gameserver
ENTRYPOINT ["/app/gameserver"]
