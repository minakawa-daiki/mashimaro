# mashimaro

A simple PoC of cloud gaming.

## Usage

```sh
docker-compose up -d
open http://localhost:8080
```

## Debugging with VNC

```sh
open vnc://localhost:5900
```

## Getting video and audios via gstreamer

```sh
# setup containers
docker-compose up -d

## stream video

# on streamer container
$ gst-launch-1.0 -v ximagesrc display-name=:0 ! videoconvert ! rtpvrawpay ! udpsink host=host.docker.internal port=9999

# on host
$ gst-launch-1.0 -v udpsrc port=9999 caps="application/x-rtp, media=(string)video, sampling=(string)RGB, width=(string)800, height=(string)600" ! rtpvrawdepay ! autovideosink


## stream audio
## Make sure that pulseaudio daemon set up on your host and is listening on TCP :4713

# on streamer container
$ gst-launch-1.0 -v pulsesrc server=localhost:4713 ! queue ! pulsesink server=host.docker.internal:4713
```
