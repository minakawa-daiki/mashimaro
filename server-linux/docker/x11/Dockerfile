FROM ubuntu:20.04

RUN apt update -y \
	&& apt install -y xserver-xorg-video-dummy x11-apps imagemagick

COPY xorg.conf /etc/xorg.conf

ENTRYPOINT ["/usr/bin/X", "-config", "/etc/xorg.conf", ":0"]
