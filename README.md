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

## Debugging video and audio

### Playing video on host

```sh
# Run on 'streamer' container
$ gst-launch-1.0 -v ximagesrc display-name=:0 remote=1 use-damage=0 ! videoconvert ! rtpvrawpay ! udpsink host=host.docker.internal port=9999

# Run on host
$ gst-launch-1.0 -v udpsrc port=9999 caps="application/x-rtp, media=(string)video, sampling=(string)RGB, width=(string)800, height=(string)600" ! rtpvrawdepay ! autovideosink
```

### Playing audio on host

Make sure a PulseAudio daemon set up on your host and is listening on TCP `:4713`.

```sh
# Run on 'streamer' container
$ gst-launch-1.0 -v pulsesrc server=localhost:4713 ! queue ! pulsesink server=host.docker.internal:4713
```
