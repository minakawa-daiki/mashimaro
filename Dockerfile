FROM golang as builder
WORKDIR /app
RUN apt update -y \
	&& apt install -y libgstreamer-plugins-base1.0-dev gstreamer1.0-plugins-base gstreamer1.0-plugins-good gstreamer1.0-plugins-bad gstreamer1.0-plugins-ugly gstreamer1.0-libav gstreamer1.0-tools gstreamer1.0-x gstreamer1.0-alsa gstreamer1.0-gl gstreamer1.0-pulseaudio
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o /bin/mashimaro ./cmd/mashimaro/main.go

FROM ubuntu:20.04

RUN apt update -y \
	&& apt install -y libgstreamer1.0-0 gstreamer1.0-plugins-base gstreamer1.0-plugins-good gstreamer1.0-plugins-bad gstreamer1.0-plugins-ugly gstreamer1.0-libav gstreamer1.0-tools gstreamer1.0-x gstreamer1.0-alsa gstreamer1.0-gl gstreamer1.0-pulseaudio

RUN apt install -y pulseaudio

WORKDIR /app

COPY docker/streamer/default.pa /etc/pulse/default.pa
COPY docker/streamer/entrypoint.sh /entrypoint.sh
COPY --from=builder /bin/mashimaro /app/mashimaro
COPY --from=builder /app/static /app/static

ENTRYPOINT ["/entrypoint.sh", "/app/mashimaro"]
