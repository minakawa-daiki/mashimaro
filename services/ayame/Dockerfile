FROM golang:alpine as builder
WORKDIR /
RUN apk add --no-cache git
RUN go get github.com/OpenAyame/ayame

FROM alpine
WORKDIR /app
COPY --from=builder /go/bin/ayame /app/ayame
COPY ./ayame_config.yaml /app/ayame.yaml
ENTRYPOINT ["/app/ayame"]
