package streamer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	defaultX264Params = "speed-preset=ultrafast tune=zerolatency byte-stream=true intra-refresh=true"
)

func TestGstMediaStream_ReadChunk(t *testing.T) {
	stream, err := NewVideoTestStream()
	assert.NoError(t, err)
	stream.Start()
	chunk, err := stream.ReadChunk()
	assert.NoError(t, err)
	assert.NotNil(t, chunk)
	assert.True(t, len(chunk.Data) > 0)
}

func TestXImageSrc(t *testing.T) {
	stream, err := NewX11VideoStream(":2", defaultX264Params)
	assert.NoError(t, err)
	stream.Start()
	chunk, err := stream.ReadChunk()
	assert.NoError(t, err)
	assert.NotNil(t, chunk)
	assert.True(t, len(chunk.Data) > 0)
	select {}
}
