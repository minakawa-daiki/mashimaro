FROM golang as builder
WORKDIR /
RUN CGO_ENABLED=0 go get github.com/OpenAyame/ayame

FROM scratch
WORKDIR /app
COPY --from=builder /go/bin/ayame /app/ayame
COPY ./ayame_config.yaml /app/ayame.yaml
ENTRYPOINT ["/app/ayame"]
