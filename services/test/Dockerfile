FROM ghcr.io/castaneai/wine:6.0-groovy
ENV WINEDLLOVERRIDES "mscoree=d;mshtml=d"

ARG GOVERSION=1.16
ENV DEBIAN_FRONTEND noninteractive
RUN apt update -y \
	&& apt install -y wget tar gcc x11-apps
ENV PATH $PATH:/usr/local/go/bin
RUN wget https://golang.org/dl/go${GOVERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GOVERSION}.linux-amd64.tar.gz

RUN apt install -y libgstreamer1.0-0 \
    libgstreamer1.0-dev \
    gstreamer1.0-plugins-base libgstreamer-plugins-base1.0-dev \
    gstreamer1.0-plugins-good \
    gstreamer1.0-plugins-bad \
    gstreamer1.0-plugins-ugly \
    gstreamer1.0-x

WORKDIR /app
COPY services/test/entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
