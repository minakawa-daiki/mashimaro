FROM golang as builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/gameprocess services/gameserver/gameprocess/main.go

FROM ghcr.io/castaneai/wine:6.0-groovy
ENV DEBIAN_FRONTEND noninteractive
RUN apt update -y \
	&& apt install -y wget xvfb x11vnc
RUN wget http://winetricks.org/winetricks \
	&& chmod +x winetricks \
	&& mv winetricks /usr/bin/winetricks \
	&& winetricks wenquanyi
RUN apt install -y locales \
	&& locale-gen ja_JP.UTF-8
ENV WINEDLLOVERRIDES "mscoree=d;mshtml=d"
ENV LANG ja_JP.UTF-8
COPY ./services/gameserver/gameprocess/entrypoint.sh /entrypoint.sh
COPY --from=builder /bin/gameprocess /bin/gameprocess
ENTRYPOINT ["/entrypoint.sh", "/bin/gameprocess"]
