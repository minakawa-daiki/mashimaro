FROM ubuntu:20.04
RUN apt update -y && apt install -y pulseaudio
COPY default.pa /etc/pulse/default.pa
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
