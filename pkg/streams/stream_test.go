package streams_test

import (
	"testing"

	"github.com/castaneai/mashimaro/pkg/streams"

	"github.com/stretchr/testify/assert"
)

func TestGstMediaStream_ReadChunk(t *testing.T) {
	stream, err := streams.GetVideoTestStream()
	assert.NoError(t, err)
	chunk, err := stream.ReadChunk()
	assert.NoError(t, err)
	assert.NotNil(t, chunk)
	assert.True(t, len(chunk.Data) > 0)
}
