package streamerserver

import (
	"net"
	"testing"
	"time"

	"github.com/castaneai/mashimaro/pkg/streamer/streamerproto"

	"github.com/castaneai/mashimaro/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestGstServer(t *testing.T) {
	lis := testutils.ListenTCPWithRandomPort(t)
	gs := newGstServer(lis)
	serveErr := make(chan error)
	go func() {
		serveErr <- gs.Serve("videotestsrc")
	}()
	conn, err := net.Dial("tcp", lis.Addr().String())
	assert.NoError(t, err)
	var sp streamerproto.SamplePacket
	assert.NoError(t, streamerproto.ReadSamplePacket(conn, &sp))
	assert.True(t, sp.Duration > 0)
	assert.True(t, len(sp.Data) > 0)
	conn.Close()
	time.Sleep(500 * time.Millisecond)
	gs.Stop()
	assert.Nil(t, <-serveErr)
}
