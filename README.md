# mashimaro

A simple PoC of cloud gaming.

## Usage

```sh
make up
open http://localhost:8080

# tear down
make down
```

## Debugging on local (without Docker)

- Requires [GStreamer](https://gstreamer.freedesktop.org/) and [Node.js](https://nodejs.org/ja/)

```sh
# Start streamer
USE_TEST_MEDIA_SOURCE=1 go run cmd/streamer/main.go

# Start signaling server
STREAMER_ADDR=localhost:50501 go run cmd/signaling/main.go

# Open web client
npx live-server ./static
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

## Testing 

```sh
make test
```

