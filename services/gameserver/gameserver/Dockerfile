FROM ubuntu:20.10 as builder
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
RUN CGO_ENABLED=1 go build -o /bin/gameserver services/gameserver/gameserver/main.go

FROM ubuntu:20.10
RUN apt update -y \
	&& apt install -y libgstreamer1.0-0 gstreamer1.0-plugins-base gstreamer1.0-plugins-good gstreamer1.0-plugins-bad gstreamer1.0-plugins-ugly gstreamer1.0-libav gstreamer1.0-tools gstreamer1.0-x gstreamer1.0-alsa gstreamer1.0-gl gstreamer1.0-pulseaudio libxrandr2 libxtst6 libxfixes3
WORKDIR /app
COPY --from=builder /bin/gameserver /app/gameserver
COPY ./services/gameserver/gameserver/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh", "/app/gameserver"]
