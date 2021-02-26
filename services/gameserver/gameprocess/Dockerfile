FROM golang as builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/gameprocess services/gameserver/gameprocess/main.go

FROM ghcr.io/castaneai/wine:6.0-groovy

# Use GCE apt servers
ARG GCE_ZONE=asia-northeast1
RUN cp /etc/apt/sources.list /etc/apt/sources.list.orig && \
    sed -i "s/archive.ubuntu.com/${GCE_ZONE}.gce.archive.ubuntu.com/g" /etc/apt/sources.list

ENV DEBIAN_FRONTEND noninteractive
RUN apt update -y \
	&& apt install -y wget
RUN wget http://winetricks.org/winetricks \
	&& chmod +x winetricks \
	&& mv winetricks /usr/bin/winetricks \
	&& winetricks wenquanyi
RUN apt install -y locales tzdata \
	&& locale-gen ja_JP.UTF-8
ENV WINEDLLOVERRIDES "mscoree=d;mshtml=d"
ENV LANG ja_JP.UTF-8
ENV TZ Asia/Tokyo
COPY ./services/gameserver/gameprocess/entrypoint.sh /entrypoint.sh
COPY --from=builder /bin/gameprocess /bin/gameprocess
ENTRYPOINT ["/entrypoint.sh", "/bin/gameprocess"]
