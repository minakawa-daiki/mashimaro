package streamer_test

import (
	"testing"

	"github.com/castaneai/mashimaro/pkg/streamer"
	"github.com/stretchr/testify/assert"
)

func TestGstMediaStream_ReadChunk(t *testing.T) {
	stream, err := streamer.NewVideoTestStream()
	assert.NoError(t, err)
	stream.Start()
	chunk, err := stream.ReadChunk()
	assert.NoError(t, err)
	assert.NotNil(t, chunk)
	assert.True(t, len(chunk.Data) > 0)
}
