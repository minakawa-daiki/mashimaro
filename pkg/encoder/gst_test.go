package encoder

import (
	"net"
	"testing"
	"time"

	"github.com/castaneai/mashimaro/pkg/encoder/encoderproto"

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
	var sp encoderproto.SamplePacket
	assert.NoError(t, encoderproto.ReadSamplePacket(conn, &sp))
	assert.True(t, sp.Duration > 0)
	assert.True(t, len(sp.Data) > 0)
	conn.Close()
	time.Sleep(100 * time.Millisecond)
	gs.Stop()
	assert.Nil(t, <-serveErr)
}
