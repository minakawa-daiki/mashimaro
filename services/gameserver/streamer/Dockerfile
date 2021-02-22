FROM ubuntu:20.10 as builder

# Use GCE apt servers
ARG GCE_ZONE=asia-northeast1
RUN cp /etc/apt/sources.list /etc/apt/sources.list.orig && \
    sed -i "s/archive.ubuntu.com/${GCE_ZONE}.gce.archive.ubuntu.com/g" /etc/apt/sources.list

ENV DEBIAN_FRONTEND noninteractive
RUN apt update -y \
	&& apt install -y wget tar gcc libgstreamer-plugins-base1.0-dev gstreamer1.0-plugins-base
ENV PATH $PATH:/usr/local/go/bin
RUN wget https://golang.org/dl/go1.16.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.16.linux-amd64.tar.gz
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /bin/streamer services/gameserver/streamer/main.go

FROM ubuntu:20.10

# Use GCE apt servers
ARG GCE_ZONE=asia-northeast1
RUN cp /etc/apt/sources.list /etc/apt/sources.list.orig && \
    sed -i "s/archive.ubuntu.com/${GCE_ZONE}.gce.archive.ubuntu.com/g" /etc/apt/sources.list

RUN apt update -y \
	&& apt install -y libgstreamer1.0-0 gstreamer1.0-plugins-base gstreamer1.0-plugins-good gstreamer1.0-plugins-bad gstreamer1.0-plugins-ugly gstreamer1.0-libav gstreamer1.0-tools gstreamer1.0-x gstreamer1.0-alsa gstreamer1.0-gl gstreamer1.0-pulseaudio libxrandr2 libxtst6 libxfixes3
WORKDIR /app
COPY --from=builder /bin/streamer /app/streamer
COPY ./services/gameserver/streamer/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh", "/app/streamer"]
