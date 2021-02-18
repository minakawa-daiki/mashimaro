FROM ubuntu:20.04
RUN apt update -y \
	&& apt install -y wget
RUN wget https://github.com/OpenAyame/ayame/releases/download/2020.1.4/ayame_linux_amd64.gz \
    && gunzip ayame_linux_amd64.gz && chmod +x ayame_linux_amd64 && mkdir -p /app && mv ayame_linux_amd64 /app/ayame
WORKDIR /app
COPY ./ayame_config.yaml /app/ayame.yaml
ENTRYPOINT ["/app/ayame"]
