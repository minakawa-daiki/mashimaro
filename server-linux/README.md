# server-linux

## Screen Capture x11

```sh
# setup containers
docker-compose up -d

make capture
```

## GStreamer example

```sh
# setup containers
docker-compose up -d

# on streamer container
$ docker-compose exec streamer bash
$ gst-launch-1.0 -v ximagesrc display-name=:0 ! videoconvert ! rtpvrawpay ! udpsink host=host.docker.internal port=9999

# on host
$ gst-launch-1.0 -v udpsrc port=9999 caps="application/x-rtp, media=(string)video, sampling=(string)RGB, width=(string)1920, height=(string)1080" ! rtpvrawdepay ! autovideosink
```
