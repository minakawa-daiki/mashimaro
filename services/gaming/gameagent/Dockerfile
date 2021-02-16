FROM golang as builder
WORKDIR /app
RUN apt update -y \
	&& apt install -y libgstreamer-plugins-base1.0-dev gstreamer1.0-plugins-base libxrandr-dev libxtst-dev
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /bin/gameagent services/gaming/gameagent/main.go

FROM ubuntu:20.04
RUN apt update -y \
	&& apt install -y libgstreamer1.0-0 gstreamer1.0-plugins-base gstreamer1.0-plugins-good gstreamer1.0-plugins-bad gstreamer1.0-plugins-ugly gstreamer1.0-libav gstreamer1.0-tools gstreamer1.0-x gstreamer1.0-alsa gstreamer1.0-gl gstreamer1.0-pulseaudio libxrandr2 libxtst6
WORKDIR /app
COPY --from=builder /bin/gameagent /app/gameagent
COPY ./services/gaming/gameagent/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh", "/app/gameagent"]
