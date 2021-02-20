package streamer

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGStreamer(t *testing.T) {
	capturer, err := NewVideoTestCapturer()
	assert.NoError(t, err)
	assert.NoError(t, capturer.Start())
	chunk, err := capturer.ReadChunk(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, chunk)
	assert.True(t, len(chunk.Data) > 0)
}

func TestX11Capturer(t *testing.T) {
	x11conf := &X11CaptureConfig{
		Display: os.Getenv("DISPLAY"),
		Area:    &ScreenCaptureArea{0, 0, 100, 100},
	}
	x264conf := &X264EncodeConfig{}
	capturer, err := NewX11Capturer(x11conf, x264conf)
	assert.NoError(t, err)
	assert.NoError(t, capturer.Start())
	chunk, err := capturer.ReadChunk(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, chunk)
	assert.True(t, len(chunk.Data) > 0)
}
