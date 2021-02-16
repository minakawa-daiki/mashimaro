package streamer

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
