package gameserver

import (
	"fmt"
	"math"
)

type GstPipeliner interface {
	CompileGstPipeline() (string, error)
}

type X11ScreenCapturer struct {
	display    string
	screenRect *ScreenRect
}

func NewX11ScreenCapturer(captureDisplay string, screenRect *ScreenRect) *X11ScreenCapturer {
	return &X11ScreenCapturer{display: captureDisplay, screenRect: screenRect}
}

func (c *X11ScreenCapturer) CompileGstPipeline() (string, error) {
	c.screenRect.FixForH264()
	startX := c.screenRect.StartX
	if startX < 0 {
		startX = 0
	}
	startY := c.screenRect.StartY
	if startY < 0 {
		startY = 0
	}
	endX := c.screenRect.EndX - 1
	if endX < 0 {
		endX = 0
	}
	endY := c.screenRect.EndY - 1
	if endY < 0 {
		endY = 0
	}
	// why use-damage=0?: https://github.com/GoogleCloudPlatform/selkies-vdi/blob/0da21b7c9432bd5c99f1f9f7c541ac9c583f9ef4/images/gst-webrtc-app/gstwebrtc_app.py#L148
	return fmt.Sprintf("ximagesrc display-name=%s remote=1 use-damage=0 startx=%d starty=%d endx=%d endy=%d ! video/x-raw,framerate=60/1",
		c.display, startX, startY, endX, endY), nil
}

type ScreenRect struct {
	StartX int
	StartY int
	EndX   int
	EndY   int
}

func (a *ScreenRect) String() string {
	return fmt.Sprintf("ScreenRect(%dx%d)", a.Width(), a.Height())
}

func (a *ScreenRect) Width() int {
	return a.EndX - a.StartX
}

func (a *ScreenRect) Height() int {
	return a.EndY - a.StartY
}

func (a *ScreenRect) IsValid() bool {
	return (a.StartX >= 0 && a.StartY >= 0 && a.EndX >= 0 && a.EndY >= 0) &&
		a.Width() >= 16 && a.Height() >= 16 // x264enc's min width/height is 16 https://gstreamer.freedesktop.org/documentation/x264/index.html?gi-language=c#sink
}

func (a *ScreenRect) FixForH264() {
	// H264 requirement is that video dimensions are divisible by 2.
	// ref: https://github.com/hzbd/kazam/blob/491869ac29860a19254fa8c226f75314a7eee83d/kazam/backend/gstreamer.py#L128
	if int(math.Abs(float64(a.StartX-a.EndX)))%2 != 0 {
		a.EndX -= 1
		if a.EndX < 0 {
			a.EndX = 0
		}
	}
	if int(math.Abs(float64(a.StartY-a.EndY)))%2 != 0 {
		a.EndY -= 1
		if a.EndY < 0 {
			a.EndY = 0
		}
	}
}

type X264Encoder struct {
	srcPipeline  GstPipeliner
	encodeParams string
}

func NewX264Encoder(srcPipeline GstPipeliner, encodeParams string) *X264Encoder {
	return &X264Encoder{srcPipeline: srcPipeline, encodeParams: encodeParams}
}

func (e *X264Encoder) CompileGstPipeline() (string, error) {
	src, err := e.srcPipeline.CompileGstPipeline()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s ! videoconvert ! video/x-raw,format=I420 ! x264enc %s", src, e.encodeParams), nil
}

type PulseAudioCapturer struct {
	PulseServer string
}

func NewPulseAudioCapturer(pulseServer string) *PulseAudioCapturer {
	return &PulseAudioCapturer{
		PulseServer: pulseServer,
	}
}

func (c *PulseAudioCapturer) CompileGstPipeline() (string, error) {
	// TODO: `provide-clock=1` causes stuttering, but the reason is still unknown to me. For now, I set it to 0 and it works fine.
	return fmt.Sprintf("pulsesrc server=%s provide-clock=0", c.PulseServer), nil
}

type OpusEncoder struct {
	srcPipeline GstPipeliner
}

func NewOpusEncoder(srcPipeline GstPipeliner) *OpusEncoder {
	return &OpusEncoder{
		srcPipeline: srcPipeline,
	}
}

func (e *OpusEncoder) CompileGstPipeline() (string, error) {
	src, err := e.srcPipeline.CompileGstPipeline()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s ! opusenc", src), nil
}
