FROM ubuntu:20.04

RUN apt update -y \
	&& apt install -y libgstreamer1.0-0 gstreamer1.0-plugins-base gstreamer1.0-plugins-good gstreamer1.0-plugins-bad gstreamer1.0-plugins-ugly gstreamer1.0-libav gstreamer1.0-tools gstreamer1.0-x gstreamer1.0-alsa gstreamer1.0-gl gstreamer1.0-pulseaudio

ENTRYPOINT ["/bin/sh", "-c", "while :; do sleep 10; done"]