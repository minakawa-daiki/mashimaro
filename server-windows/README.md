# server-windows

WIP

## Receiving RTPFrameWriter via GStreamer

Sakura.exe example

```sh
$ export caps="application/x-rtp, media=(string)video, clock-rate=(int)90000, encoding-name=(string)RAW, sampling=(string)BGRA, depth=(string)8, width=(string)1282, height=(string)747, colorimetry=(string)BT601-5, payload=(int)127, ssrc=(uint)0, timestamp-offset=(uint)0, seqnum-offset=(uint)0"

$ gst-launch-1.0 -v udpsrc port=9999 caps="$caps" ! rtpvrawdepay ! autovideosink
```

